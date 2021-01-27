package agent

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/config"
	"code.xxxxx.cn/platform/galaxy/pkg/agent/syncer"
	"code.xxxxx.cn/platform/galaxy/pkg/agent/updater"
	"code.xxxxx.cn/platform/galaxy/pkg/agent/vm"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/filedb"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/runtime"
)

// Agent hold every thing of training agent
type Agent struct {
	config     *config.AgentConfiguration
	kubeClient clientset.Interface
	informers  informers.SharedInformerFactory
	conn       conns.GlobalConn
	cmdClient  cmd.Client
	cmdServer  cmd.Server
	grpcServer *grpc.Server
	fdb        *filedb.FileDB
	vm         *vm.VersionManager
	fm         *updater.FileMapper
	updateHub  *updater.UpdateHub
	syncer     *syncer.Syncer

	podLister corelisters.PodLister

	InformerSynced       chan struct{}
	ClusterStatusSynced  bool
	afterSendRegisterCmd chan struct{}
	// Close this to shut down the world.
	stopEverything <-chan struct{}
}

// New build a agent
func New(cfg *config.AgentConfiguration,
	client clientset.Interface,
	informers informers.SharedInformerFactory,
	stopCh <-chan struct{}) (*Agent, error) {

	clientCfg, err := conns.NewClientConfig(cfg.ManagerAddr, cfg.ManagerCert, conns.ManagerAddrDN)
	if err != nil {
		return nil, err
	}
	conn := conns.NewGlobalConn(clientCfg)

	connectID := strings.Join([]string{cfg.ID, strconv.Itoa(int(time.Now().Unix()))}, apis.ConnectionSplit)
	cmdClient := cmd.NewCmdClient(connectID, conn, stopCh)
	cmdServer := cmd.NewCmdServer(stopCh)

	grpcServer := newGRPCServer()
	fdb, err := filedb.NewFileDB(&filedb.Config{Workdir: cfg.WorkDir, CasDisable: true})
	if err != nil {
		return nil, err
	}

	fileMapper := updater.NewFileMapper(fdb)
	appSyncer := syncer.NewSyncer(syncer.Config{CmdServer: cmdServer})
	updateHub := updater.NewUpdateHub(updater.Config{FDB: fdb, FM: fileMapper, SiteID: cfg.ID, Syncer: appSyncer})

	vmins := vm.NewVersionManager(vm.Config{
		Fdb:             fdb,
		SiteID:          cfg.ID,
		PmpSecret:       cfg.PMPSecret,
		ProviderAddress: cfg.ManagerProviderAddress,
		UpdateHub:       updateHub,
	})

	ca := &Agent{
		config:     cfg,
		kubeClient: client,
		informers:  informers,
		conn:       conn,
		cmdClient:  cmdClient,
		cmdServer:  cmdServer,
		grpcServer: grpcServer,
		fdb:        fdb,
		vm:         vmins,
		fm:         fileMapper,
		updateHub:  updateHub,
		syncer:     appSyncer,

		podLister: informers.Core().V1().Pods().Lister(),

		InformerSynced:       make(chan struct{}),
		afterSendRegisterCmd: make(chan struct{}),
		stopEverything:       stopCh,
	}

	cmdServer.GetConnManager().OnReady(func(conn *conns.Conn) {
		files := map[string]struct{}{}
		for _, file := range strings.Split(conn.Info["files"], ",") {
			if file == "" {
				continue
			}
			files[file] = struct{}{}
		}

		ca.syncer.RegisterApp(&syncer.AppDescribe{
			AppID:      conn.Info["appName"],
			Hostname:   conn.Info["hostname"],
			PodIP:      conn.Info["ip"],
			Namespaces: strings.Split(conn.Info["namespaces"], ","),
			Files:      files,
		}, conn.Key)
		alog.Infof("Registered App of %q, info: %v", conn.Info["appName"], conn.Info)
	})

	cmdServer.GetConnManager().OnNotReady(func(conn *conns.Conn) {
		app := appSyncer.GetAppByConn(conn.Key)
		if app == nil {
			alog.Infof("%v could not get app when cancel", conn.Key)
			return
		}
		alog.Infof("App:%v cancel, hostname: %v", app.AppID, app.Hostname)

		appSyncer.CancelApp(conn.Key)
		_, err := cmdClient.SendSync(cmdClient.NewCmdReq("appcancel", cmd.Args{
			"site":     cfg.ID,
			"app":      app.AppID,
			"hostname": app.Hostname,
			"ip":       app.PodIP,
		}), DefaultServerTimeout)
		if err != nil {
			alog.Errorf(" Cancel failed: %v ", err)
		}
	})

	ca.cmdClient.OnReady(func() {
		alog.Info("Agent to server is Ready!!!")
		ca.syncLocalData()

		for _, app := range ca.syncer.GetAllAppsCopy() {
			filesMap := map[string]map[string]string{}
			for filename := range app.Files {
				s := strings.Split(filename, "/")
				if len(s) < 2 {
					alog.Infof("file :%v is not right format when report", filename)
					continue
				}
				ns := s[0]
				name := s[1]
				if _, ok := filesMap[ns]; !ok {
					filesMap[ns] = map[string]string{}
				}
				// fill current file digest from file mapper
				filesMap[ns][name] = ca.fm.Get(ns, name)
			}

			for ns, files := range filesMap {
				ca.reportUpdatedInfoToRemote(app, ns, files)
			}
		}
	})

	go ca.syncClusterStatus()
	ca.addCmdHandlers()
	ca.RegisterService()

	return ca, nil
}

// Run start training agent
func (ca *Agent) Run() {
	defer runtime.HandleCrash()
	alog.Infof("Starting galaxy agent")
	defer alog.Infof("Shutting down galaxy agent")

	// start to connect manager server
	alog.Infof("Starting connect to galaxy server")
	// will block until got a active conn
	ca.conn.PollConn()

	// start command tunnel of manager
	go ca.updateHub.Start()
	go ca.vm.Start()
	go ca.fm.Run()
	go ca.syncer.Run()
	go ca.cmdClient.StartListen(ca.afterSendRegisterCmd)
	go ca.StartListen()

	<-ca.stopEverything
}

func (ca *Agent) syncLocalData() {
	err := ca.refreshAgentFDBFromRemote(ca.fm.GetAll(), true)
	for err != nil {
		alog.Errorf("refresh failed: %v", err)
		time.Sleep(time.Second)
		err = ca.refreshAgentFDBFromRemote(ca.fm.GetAll(), true)
	}

	local := ca.refreshFilemapperFromLocal()
	if len(local) != 0 {
		alog.Infof("Local Disk Scan Over, exist %v ns not correct, start refresh", len(local))
		err := ca.refreshAgentFDBFromRemote(local, false)
		for err != nil {
			alog.Errorf("refresh failed: %v", err)
			time.Sleep(time.Second)
			err = ca.refreshAgentFDBFromRemote(local, false)
		}
	} else {
		alog.Info("Local Disk Scan Over, no inconsistency file mapper")
	}
}

// RegisterService register service
func (ca *Agent) RegisterService() {
	pb.RegisterCmdManagerServer(ca.grpcServer, ca.cmdServer)
}

// StartListen start listen kinds of service
func (ca *Agent) StartListen() {
	go ca.ListenGRPC()
}

// ListenGRPC start grpc service
func (ca *Agent) ListenGRPC() {
	lis, err := net.Listen("tcp", ":"+ca.config.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	alog.Infof("GRPC server is listening at: %s", ":"+ca.config.GRPCPort)

	alog.Fatal(ca.grpcServer.Serve(lis))
}

func newGRPCServer() *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(conns.NewConnManager()),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}

	alog.V(4).Infof("Use insecure grpc server without tls")

	return grpc.NewServer(opts...)
}

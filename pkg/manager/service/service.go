package service

import (
	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/application"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/auth"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/cluster"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/config"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/configserver"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/log"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/notifier"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/provider"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/service/rest"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/emicklei/go-restful"
	"google.golang.org/grpc"
)

// Manager defined a manager to hold all services
type Manager struct {
	config          *config.ManagerConfiguration
	grpcServer      *grpc.Server
	webServer       *restful.Container
	cmdServer       cmd.Server
	logManager      log.Manager
	notifierManager notifier.Manager
	authManager     auth.Manager
	providerManager provider.Manager
	notifier        notifier.Manager
	configManager   configserver.Manager
	appManager      application.Manager
	clusterManager  cluster.Manager
	stopWorld       <-chan struct{}
}

// NewServiceManager build a manager to control all services
func NewServiceManager(cfg *config.ManagerConfiguration, grpcServer *grpc.Server,
	webServer *restful.Container, cs cmd.Server, authManager auth.Manager, stopCh <-chan struct{}) *Manager {

	notifierManager := notifier.NewManager()
	logManager := log.NewManager(cfg.ESURL, cs)
	configManager, fdb := configserver.NewConfigManager(cfg.DataDir, cs, stopCh)
	providerManager := provider.NewProvider(fdb, stopCh)
	appManager := application.NewManager(cs, stopCh)
	clusterManager := cluster.NewManager(cs, cfg.ResourceManagerAPI, cfg.ResourceManagerSecret, stopCh)

	return &Manager{
		config:          cfg,
		grpcServer:      grpcServer,
		webServer:       webServer,
		cmdServer:       cs,
		authManager:     authManager,
		logManager:      logManager,
		configManager:   configManager,
		providerManager: providerManager,
		appManager:      appManager,
		clusterManager:  clusterManager,
		notifier:        notifierManager,
		stopWorld:       stopCh,
	}
}

// registerGRPCServices register all grpc services
func (sm *Manager) registerGRPCServices() {
	// TODO add more grpc servers
	pb.RegisterCmdManagerServer(sm.grpcServer, sm.cmdServer)
}

// registerWebServices register all web services
func (sm *Manager) registerWebServices() {
	// TODO add more restful api services
	sm.webServer.Add(rest.NewConfigService(sm.configManager).RestfulService())
	sm.webServer.Add(rest.NewAppService(sm.appManager).RestfulService())
	sm.webServer.Add(rest.NewClusterService(sm.clusterManager).RestfulService())
	sm.webServer.Add(rest.NewLogService(sm.cmdServer).RestfulService())
	sm.webServer.Add(rest.NewProviderService(sm.providerManager).RestfulService())

	// add error handler
	sm.webServer.ServiceErrorHandler(func(serviceError restful.ServiceError, request *restful.Request, response *restful.Response) {
		if err := response.WriteAsJson(apis.NewRespErr(apis.ErrSvcFailed, serviceError)); err != nil {
			alog.Errorf("RespAPI error: %v", err)
		}
	})

	// add swagger service finally
	if sm.config.SwaggerEnable {
		sm.webServer.Add(rest.NewSwaggerService(sm.webServer.RegisteredWebServices()).RestfulService())
	}

	// excluded urls to auth
	sm.authManager.Excluded(apis.SwaggerAPIDocsPath)
}

// RegisterServices register all services, include grpc services, restful services, websocket services
func (sm *Manager) RegisterServices() {
	sm.registerGRPCServices()
	sm.registerWebServices()
}

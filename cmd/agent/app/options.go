package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/config"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	refutil "code.xxxxx.cn/platform/galaxy/pkg/util/ref"
	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// Options contains everything necessary to create and run a server.
type Options struct {
	configFile string
	config     *config.AgentConfiguration
}

// Configuration holds everything needed by kubernetes
type Configuration struct {
	// Config contains all params for agent
	Config *config.AgentConfiguration
	// Client is a k8s client
	Client clientset.Interface
	// InformerFactory build all kinds of k8s informers
	InformerFactory informers.SharedInformerFactory
	// LeaderElection is optional.
	LeaderElection *leaderelection.LeaderElectionConfig
}

// NewOptions build option instance
func NewOptions() *Options {
	return &Options{
		config: config.NewDefaultConfig(),
	}
}

// AddFlags add cmd flags
func (opt *Options) AddFlags(fs *pflag.FlagSet) {
	hideFlags()

	fs.StringVar(&opt.configFile, "config", opt.configFile, "configuration file path")
	fs.StringVar(&opt.config.ID, "id", opt.config.ID, "uniq id report to manager")
	fs.StringVar(&opt.config.WorkDir, "work-dir", opt.config.WorkDir, "work dir")
	fs.StringVar(&opt.config.ManagerAddr, "manager-addr", opt.config.ManagerAddr, "manager server address")
	fs.StringVar(&opt.config.PMPSecret, "pmp-secret", opt.config.PMPSecret, "pmp secret")
	fs.StringVar(&opt.config.ManagerProviderAddress, "manager-provider-addr", opt.config.ManagerAddr, "manager provider address")
	fs.StringVar(&opt.config.ManagerCert, "manager-cert", opt.config.ManagerCert, "manager server cert, must pkcs file")
	fs.DurationVar(&opt.config.CacheTTL, "ttl", opt.config.CacheTTL, "cache ttl, device keepalive timeout, ")
}

// Config build all configuration
func (opt *Options) Config() (*Configuration, error) {
	cfg, err := loadConfigFromFile(opt.configFile)
	if err != nil {
		alog.Errorf("load config from file failed: %v", err)
		return nil, err
	}

	// flags overwrite cfg
	if cfg, err = opt.completeConfig(cfg); err != nil {
		return nil, err
	}
	// validate cfg
	if err := opt.validate(cfg); err != nil {
		return nil, err
	}

	// create k8s client and leader elect client
	kubeClient, elClient, err := opt.createClients(cfg)
	if err != nil {
		alog.Errorf("create clients failed: %v", err)
		return nil, err
	}

	// Set up leader election if enabled.
	var leaderElectionConfig *leaderelection.LeaderElectionConfig
	if cfg.LeaderElection.LeaderElect {
		leaderElectionConfig, err = makeLeaderElectionConfig(cfg, elClient)
		if err != nil {
			alog.Errorf("make leader election config failed: %v", err)
			return nil, err
		}
	}

	return &Configuration{
		Config:          cfg,
		Client:          kubeClient,
		InformerFactory: informers.NewSharedInformerFactory(kubeClient, 0),
		LeaderElection:  leaderElectionConfig,
	}, nil
}

// createClients create a k8s client by leader election or not
func (opt *Options) createClients(cfg *config.AgentConfiguration) (kubeclient clientset.Interface, leaderElectionClient clientset.Interface, err error) {
	if len(cfg.KubeConnection.Kubeconfig) == 0 && len(cfg.KubeMaster) == 0 {
		alog.Warningf("kubeconfig and master both are empty, connect apiserver in cluster.")
	}
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.KubeConnection.Kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: cfg.KubeMaster}}).ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	cfg.KubeMaster = kubeConfig.Host

	kubeConfig.AcceptContentTypes = cfg.KubeConnection.AcceptContentTypes
	kubeConfig.ContentType = cfg.KubeConnection.ContentType
	kubeConfig.QPS = cfg.KubeConnection.QPS
	kubeConfig.Burst = cfg.KubeConnection.Burst

	kubeclient, err = clientset.NewForConfig(restclient.AddUserAgent(kubeConfig, config.AgentName))
	if err != nil {
		return nil, nil, err
	}

	restConfig := *kubeConfig
	restConfig.Timeout = cfg.LeaderElection.RenewDeadline
	leaderElectionClient, err = clientset.NewForConfig(restclient.AddUserAgent(&restConfig, "leader-election"))
	if err != nil {
		return nil, nil, err
	}

	return kubeclient, leaderElectionClient, err
}

// makeLeaderElectionConfig build config for leader election
func makeLeaderElectionConfig(cfg *config.AgentConfiguration, client clientset.Interface) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}
	id := hostname + "_" + string(uuid.NewUUID())
	rl, err := resourcelock.New(cfg.LeaderElection.ResourceLock,
		cfg.LeaderElection.ResourceNamespace,
		cfg.LeaderElection.ResourceName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		})
	if err != nil {
		return nil, fmt.Errorf("couldn't create resource lock: %v", err)
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: cfg.LeaderElection.LeaseDuration,
		RenewDeadline: cfg.LeaderElection.RenewDeadline,
		RetryPeriod:   cfg.LeaderElection.RetryPeriod,
		WatchDog:      leaderelection.NewLeaderHealthzAdaptor(time.Second * 20),
		Name:          config.AgentName,
	}, nil
}

// loadConfigFromFile load config from file
func loadConfigFromFile(file string) (*config.AgentConfiguration, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := &config.AgentConfiguration{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		alog.Errorf("Cluster Agent Config file convert failed: %v", err)
		return nil, err
	}

	return cfg, nil
}

// completeConfig complete build all args: input args > load from file > default args
func (opt *Options) completeConfig(cfg *config.AgentConfiguration) (*config.AgentConfiguration, error) {
	def := config.NewDefaultConfig()
	input, err := refutil.Diff(opt.config, def)
	if err != nil {
		return nil, fmt.Errorf("diff input args complete configuration failed: %v", err)
	}
	load, err := refutil.Diff(cfg, def)
	if err != nil {
		return nil, fmt.Errorf("diff file config complete configuration failed: %v", err)
	}
	du, err := refutil.Union(load, input)
	if err != nil {
		return nil, fmt.Errorf("union diffs complete configuration failed: %v", err)
	}
	c, err := refutil.Union(def, du)
	if err != nil || c == nil {
		return nil, fmt.Errorf("union default complete configuration failed: %v", err)
	}
	data, _ := json.Marshal(c)
	alog.V(4).Infof("Complete config: %v", string(data))
	return c.(*config.AgentConfiguration), nil
}

// validate check if all needed flags are set
func (opt *Options) validate(cfg *config.AgentConfiguration) error {
	if cfg == nil {
		return fmt.Errorf("config can't be nil")
	}
	if len(cfg.ID) == 0 {
		return fmt.Errorf("id can't be blank")
	}
	if len(cfg.ManagerAddr) == 0 {
		return fmt.Errorf("manager address can't be blank")
	}
	return nil
}

// hideFlags hide all flags not needed
func hideFlags() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	_ = pflag.Set("logtostderr", "true")
	_ = pflag.CommandLine.MarkHidden("log-flush-frequency")
	_ = pflag.CommandLine.MarkHidden("alsologtostderr")
	_ = pflag.CommandLine.MarkHidden("log_backtrace_at")
	_ = pflag.CommandLine.MarkHidden("log_file")
	_ = pflag.CommandLine.MarkHidden("logtostderr")
	_ = pflag.CommandLine.MarkHidden("stderrthreshold")
	_ = pflag.CommandLine.MarkHidden("vmodule")
	_ = pflag.CommandLine.MarkHidden("skip_headers")
}

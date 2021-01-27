package config

import (
	"time"
)

/* Some default value of config fields */
const (
	AgentName                    = "training-agent"
	DefaultLockObjectNamespace   = "manage"
	DefaultLockObjectName        = "training-agent"
	DefaultWorkDir               = "/etc/galaxy"
	DefaultCacheTTL              = 120 * time.Second
	DefaultHealthzBindAddr       = "0.0.0.0:50066"
	DefaultMetricsBindAddr       = "0.0.0.0:50077"
	DefaultKubeConnContentType   = "application/vnd.kubernetes.protobuf"
	DefaultKubeConnectionQPS     = 50
	DefaultKubeConnectionBurst   = 100
	DefaultGRPCListenPort        = "8090"
	DefaultResourceLock          = "endpoints"
	DefaultElectionLeaseDuration = 15 * time.Second
	DefaultElectionRenewDeadline = 10 * time.Second
	DefaultElectionRetryPeriod   = 2 * time.Second
	DefaultCacheCleanPeriod      = 10 * time.Second
)

// AgentConfiguration is the config file for agent
type AgentConfiguration struct {
	// ID is uniq id of the agent
	ID string `yaml:"id,omitempty"`
	// WorkDir device agent work dir
	WorkDir string `yaml:"workDir,omitempty"`
	// ManagerAddr is the domain address and port of manager server to connect, eg. rm.xxxxx.cn:50051
	ManagerAddr string `yaml:"managerAddr,omitempty"`
	// ManagerCert is the cert file path provide to manager server connection
	ManagerCert string `yaml:"managerCert,omitempty"`
	// CacheTTL is the expire time of cache
	CacheTTL         time.Duration `yaml:"cacheTTL,omitempty"`
	CacheCleanPeriod time.Duration `yaml:"cacheCleanPeriod,omitempty"`

	// PMPSecret provide path to connect with pmp for download
	PMPSecret string `yaml:"pmpSecret,omitempty"`
	// Port grpc listen port
	GRPCPort string `yaml:"grpcPort"`

	// ManagerProviderAddr
	ManagerProviderAddress string `yaml:"managerProviderAddr"`

	// LeaderElection defines the configuration of leader election client.
	LeaderElection LeaderElectionConfig `yaml:"leaderElection,omitempty"`
	// KubeConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	KubeConnection ClientConnectionConfig `yaml:"kubeConnection,omitempty"`
	KubeMaster     string                 `yaml:"kubeMaster,omitempty"`

	// HealthzBindAddr is the IP address and port for the health check server to serve on,
	// defaulting to 0.0.0.0:60052
	HealthzBindAddr string `yaml:"healthzBindAddr,omitempty"`
	// MetricsBindAddr is the IP address and port for the metrics server to
	// serve on, defaulting to 0.0.0.0:60052.
	MetricsBindAddr string `yaml:"metricsBindAddr,omitempty"`
}

// LeaderElectionConfig is config for leader election
type LeaderElectionConfig struct {
	LeaderElect       bool          `yaml:"leaderElect"`
	LeaseDuration     time.Duration `yaml:"leaseDuration"`
	RenewDeadline     time.Duration `yaml:"renewDeadline"`
	RetryPeriod       time.Duration `yaml:"retryPeriod"`
	ResourceLock      string        `yaml:"resourceLock"`
	ResourceName      string        `yaml:"resourceName"`
	ResourceNamespace string        `yaml:"resourceNamespace"`
}

// ClientConnectionConfig is config for kubernetes connection
type ClientConnectionConfig struct {
	Kubeconfig         string  `yaml:"kubeconfig"`
	Server             string  `yaml:"server"`
	AcceptContentTypes string  `yaml:"acceptContentTypes"`
	ContentType        string  `yaml:"contentType"`
	QPS                float32 `yaml:"qps"`
	Burst              int     `yaml:"burst"`
}

// NewDefaultConfig build a default configuration of agent
func NewDefaultConfig() *AgentConfiguration {
	return &AgentConfiguration{
		WorkDir:  DefaultWorkDir,
		CacheTTL: DefaultCacheTTL,
		LeaderElection: LeaderElectionConfig{
			LeaseDuration: DefaultElectionLeaseDuration,
			RenewDeadline: DefaultElectionRenewDeadline,
			RetryPeriod:   DefaultElectionRetryPeriod,

			ResourceLock:      DefaultResourceLock,
			ResourceNamespace: DefaultLockObjectNamespace,
			ResourceName:      DefaultLockObjectName,
		},
		KubeConnection: ClientConnectionConfig{
			AcceptContentTypes: "",
			ContentType:        DefaultKubeConnContentType,
			QPS:                DefaultKubeConnectionQPS,
			Burst:              DefaultKubeConnectionBurst,
		},
		GRPCPort: DefaultGRPCListenPort,

		HealthzBindAddr: DefaultHealthzBindAddr,
		MetricsBindAddr: DefaultMetricsBindAddr,

		CacheCleanPeriod: DefaultCacheCleanPeriod,
	}
}

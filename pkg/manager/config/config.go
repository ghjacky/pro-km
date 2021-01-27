package config

import "time"

/* all default params of manager */
const (
	DefaultGRPCServeAddr   = "0.0.0.0:50070"
	DefaultWebServeAddr    = "0.0.0.0:50080"
	DefaultMetricServeAddr = "0.0.0.0:50090"
	DefaultCertFilePath    = "/etc/galaxy/pki/tls.crt"
	DefaultKeyFilePath     = "/etc/galaxy/pki/tls.key"
	DefaultCAFilePath      = "/etc/galaxy/pki/ca.crt"
	DefaultDBConfigPath    = "/etc/galaxy/config/db.yaml"
	DefaultAuthSkip        = false
	DefaultAuthTokenTTL    = 30 * time.Minute
	DefaultPrometheusAddr  = "http://prometheus.monitoring:9090"
	DefaultDataDir         = "/data"
)

// ManagerConfiguration holds whole configuration of server
type ManagerConfiguration struct {
	// WorkDir work dir
	WorkDir string
	// DataDir dir to store configs etc.
	DataDir string
	// GRPCServerAddr rpc server bind address: ip:port
	GRPCServerAddr string
	// GRPCInsecure if use insecure rpc connection
	GRPCInsecure bool
	// WebServerAddr web api server address: ip:port
	WebServerAddr string
	// MetricServerAddr metric server address: ip:port
	MetricServerAddr string
	// WebInsecure if use insecure http web api server
	WebInsecure bool
	// CertFile cert file path for rpc server and api server
	CertFile string
	// KeyFile key file path for secure rpc server and api server
	KeyFile string
	// CAFile ca file path for secure rpc server and api server
	CAFile string
	// DBConfig database config file
	DBConfig string
	// AuthAddr system auth access address: scheme://ip:port
	AuthAddr string
	// PMPSecret access token to pmp
	PMPSecret string
	// AuthSkip if skip authorization for all apis
	AuthSkip bool
	// AuthClientID client id to access system auth
	AuthClientID int64
	// AuthClientSecret client secret to access system auth
	AuthClientSecret string
	// AuthTokenTTL the expire time of token for accessing system auth
	AuthTokenTTL time.Duration
	// ESURL the url of elastic search engine
	ESURL string
	// PrometheusAddr address of prometheus at cloud
	PrometheusAddr string
	// API address of  ResourceManager at cloud
	ResourceManagerAPI string
	// API secret of  ResourceManager at cloud
	ResourceManagerSecret string
	// SwaggerEnable if enable swagger to open swagger api docs
	SwaggerEnable bool
}

// NewDefaultConfig build default manager configuration
func NewDefaultConfig() *ManagerConfiguration {
	return &ManagerConfiguration{
		GRPCServerAddr:   DefaultGRPCServeAddr,
		WebServerAddr:    DefaultWebServeAddr,
		MetricServerAddr: DefaultMetricServeAddr,
		CertFile:         DefaultCertFilePath,
		KeyFile:          DefaultKeyFilePath,
		CAFile:           DefaultCAFilePath,
		DBConfig:         DefaultDBConfigPath,
		AuthSkip:         DefaultAuthSkip,
		AuthTokenTTL:     DefaultAuthTokenTTL,
		PrometheusAddr:   DefaultPrometheusAddr,
	}
}

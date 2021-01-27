package app

import (
	"flag"
	"fmt"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/config"
	"github.com/spf13/pflag"
)

// Options contains anything necessary of cmd to create and run a server.
type Options struct {
	config *config.ManagerConfiguration
}

// NewOptions build a default Option instance
func NewOptions() *Options {
	return &Options{
		config: config.NewDefaultConfig(),
	}
}

// AddFlags add all flags of cmd
func (opt *Options) AddFlags(fs *pflag.FlagSet) {
	hideFlags()
	fs.StringVar(&opt.config.WorkDir, "work-dir", "/etc/galaxy", "work dir")
	fs.StringVar(&opt.config.DataDir, "storage-dir", config.DefaultDataDir, "data dir")
	fs.StringVar(&opt.config.GRPCServerAddr, "rpc-addr", opt.config.GRPCServerAddr, "rpc server bind address")
	fs.BoolVar(&opt.config.GRPCInsecure, "rpc-insecure", opt.config.GRPCInsecure, "if use insecure grpc server without tls")
	fs.StringVar(&opt.config.WebServerAddr, "web-addr", opt.config.WebServerAddr, "web server bind address")
	fs.StringVar(&opt.config.MetricServerAddr, "metric-addr", opt.config.MetricServerAddr, "metric server bind address")
	fs.BoolVar(&opt.config.WebInsecure, "web-insecure", opt.config.WebInsecure, "if true use http web server instead of https")
	fs.StringVar(&opt.config.CertFile, "cert", opt.config.CertFile, "server cert file path")
	fs.StringVar(&opt.config.KeyFile, "key", opt.config.KeyFile, "server key file path")
	fs.StringVar(&opt.config.CAFile, "ca", opt.config.CAFile, "server ca file path")
	fs.StringVar(&opt.config.DBConfig, "db", opt.config.DBConfig, "database config file path")

	fs.StringVar(&opt.config.AuthAddr, "auth-addr", opt.config.AuthAddr, "auth platform address")
	fs.Int64Var(&opt.config.AuthClientID, "auth-client-id", opt.config.AuthClientID, "client-id to access auth")
	fs.StringVar(&opt.config.AuthClientSecret, "auth-client-secret", opt.config.AuthClientSecret, "client-secret to access auth")
	fs.DurationVar(&opt.config.AuthTokenTTL, "auth-token-ttl", opt.config.AuthTokenTTL, "auth token expire time")
	fs.StringVar(&opt.config.PMPSecret, "pmp-secret", opt.config.PMPSecret, "client-secret to access pmp")
	fs.BoolVar(&opt.config.AuthSkip, "auth-skip", opt.config.AuthSkip, "skip auth user")
	fs.BoolVar(&opt.config.SwaggerEnable, "swagger-enable", opt.config.SwaggerEnable, "enable swagger api docs")
	fs.StringVar(&opt.config.ESURL, "es-url", opt.config.ESURL, "es url")
	fs.StringVar(&opt.config.PrometheusAddr, "prom-url", opt.config.PrometheusAddr, "prometheus url for monitoring self metrics")
	fs.StringVar(&opt.config.ResourceManagerAPI, "resource-manager-api", opt.config.ResourceManagerAPI, "api address of resource managerment system")
	fs.StringVar(&opt.config.ResourceManagerSecret, "resource-manager-secret", opt.config.ResourceManagerSecret, "api secret of resource managerment system")
}

// hideFlags hide some help cmdline flags
func hideFlags() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	_ = pflag.Set("logtostderr", "true")
	_ = pflag.CommandLine.MarkHidden("log-flush-frequency")
	_ = pflag.CommandLine.MarkHidden("alsologtostderr")
	_ = pflag.CommandLine.MarkHidden("log_backtrace_at")
	_ = pflag.CommandLine.MarkHidden("log_dir")
	//_ = pflag.CommandLine.MarkHidden("log_file")
	_ = pflag.CommandLine.MarkHidden("logtostderr")
	_ = pflag.CommandLine.MarkHidden("stderrthreshold")
	_ = pflag.CommandLine.MarkHidden("vmodule")
	_ = pflag.CommandLine.MarkHidden("skip_headers")
}

// validate check necessary config
func (opt *Options) validate() error {

	if opt.config.AuthAddr == "" {
		return fmt.Errorf("ERROR: flag --auth-addr can't be empty")
	}

	if opt.config.AuthClientID == 0 {
		return fmt.Errorf("ERROR: flag --auth-client-id can't be empty")
	}

	if opt.config.AuthClientSecret == "" {
		return fmt.Errorf("ERROR: flag --auth-client-secret can't be empty")
	}

	if opt.config.PMPSecret == "" {
		return fmt.Errorf("ERROR: flag --pmp-secret can't be empty")
	}
	return nil
}

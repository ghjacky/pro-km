package server

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/auth"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/config"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/metrics"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/service"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/emicklei/go-restful"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server hold all to build a server provide services, contains grpc services, restful services, and websocket services
type Server struct {
	// config is all config info of server
	config *config.ManagerConfiguration
	// grpcServer provide grpc services
	grpcServer *grpc.Server
	// webServer provide restful services
	webServer *restful.Container
	// authManager provide authorization and authentication
	authManager auth.Manager
	// serviceManager
	serviceManager *service.Manager
	// stopWorld
	stopWorld <-chan struct{}
}

// NewServer init a server to listen and serve
func NewServer(cfg *config.ManagerConfiguration, stopCh <-chan struct{}) *Server {
	webServer := restful.NewContainer()
	cmdServer := cmd.NewCmdServer(stopCh)
	grpcServer := newGRPCServer(cfg.GRPCInsecure, cfg.CertFile, cfg.KeyFile, cfg.CAFile, cmdServer.GetConnManager())
	tokenManager := auth.NewJwtTokenManager(cfg.PMPSecret, cfg.AuthTokenTTL)
	authSDK := auth.NewAuthSDK(cfg.AuthClientID, cfg.AuthClientSecret, cfg.AuthAddr)
	authManager := auth.NewAuthManager(webServer, tokenManager, authSDK, cfg.AuthSkip)
	serviceManager := service.NewServiceManager(cfg, grpcServer, webServer, cmdServer, authManager, stopCh)
	server := &Server{
		config:         cfg,
		grpcServer:     grpcServer,
		webServer:      webServer,
		authManager:    authManager,
		serviceManager: serviceManager,
		stopWorld:      stopCh,
	}

	// register grpc servers
	server.registerServices()
	// register filters
	server.registerFilters()

	// TODO: start build and upload all resources after register all services
	go authManager.UploadAuthRolesAndResources()

	return server
}

// StartManagerServer initialize a http, grpc server to response request, and start metric server
func (s *Server) StartManagerServer() {
	go s.startGRPCServer(s.config.GRPCServerAddr)
	go s.startWebServer(s.config.WebServerAddr, s.config.WebInsecure, s.config.CertFile, s.config.KeyFile)
	go s.startMetricServer(s.config.MetricServerAddr)
}

func newGRPCServer(insecure bool, certFile, keyFile, caFile string, connManager *conns.ConnManager) *grpc.Server {
	// FIXME NOW EXPERIMENTAL， make configurable?
	//var kaep = keepalive.EnforcementPolicy{
	//	MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
	//	PermitWithoutStream: true,            // Allow pings even when there are no active streams
	//}
	//
	//var kasp = keepalive.ServerParameters{
	//	MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
	//	MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
	//	MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
	//	Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
	//	Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	//}

	opts := []grpc.ServerOption{
		grpc.StatsHandler(connManager),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		//grpc.KeepaliveEnforcementPolicy(kaep),
		//grpc.KeepaliveParams(kasp),
	}
	if insecure {
		alog.V(4).Infof("Use insecure grpc server without tls")
	} else {
		alog.V(4).Infof("Use secure grpc server with tls")
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			alog.Fatalf("Load keys and certs failed: %s", err)
		}

		tlsConfig := &tls.Config{
			ClientAuth:         tls.RequireAndVerifyClientCert,
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: false,
			Rand:               rand.Reader,
			ServerName:         conns.ManagerAddrDN,
		}

		if caFile != "" {
			certPool := x509.NewCertPool()
			caCrt, err := ioutil.ReadFile(caFile)
			if err != nil {
				alog.Fatalf("Read ca file failed: %v", err)
			}
			certPool.AppendCertsFromPEM(caCrt)
			tlsConfig.ClientCAs = certPool
		}

		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
	}

	return grpc.NewServer(opts...)
}

func (s *Server) startWebServer(addr string, insecure bool, certFile, keyFile string) {
	alog.Infof("Web server is listening at: %s, use tls: %t", addr, !insecure)
	if insecure {
		alog.Fatal(http.ListenAndServe(addr, s))
	} else {
		alog.Fatal(http.ListenAndServeTLS(addr, certFile, keyFile, s))
	}
}

func (s *Server) startGRPCServer(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	alog.Infof("GRPC server is listening at: %s", addr)
	alog.Fatal(s.grpcServer.Serve(lis))
}

func (s *Server) startMetricServer(addr string) {
	alog.Infof("Metric server is listening at: %s", addr)
	// ADD prometheus metrics handlers
	grpc_prometheus.Register(s.grpcServer)

	// ADD custom metrics handlers
	http.Handle("/metrics", metrics.NewHandler())

	alog.Fatal(http.ListenAndServe(addr, nil))
}

// registerFilters add all filters to restful apis
func (s *Server) registerFilters() {
	// Add cross filter
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-MAYA-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      s.webServer}
	s.webServer.Filter(cors.Filter)
	// Add container filter to respond to OPTIONS
	s.webServer.Filter(s.webServer.OPTIONSFilter)
	// Add filter to log every request
	s.webServer.Filter(func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		alog.V(4).Infof("%s -> %s %s Header=%v Form=%v", request.Request.RemoteAddr, request.Request.Method,
			request.Request.RequestURI, request.Request.Header, request.Request.Form)
		chain.ProcessFilter(request, response)
	})

	// Add Authorization filter
	s.webServer.Filter(s.authManager.AuthFilter)
}

// registerServices register all services, includes grpc services, web services, ws services
func (s *Server) registerServices() {
	s.serviceManager.RegisterServices()
}

// restful api handler
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// monitor http requests
	method, path := req.Method, req.URL.Path

	metrics.HTTPRequests.WithLabelValues(method, path).Inc()
	metrics.HTTPInflightRequests.WithLabelValues(method, path).Inc()
	defer metrics.HTTPInflightRequests.WithLabelValues(method, path).Dec()

	startTime := time.Now()
	defer metrics.HTTPRequestsDuration.WithLabelValues(method, path).Observe(metrics.SinceInSeconds(startTime))

	s.webServer.ServeHTTP(w, req)
}

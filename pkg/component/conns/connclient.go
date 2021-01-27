package conns

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	// ManagerAddrDN is domain name of manager server
	ManagerAddrDN         = "m.xxxxx.cn"
	defaultMonitorPeriod  = 15 * time.Second
	defaultConnectTimeout = 10 * time.Second
	defaultP12Password    = "xxxxx@2020^infra"
)

type globalClientConn struct {
	lock        sync.RWMutex
	clientConn  *grpc.ClientConn
	readyCh     chan struct{}
	clientQueue []*ClientConfig
}

// ClientConfig is configuration for global client connection
type ClientConfig struct {
	Addr       string
	Cert       string
	ServerName string
	Creds      credentials.TransportCredentials
}

// GlobalConn define connection actions of client
type GlobalConn interface {
	// PollConn block goroutine to get active conn
	PollConn()
	GetConn() *grpc.ClientConn
	ReConn(cli *ClientConfig) error
	CloseConn() error
	IsActive() bool

	// ConnMonitor
	ConnMonitor(stopCh <-chan struct{})
	RegisterMonitorConn(addr, certFile, serverName string) error
	AddClientQueue(addr, certFile, serverName string) error
	ConnOnStates(stats ...connectivity.State) <-chan struct{}
	ConnOnReady() <-chan struct{}

	GetCmdManagerClient() pb.CmdManagerClient
	GetCmdManagerClientExecuteStream() (pb.CmdManager_ExecuteClient, error)
}

// NewGlobalConn build a GlobalConn
func NewGlobalConn(queue ...*ClientConfig) GlobalConn {
	return &globalClientConn{
		clientQueue: queue,
	}
}

// NewClientConfig create a ClientConfig
func NewClientConfig(addr, certFile, serverName string) (*ClientConfig, error) {
	cert, certPool, err := ReadPKCS12(certFile)
	if err != nil {
		return nil, fmt.Errorf("read p12 file failed: %v", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
		RootCAs:            certPool,
		ServerName:         serverName,
	})
	return &ClientConfig{
		Addr:       addr,
		Cert:       certFile,
		ServerName: serverName,
		Creds:      creds,
	}, nil
}

// ReadPKCS12 read and parse p12 file to get certificates
func ReadPKCS12(p12File string) (tls.Certificate, *x509.CertPool, error) {
	data, err := ioutil.ReadFile(p12File)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	key, _, caCerts, err := pkcs12.DecodeChain(data, defaultP12Password)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("decode chain to get private key failed: %v", err)
	}
	if err := key.(*rsa.PrivateKey).Validate(); err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("private key validata failed: %v", err)
	}
	certPool := x509.NewCertPool()
	for _, caCrt := range caCerts {
		certPool.AddCert(caCrt)
	}
	blocks, err := pkcs12.ToPEM(data, defaultP12Password)
	if err != nil {
		return tls.Certificate{}, &x509.CertPool{}, fmt.Errorf("error while converting to PEM: %s", err)
	}
	var pemData []byte
	for _, b := range blocks {
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	cert, err := tls.X509KeyPair(pemData, pemData)
	if err != nil {
		return tls.Certificate{}, &x509.CertPool{}, fmt.Errorf("err while converting to key pair: %v", err)
	}

	return cert, certPool, nil
}

// RegisterMonitorConn register a monitor for grcp connection of addr, if connection error will auto reconnect
func (c *globalClientConn) RegisterMonitorConn(addr, certFile, serverName string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, cl := range c.clientQueue {
		if cl.Addr == addr {
			return nil
		}
	}

	cc, err := NewClientConfig(addr, certFile, serverName)
	if err != nil {
		return err
	}
	var queue []*ClientConfig
	queue = append(queue, cc)
	queue = append(queue, c.clientQueue...)
	c.clientQueue = queue
	return nil
}

// AddClientQueue add client config to queue to switch connection orderly
func (c *globalClientConn) AddClientQueue(addr, certFile, serverName string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, cl := range c.clientQueue {
		if cl.Addr == addr {
			return nil
		}
	}
	cc, err := NewClientConfig(addr, certFile, serverName)
	if err != nil {
		return err
	}
	c.clientQueue = append(c.clientQueue, cc)
	return nil
}

// GetMonitorConn get the first client config of client monitored
func (c *globalClientConn) GetMonitorConn() *ClientConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if len(c.clientQueue) == 0 {
		return nil
	}
	return c.clientQueue[0]
}

// GetClientQueue get client queue
func (c *globalClientConn) GetClientQueue() []*ClientConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.clientQueue
}

// PollConn poll conn
func (c *globalClientConn) PollConn() {
	alog.Infof("Wait for active connection...")
	if c.IsActive() {
		alog.Infof("Current connection is active, no need poll conn")
		return
	}
	if err := c.CloseConn(); err != nil {
		alog.Errorf("Close connection %s failed: %v", c.GetConn().Target(), err)
	}
	for {
		for _, cli := range c.GetClientQueue() {
			conn, err := dialGRPCConn(cli)
			if err == nil {
				alog.Infof("Connected to %s", cli.Addr)
				c.SetConn(conn)
				close(c.readyCh)
				return
			}
			alog.Errorf("Connect to %s failed: %v", cli.Addr, err)
		}
	}
}

func (c *globalClientConn) ConnMonitor(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		t := time.NewTimer(defaultMonitorPeriod)
		monitorCli := c.GetMonitorConn() // monitoring the first conn
		alog.V(4).Infof("==> Monitor connection: %s, current: %s", monitorCli.Addr, c.GetConn().Target())
		oldConn := c.GetConn()
		if oldConn.Target() != monitorCli.Addr {
			// if is not the first client, reconnect and switch conn
			if err := c.ReConn(monitorCli); err != nil {
				// reconnect failed
				alog.Errorf("==> Failed connection: %s: %v", monitorCli.Addr, err)
				continue
			} else {
				// reconnect succeed, close old connection
				if err := oldConn.Close(); err != nil {
					alog.Errorf("==> Close old connection %s failed: %v", oldConn.Target(), err)
				}
				alog.V(4).Infof("==> Switched connection: %s -> %s", oldConn.Target(), monitorCli.Addr)
			}
		} else {
			alog.V(4).Infof("==> Connection %s is active, no need to reconnect", monitorCli.Addr)
		}

		select {
		case <-stopCh:
			return
		case <-t.C:
		}
	}
}

func (c *globalClientConn) GetCmdManagerClient() pb.CmdManagerClient {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return pb.NewCmdManagerClient(c.clientConn)
}

func (c *globalClientConn) GetCmdManagerClientExecuteStream() (pb.CmdManager_ExecuteClient, error) {
	client := c.GetCmdManagerClient()
	stream, err := client.Execute(context.Background())
	if err != nil {
		alog.Errorf("Create stream failed: %v", err)
		return nil, err
	}
	return stream, nil
}

func (c *globalClientConn) IsActive() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.clientConn != nil && c.clientConn.GetState() == connectivity.Ready
}

func (c *globalClientConn) GetConn() *grpc.ClientConn {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.clientConn
}

func (c *globalClientConn) SetConn(conn *grpc.ClientConn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.clientConn = conn
}

func (c *globalClientConn) ReConn(cli *ClientConfig) error {
	conn, err := dialGRPCConn(cli)
	if err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.clientConn = conn
	return nil
}

func (c *globalClientConn) CloseConn() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.readyCh = make(chan struct{})

	if c.clientConn == nil {
		return nil
	}
	if err := c.clientConn.Close(); err != nil {
		return err
	}
	return nil
}

func dialGRPCConn(cli *ClientConfig) (*grpc.ClientConn, error) {
	// FIXME NOW EXPERIMENTAL make configurable?
	//var kacp = keepalive.ClientParameters{
	//	Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
	//	Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
	//	PermitWithoutStream: true,             // send pings even without active streams
	//}
	opts := []grpc.DialOption{
		grpc.WithBlock(), /*grpc.WithKeepaliveParams(kacp)*/
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()
	if len(cli.Cert) != 0 {
		opts = append(opts, grpc.WithTransportCredentials(cli.Creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	return grpc.DialContext(ctx, cli.Addr, opts...)
}

func (c *globalClientConn) ConnOnReady() <-chan struct{} {
	return c.readyCh
}

func (c *globalClientConn) ConnOnStates(stats ...connectivity.State) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			for _, s := range stats {
				if c.GetConn().GetState() == s {
					return
				}
			}
			change := c.GetConn().WaitForStateChange(context.Background(), c.GetConn().GetState())
			if !change {
				// not changed but ctx done
				return
			}

			// changed
			for _, s := range stats {
				if c.GetConn().GetState() == s {
					return
				}
			}
		}
	}()

	return done
}

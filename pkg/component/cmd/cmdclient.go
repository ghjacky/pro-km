package cmd

import (
	"context"
	"io"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
	"google.golang.org/grpc/connectivity"
)

// cmdClient implements interface CmdListener
type cmdClient struct {
	conn            conns.GlobalConn
	cmdManager      *cmdManager
	stopCh          <-chan struct{}
	onReadyFuncs    []func()
	onNotReadyFuncs []func()
}

// NewCmdClient build command grpc tunnel to lis
func NewCmdClient(name string, conn conns.GlobalConn, stopCh <-chan struct{}) Client {
	cc := &cmdClient{}
	cc.conn = conn
	cc.cmdManager = newCmdManager(name, cc.sendCmdPackage, cc.respCmdSure, stopCh)
	cc.stopCh = stopCh
	return cc
}

// StartCmdListener start listen and receive cmd and feedback cmd result in loop
func (cc *cmdClient) StartListen(afterSendRegisterCmd chan struct{}) {
	alog.Infof("Starting receive and execute cmds from server")

	for {
		select {
		case <-cc.stopCh:
			return
		default:
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()

			// call manager to get reader of execution
			client := cc.conn.GetCmdManagerClient()
			stream, err := client.Execute(ctx)
			if err != nil {
				alog.Errorf("Create reader failed: %v", err)
				return
			}

			// send RegCallback command to server, register the grpc reader between client and server on server side
			if err := cc.sendRegisterCmd(stream); err != nil {
				return
			}

			// call back ready funcs
			cc.fireOnReady()

			afterSendRegisterCmd <- struct{}{}

			// Always receive command to execute and response sure
			for {
				select {
				case <-cc.stopCh:
					return
				default:
				}

				alog.V(4).Info("Listen to receive cmd")
				_, err := cc.cmdManager.onReceive(stream.Recv)
				if err == io.EOF {
					alog.V(4).Infof("Client received EOF msg")
					continue
				}
				if err != nil {
					alog.Errorf("Client receive msg from server failed: %v", err)
					return
				}
			}
		}()

		select {
		case <-cc.stopCh:
			return
		case <-ctx.Done():
			alog.V(4).Infof("Stream connection closed, reconnect to server")
			// call back not ready funcs
			cc.fireOnNotReady()
			cc.conn.PollConn()
		case <-cc.conn.ConnOnStates(connectivity.Shutdown, connectivity.TransientFailure):
			alog.V(4).Infof("Connection shutdown or failure, reconnect to server")
			// call back not ready funcs
			cc.fireOnNotReady()
			cc.conn.PollConn()

		}
	}
}

// NewCmdReq build cmd for agent
func (cc *cmdClient) NewCmdReq(name Name, args Args) *Req {
	return cc.cmdManager.newCmdReq(name, args, "")
}

// AddCmdHandler register cmd and handler for executor
func (cc *cmdClient) AddCmdHandler(name Name, handler Handler) {
	cc.cmdManager.AddCmdHandler(name, handler)
}

// SendSync send cmd sync
func (cc *cmdClient) SendSync(cmd *Req, timeoutSecond int) (*Resp, error) {
	return cc.cmdManager.SendSync(cmd, timeoutSecond)
}

// SendAsync send cmd async
func (cc *cmdClient) SendAsync(cmd *Req, callback Callback) error {
	return cc.cmdManager.SendAsync(cmd, callback)
}

// Name return the unique name of cmdClient
func (cc *cmdClient) Name() string {
	return cc.cmdManager.Name()
}

func (cc *cmdClient) OnReady(ready func()) {
	cc.onReadyFuncs = append(cc.onReadyFuncs, ready)
}

func (cc *cmdClient) OnNotReady(notReady func()) {
	cc.onNotReadyFuncs = append(cc.onNotReadyFuncs, notReady)
}

func (cc *cmdClient) sendCmdPackage(cmd *pb.CmdPackage) error {
	client, err := cc.conn.GetCmdManagerClientExecuteStream()
	if err != nil {
		return err
	}
	return client.Send(cmd)
}

func (cc *cmdClient) respCmdSure(resp *pb.CmdPackage) error {
	for {
		select {
		case <-cc.stopCh:
			return nil
		default:
		}

		// if send succeed return loop
		err := cc.sendCmdPackage(resp)
		if err == nil {
			alog.V(4).Infof("Send cmd response succeed: id=%s name=%s, code=%d msg=%s data=[%d bytes]",
				resp.UUID, resp.Name, resp.RespCode, resp.RespMsg, len(resp.RespData))
			return nil
		}

		alog.Errorf("Send cmd with id %s response failed: %v", resp.UUID, err)
		select {
		case <-cc.conn.ConnOnReady():
			alog.V(4).Infof("Send cmd on states Ready")
		}
	}
}

// genRegisterCmd build a special cmd to notify server ready to execute command
func (cc *cmdClient) genRegisterCmd() *pb.CmdPackage {
	return &pb.CmdPackage{
		UUID:   uuid.NewUUID(),
		Name:   string(Register),
		Caller: cc.cmdManager.Name(),
	}
}

// sendRegisterCmd send register cmd to grpc server
func (cc *cmdClient) sendRegisterCmd(stream pb.CmdManager_ExecuteClient) error {
	// send register command to server
	if err := stream.Send(cc.genRegisterCmd()); err != nil {
		alog.Errorf("Send register command failed: %v", err)
		return err
	}
	alog.V(4).Infoln("Sent register command successfully")
	return nil
}

func (cc *cmdClient) fireOnReady() {
	go func() {
		for _, f := range cc.onReadyFuncs {
			f()
			select {
			case <-cc.stopCh:
				return
			}
		}
	}()
}

func (cc *cmdClient) fireOnNotReady() {
	go func() {
		for _, f := range cc.onNotReadyFuncs {
			f()
			select {
			case <-cc.stopCh:
				return
			}
		}
	}()
}

package cmd

import (
	"context"
	"fmt"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

type cmdServer struct {
	cmdManager  *cmdManager
	connManager *conns.ConnManager
}

// NewCmdServer build a Server instance
func NewCmdServer(stopCh <-chan struct{}) Server {
	cs := &cmdServer{}
	cs.cmdManager = newCmdManager("CmdServer", cs.reqCmd, cs.respCmd, stopCh)
	cs.connManager = conns.NewConnManager()
	return cs
}

// NewCmdReq build a CmdReq instance
func (cs *cmdServer) NewCmdReq(name Name, args Args, executor string) *Req {
	return cs.cmdManager.newCmdReq(name, args, executor)
}

// Execute impl grpc service, manager send command to device agent，and receive response
func (cs *cmdServer) Execute(stream pb.CmdManager_ExecuteServer) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			alog.Infof("Received terminated signal")
			return stream.Context().Err()
		default:
		}

		// Listen to receive cmd
		recv, err := cs.cmdManager.onReceive(stream.Recv)
		if err != nil {
			alog.Errorf("Server receive msg from agent failed: %v", err)
			continue
		}
		// register executor when receive register cmd
		cs.handleRegisterIfNeed(ctx, recv, stream)
	}
}

// SendSync send cmd until get result, will block
func (cs *cmdServer) SendSync(cmd *Req, timeoutSecond int) (*Resp, error) {
	return cs.cmdManager.SendSync(cmd, timeoutSecond)
}

// SendAsync send cmd async
func (cs *cmdServer) SendAsync(cmd *Req, callback Callback) error {
	return cs.cmdManager.SendAsync(cmd, callback)
}

// AddCmdHandler add handler func to a cmd
func (cs *cmdServer) AddCmdHandler(name Name, handler Handler) {
	cs.cmdManager.AddCmdHandler(name, handler)
}

// Name return the unique name of cmdServer
func (cs *cmdServer) Name() string {
	return cs.cmdManager.Name()
}

// GetConnManager return connection manager
func (cs *cmdServer) GetConnManager() *conns.ConnManager {
	return cs.connManager
}

// reqCmd request cmd to executor at client
func (cs *cmdServer) reqCmd(c *pb.CmdPackage) error {
	executor, err := cs.connManager.GetConnValue(c.Executor)
	if err != nil {
		return fmt.Errorf("no available executors %q, please check executor", c.Executor)

	}
	return executor.(pb.CmdManager_ExecuteServer).Send(c)
}

// respCmd response cmd to caller at agent
func (cs *cmdServer) respCmd(c *pb.CmdPackage) error {
	executor, err := cs.connManager.GetConnValue(c.Caller)
	if err != nil || executor == nil {
		return fmt.Errorf("agent %q connection is lost: %v", c.Caller, err)
	}
	for i := 1; i <= 3; i++ {
		if err = executor.(pb.CmdManager_ExecuteServer).Send(c); err == nil {
			alog.V(4).Infof("Response cmd %s[%s] succeed", c.Name, c.UUID)
			return nil
		}
	}
	return fmt.Errorf("retry 3 times response cmd to agent %s failed: %v", c.Caller, err)
}

// handleRegisterIfNeed handler register connection of client
func (cs *cmdServer) handleRegisterIfNeed(ctx context.Context, recv *pb.CmdPackage, stream pb.CmdManager_ExecuteServer) {
	if recv.Name != string(Register) {
		return
	}
	registerResp := &pb.CmdPackage{
		Type:     pb.CmdPackage_RESPONSE,
		UUID:     recv.UUID,
		Name:     recv.Name,
		Caller:   recv.Caller,
		Executor: recv.Executor,
	}
	// register executor when receive register cmd
	if err := cs.connManager.SaveConn(ctx, recv.Caller, recv.Args, stream); err != nil {
		alog.Errorf("Register agent connection %s failed: %v", recv.Caller, err)
		registerResp.RespCode = FailCode
		registerResp.RespMsg = FailMsg
	} else {
		alog.V(4).Infof("Registered agent connection %q, info: %v", recv.Caller, recv.Args)
		registerResp.RespCode = SuccessCode
		registerResp.RespMsg = SuccessMsg
	}
	// response register cmd
	if err := cs.respCmd(registerResp); err != nil {
		alog.Warningf("Response register cmd failed: %v", err)
		return
	}
	alog.V(4).Info("Response register cmd succeed")
}

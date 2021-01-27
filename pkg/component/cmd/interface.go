package cmd

import (
	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
)

// Manager define common funcs of a cmdManager
type Manager interface {
	// AddCmdHandler register cmd and handler for executor
	AddCmdHandler(name Name, handler Handler)
	// SendSync send cmd sync, will block until received response or timeout
	SendSync(cmd *Req, timeoutSecond int) (*Resp, error)
	// SendAsync send cmd async, will return result immediately, and call callback if response received
	SendAsync(cmd *Req, callback Callback) error
	// Name return the name of executor
	Name() string
}

// Server manage the cmd between agent and server
type Server interface {
	pb.CmdManagerServer
	Manager
	// NewCmdReq build a CmdReq instance
	NewCmdReq(name Name, args Args, executor string) *Req
	// GetConnManager return connection manager
	GetConnManager() *conns.ConnManager
}

// Client start a command bi-tunnel to listen and exec command
type Client interface {
	Manager
	// StartListen start listen and receive cmd and feedback cmd result in loop
	StartListen(afterSendRegisterCmd chan struct{})
	// NewCmdReq build a CmdReq instance
	NewCmdReq(name Name, args Args) *Req
	// OnReady listen connection is ready
	OnReady(ready func())
	// OnNotReady listen connection is not ready
	OnNotReady(ready func())
}

// Callback is the func to call when receive cmd response
type Callback func(resp *Resp)

// Handler handle func to do cmd, return response and onComplete
type Handler func(req *Req) (*Resp, OnComplete)

// OnComplete handle the result when complete handler response
type OnComplete func(err error)

package cmd

import (
	"fmt"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

type executor struct {
	name        string
	cmdHandlers map[Name]Handler
	streams     map[string]*streamer
}

func newExecutor(name string) *executor {
	exec := &executor{
		name:        name,
		cmdHandlers: make(map[Name]Handler),
		streams:     make(map[string]*streamer),
	}
	exec.addHandler(CloseStream, exec.closeSteamHandler)
	return exec
}

func (e *executor) exec(cmd *pb.CmdPackage) (*Resp, func(err error)) {
	cmd.Type = pb.CmdPackage_RESPONSE

	handler, ok := e.cmdHandlers[Name(cmd.Name)]
	if !ok {
		alog.Errorf("cmd %s not supported at executor %s", cmd.Name, e.name)
		return &Resp{
			Code: FailCode,
			Msg:  fmt.Sprintf("cmd %s not supported at executor %s", cmd.Name, e.name),
			Data: "",
		}, nil
	}

	resp, onComplete := handler(&Req{
		UUID:     cmd.UUID,
		Name:     Name(cmd.Name),
		Args:     cmd.Args,
		Caller:   cmd.Caller,
		Executor: cmd.Executor,
	})
	if resp.IsStream() {
		// save cmd stream to close it when received CloseStream cmd
		e.streams[cmd.UUID] = resp.stream
	}
	return resp, onComplete
}

func (e *executor) addHandler(name Name, handler Handler) {
	e.cmdHandlers[name] = handler
}

func (e *executor) closeSteamHandler(req *Req) (*Resp, OnComplete) {
	if stream, ok := e.streams[req.UUID]; ok {
		stream.close()
	}
	alog.Infof("Closed cmd stream succeed: %s", req.UUID)
	return RespSucceed("ok"), nil
}

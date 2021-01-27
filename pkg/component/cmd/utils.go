package cmd

import (
	"fmt"

	pb "code.xxxxx.cn/platform/galaxy/pkg/component/cmd/v1"
)

func genReqCmdPackage(cmd *Req, caller string) (*pb.CmdPackage, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("cmd request name can't be empty")
	}
	return &pb.CmdPackage{
		UUID:     cmd.UUID,
		Name:     string(cmd.Name),
		Args:     cmd.Args,
		Caller:   caller,
		Executor: cmd.Executor,
		Type:     pb.CmdPackage_REQUEST,
	}, nil
}

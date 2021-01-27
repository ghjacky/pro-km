package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"code.xxxxx.cn/platform/galaxy/cmd/agent/app"
	"code.xxxxx.cn/platform/galaxy/pkg/util/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewAgentCommand()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

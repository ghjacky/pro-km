package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"code.xxxxx.cn/platform/galaxy/cmd/server/app"
	"code.xxxxx.cn/platform/galaxy/pkg/util/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.NewCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

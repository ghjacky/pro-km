package app

import (
	"os"

	"code.xxxxx.cn/platform/galaxy/pkg/manager"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	utilflag "code.xxxxx.cn/platform/galaxy/pkg/util/flag"
	"code.xxxxx.cn/platform/galaxy/pkg/util/wait"
	"code.xxxxx.cn/platform/galaxy/pkg/version"
	"code.xxxxx.cn/platform/galaxy/pkg/version/verflag"
	"github.com/spf13/cobra"
)

const (
	// EnvHome env work home dir
	EnvHome = "GALAXY_HOME"
)

// NewCommand build a cobra command
func NewCommand() *cobra.Command {
	opts := NewOptions()
	cmd := &cobra.Command{
		Use:  "training-manager",
		Long: `This is a training manager(maya)`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCmd(cmd, opts); err != nil {
				alog.Fatal(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// runCmd start run cmd
func runCmd(cmd *cobra.Command, opt *Options) error {
	// print version if needed
	verflag.PrintAndExitIfRequested()
	// print all cmd options
	utilflag.PrintFlags(cmd.Flags())

	alog.Infof("Galaxy Version: %+v", version.Get())

	// check cmd option completed if necessary
	err := opt.validate()
	if err != nil {
		return err
	}

	// start run server
	return Run(opt, wait.NeverStop)
}

// Run start run server
func Run(opt *Options, stopCh <-chan struct{}) error {
	// set home env
	if err := os.Setenv(EnvHome, opt.config.WorkDir); err != nil {
		alog.Errorf("Set env GALAXY_HOME failed: %v", err)
		return err
	}
	// start run server
	manager.New(opt.config, stopCh).Run()
	return nil
}

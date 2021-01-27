package app

import (
	"context"
	"fmt"
	"os"

	"code.xxxxx.cn/platform/galaxy/pkg/agent"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	utilflag "code.xxxxx.cn/platform/galaxy/pkg/util/flag"
	"code.xxxxx.cn/platform/galaxy/pkg/util/runtime"
	"code.xxxxx.cn/platform/galaxy/pkg/version"
	"code.xxxxx.cn/platform/galaxy/pkg/version/verflag"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/leaderelection"
)

// EnvHome home system environment
const EnvHome = "MAYA_HOME"

// NewAgentCommand build a agent command
func NewAgentCommand() *cobra.Command {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:  "training-agent",
		Long: `this is training agent`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCmd(cmd, opts); err != nil {
				alog.Fatal(err)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func runCmd(cmd *cobra.Command, opt *Options) error {
	verflag.PrintAndExitIfRequested()
	utilflag.PrintFlags(cmd.Flags())
	alog.Infof("Starting Agent version: %+v", version.Get())

	cfg, err := opt.Config()
	if err != nil {
		return err
	}
	stopCh := make(chan struct{})

	return Run(cfg, stopCh)
}

// Run start agent, start sync cache of informers and leader election
func Run(c *Configuration, stopCh <-chan struct{}) error {
	// set env MAYA_HOME
	if err := os.Setenv(EnvHome, c.Config.WorkDir); err != nil {
		alog.Errorf("Set env MAYA_HOME failed: %v", err)
		return err
	}

	a, err := agent.New(c.Config, c.Client, c.InformerFactory, stopCh)
	if err != nil {
		return err
	}

	// TODO start up the healthz server.
	// TODO start up the metrics server.

	// start informers and wait for syncing cache
	c.InformerFactory.Start(stopCh)
	c.InformerFactory.WaitForCacheSync(stopCh)
	alog.Infof("Informers cache synced")

	// build a context
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go func() {
		select {
		case <-stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// prepare a run func
	run := func(ctx context.Context) {
		a.Run()
		<-ctx.Done()
	}

	// if enable leader election
	if c.LeaderElection != nil {
		c.LeaderElection.Callbacks = leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				runtime.HandleError(fmt.Errorf("lost master"))
			},
		}
		leaderElector, err := leaderelection.NewLeaderElector(*c.LeaderElection)
		if err != nil {
			return fmt.Errorf("couldn't create leader elector: %v", err)
		}

		leaderElector.Run(ctx)

		return fmt.Errorf("lost lease")
	}

	// if disabled leader election
	run(ctx)
	return fmt.Errorf("exit without leader elect")
}

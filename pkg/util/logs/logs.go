/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logs

import (
	"flag"
	"fmt"
	"log"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/wait"
	"github.com/spf13/pflag"
)

const logFlushFreqFlagName = "log-flush-frequency"

var logFlushFreq = pflag.Duration(logFlushFreqFlagName, 5*time.Second, "Maximum number of seconds between log flushes")

// TODO(thockin): This is temporary until we agree on log dirs and put those into each cmd.
func init() {
	alog.InitFlags(flag.CommandLine)
	_ = flag.Set("logtostderr", "true")
}

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(logFlushFreqFlagName))
}

// AlogWriter serves as a bridge between the standard log package and the glog package.
type AlogWriter struct{}

// Write implements the io.Writer interface.
func (writer AlogWriter) Write(data []byte) (n int, err error) {
	alog.InfoDepth(1, string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	log.SetOutput(AlogWriter{})
	log.SetFlags(0)
	// The default glog flush interval is 5 seconds.
	go wait.Forever(alog.Flush, *logFlushFreq)
}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	alog.Flush()
}

// NewLogger creates a new log.Logger which sends logs to alog.Info.
func NewLogger(prefix string) *log.Logger {
	return log.New(AlogWriter{}, prefix, 0)
}

// GlogSetter is a setter to set glog level.
func GlogSetter(val string) (string, error) {
	var level alog.Level
	if err := level.Set(val); err != nil {
		return "", fmt.Errorf("failed set alog.logging.verbosity %s: %v", val, err)
	}
	return fmt.Sprintf("successfully set alog.logging.verbosity to %s", val), nil
}

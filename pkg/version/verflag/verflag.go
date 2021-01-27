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

// Package verflag defines utility functions to handle command line flags
// related to version of Kubernetes.
package verflag

import (
	"fmt"
	"os"
	"strconv"

	flag "github.com/spf13/pflag"

	"code.xxxxx.cn/platform/galaxy/pkg/version"
)

// VersionValue enum type of version value
type VersionValue int

/* version option consts */
const (
	VersionFalse VersionValue = 0
	VersionTrue  VersionValue = 1
	VersionRaw   VersionValue = 2
)

const strRawVersion string = "raw"

// IsBoolFlag check if is bool flag
func (v *VersionValue) IsBoolFlag() bool {
	return true
}

// Get return version value
func (v *VersionValue) Get() interface{} {
	return VersionValue(*v)
}

// Set set version value
func (v *VersionValue) Set(s string) error {
	if s == strRawVersion {
		*v = VersionRaw
		return nil
	}
	boolVal, err := strconv.ParseBool(s)
	if boolVal {
		*v = VersionTrue
	} else {
		*v = VersionFalse
	}
	return err
}

// String return string format content
func (v *VersionValue) String() string {
	if *v == VersionRaw {
		return strRawVersion
	}
	return fmt.Sprintf("%v", bool(*v == VersionTrue))
}

// Type The type of the flag as required by the pflag.Value interface
func (v *VersionValue) Type() string {
	return "version"
}

// VersionVar put a version variable
func VersionVar(p *VersionValue, name string, value VersionValue, usage string) {
	*p = value
	flag.Var(p, name, usage)
	// "--version" will be treated as "--version=true"
	flag.Lookup(name).NoOptDefVal = "true"
}

// Version build a version value
func Version(name string, value VersionValue, usage string) *VersionValue {
	p := new(VersionValue)
	VersionVar(p, name, value, usage)
	return p
}

const versionFlagName = "version"

var (
	versionFlag = Version(versionFlagName, VersionFalse, "Print version information and quit")
)

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *flag.FlagSet) {
	fs.AddFlag(flag.Lookup(versionFlagName))
}

// PrintAndExitIfRequested will check if the -version flag was passed
// and, if so, print the version and exit.
func PrintAndExitIfRequested() {
	if *versionFlag == VersionRaw {
		fmt.Printf("%#v\n", version.Get())
		os.Exit(0)
	} else if *versionFlag == VersionTrue {
		fmt.Printf("Maya version %s\n", version.Get())
		os.Exit(0)
	}
}

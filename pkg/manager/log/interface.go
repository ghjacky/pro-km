/*
Copyright 2020 The Maya Authors.

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

package log

import "io"

// Manager is controller of training task log
type Manager interface {
	// GetLog return logs matched by params
	GetLog(params map[string]string) ([]string, error)
	// GetLogStream return logs stream
	GetLogStream(params map[string]string, executor string) (io.ReadCloser, error)
}

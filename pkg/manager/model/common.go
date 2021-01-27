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

package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// StringArray type of containers filed of TaskRun
type StringArray []string

// Scan auto translate value from db to InputConfigs
func (i *StringArray) Scan(value interface{}) error {
	if value == nil {
		*i = nil
		return nil
	}
	if b, ok := value.([]byte); ok {
		return json.Unmarshal(b, i)
	}
	return errors.New("invalid scan value")
}

// Value auto translate InputConfigs to db json string
func (i StringArray) Value() (driver.Value, error) {
	j, err := json.Marshal(&i)
	return string(j), err
}

// IntArray type of containers filed of TaskRun
type IntArray []int

// Scan auto translate value from db to IntArray
func (i *IntArray) Scan(value interface{}) error {
	if value == nil {
		*i = nil
		return nil
	}
	if b, ok := value.([]byte); ok {
		return json.Unmarshal(b, i)
	}
	return errors.New("invalid scan value")
}

// Value auto translate IntArray to db json
func (i IntArray) Value() (driver.Value, error) {
	j, err := json.Marshal(&i)
	return string(j), err
}

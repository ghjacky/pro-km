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

package res

import (
	"fmt"
	"strconv"
	"strings"
)

type resourceEnum string

const (
	//ResourceCPU CPU
	ResourceCPU resourceEnum = "cpu"
	//ResourceGPU nvidia.com/gpu
	ResourceGPU resourceEnum = "nvidia.com/gpu"
	//ResourceMemory memory
	ResourceMemory resourceEnum = "memory"
)

// ParseResource return resource value, cpu(core),gpu(card),memory(Gi)
func ParseResource(res string, enum resourceEnum) (float64, error) {
	res = strings.TrimSpace(res)
	if res == "" {
		return 0, fmt.Errorf("value is nil")
	}
	var divisor float64 = 0
	var result float64 = 0
	var err error
	switch enum {
	case ResourceCPU:
		if strings.HasSuffix(res, "m") {
			res = strings.TrimSuffix(res, "m")
			divisor = 1000
		}
		divisor = 1
		if result, err = strconv.ParseFloat(res, 10); err != nil {
			return 0, fmt.Errorf("value format invalid, eg. 1000m or 1")
		}
	case ResourceMemory:
		if strings.HasSuffix(res, "Mi") {
			res = strings.TrimSuffix(res, "Mi")
			divisor = 1024
		}
		if strings.HasSuffix(res, "MiB") {
			res = strings.TrimSuffix(res, "MiB")
			divisor = 1024
		}
		if strings.HasSuffix(res, "Gi") {
			res = strings.TrimSuffix(res, "Gi")
			divisor = 1
		}
		if strings.HasSuffix(res, "GiB") {
			res = strings.TrimSuffix(res, "GiB")
			divisor = 1
		}
		if divisor == 0 {
			return 0, fmt.Errorf("value %q unit not present, use Mi(B), Gi(B)", res)
		}
		if result, err = strconv.ParseFloat(res, 10); err != nil {
			return 0, fmt.Errorf("value %q format invalid, eg. 1024Mi(B) or 1Gi(B)", res)
		}
	case ResourceGPU:
		if result, err = strconv.ParseFloat(res, 10); err != nil {
			return 0, fmt.Errorf("value format invalid, eg. 1")
		}
	default:
		return 0, fmt.Errorf("unknown resource type %s", res)
	}
	return result / divisor, nil
}

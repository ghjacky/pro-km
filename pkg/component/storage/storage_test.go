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

package storage

import (
	"testing"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
)

func TestBuildStorageURL(t *testing.T) {
	url, err := BuildStorageURL(apis.StorageHDFS, "prod-hdfs.xxxxx.cn", "home/task", nil)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	t.Logf(url)
	url, err = BuildStorageURL(apis.StorageCPFS, "", "home/task", nil)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	t.Logf(url)
	url, err = BuildStorageURL(apis.StorageGit, "code.xxxxx.cn", "platform/maya", map[string][]string{"tag": {"v1.0.0"}, "b": {"master"}})
	if err != nil {
		t.Errorf("error: %v", err)
	}
	t.Logf(url)
}

func TestParseStorageURL(t *testing.T) {
	t.Log(ParseStorageURL(""))
	t.Log(ParseStorageURL("hdfs://prod-hdfs.xxxxx.cn/home/task"))
	t.Log(ParseStorageURL("cpfs:///home/task"))
	t.Log(ParseStorageURL("hive://bj-hive.xxxxx.cn/home/task?sql=select * from"))
	t.Log(ParseStorageURL("https://code.xxxxx.cn/platform/galaxy?b=master&tag=v1.0.0"))
	t.Log(ParseStorageURL("https://code.xxxxx.cn/platform/galaxy.git?b=master&tag=v1.0.0"))
	t.Log(ParseStorageURL("git@code.xxxxx.cn/platform/galaxy?b=master&tag=v1.0.0"))
	t.Log(ParseStorageURL("/home/task"))
	t.Log(ParseStorageURL("hdfs://home/task"))
}

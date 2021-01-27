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

package harbor

import (
	"encoding/json"
	"testing"
)

const (
	DefaultHarborURL = "https://registry.xxxxx.cn"
	UserName         = "storepull"
	Password         = "!j5pK$Rn!o"
)

var cli = NewHarborBasicClient(DefaultHarborURL, UserName, Password)

func TestCli_SearchRepositories(t *testing.T) {
	repos, err := cli.SearchRepositories("maya")
	if err != nil {
		t.Errorf("list repos failed: %v", err)
	}
	t.Logf("searched %d repos", len(repos))
	for _, repo := range repos {
		repoJSON, _ := json.Marshal(&repo)
		t.Logf("repo: %v", string(repoJSON))
	}

}

func TestCli_ListRepositoryTags(t *testing.T) {
	tags, err := cli.ListRepositoryTags("platform/maya_build")
	if err != nil {
		t.Errorf("list tags failed: %v", err)
	}
	t.Logf("get %d tags", len(tags))
	for _, tag := range tags {
		tagJSON, _ := json.Marshal(&tag)
		t.Logf("tag: %v", string(tagJSON))
	}
}

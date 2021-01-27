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

package git

import (
	"fmt"
	"net/url"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
)

// ParseRepoURL parse git repo url to schema, domain, repo path and params values
func ParseRepoURL(repoURL string) (scheme, domain, repo string, values url.Values, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", "", nil, err
	}

	// if http or https
	if u.Scheme != "" {
		scheme = u.Scheme
		domain = u.Host
		repo = strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git")
		values = u.Query()
		return
	}

	if strings.HasPrefix(u.Path, "git@") {
		scheme = "git"
		words := strings.Split(strings.TrimPrefix(repoURL, "git@"), "/")
		domain = words[0]
		repo = strings.TrimSuffix(strings.TrimPrefix(u.Path, "git@"+domain+"/"), ".git")
		values = u.Query()
		return
	}

	err = fmt.Errorf("invalid repo url %s", repoURL)
	return
}

// GetGitToken return the git token of user
func GetGitToken(codePath, user string) (string, error) {

	_, dm, _, _, err := ParseRepoURL(codePath)
	if err != nil {
		return "", fmt.Errorf("parse code remote path failed: %v", err)
	}
	gitSetting, err := model.GetOnesGitSetting(user, dm)
	if err != nil {
		return "", fmt.Errorf("get %s's git setting at %s failed: %v", user, dm, err)
	}
	if gitSetting == nil {
		return "", fmt.Errorf("%s's git setting at %s not found", user, dm)
	}

	return gitSetting.AccessToken, nil
}

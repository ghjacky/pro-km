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
	"fmt"
	"net/url"
	"path"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
)

// URL the url of storage resource location
type URL struct {
	Scheme apis.StorageType
	Domain string
	Path   string
	Params url.Values
}

func (url *URL) String() string {
	if url == nil {
		return ""
	}
	result, _ := BuildStorageURL(url.Scheme, url.Domain, url.Path, url.Params)
	return result
}

// BuildStorageURL 生成存储URL
func BuildStorageURL(t apis.StorageType, domain string, p string, params url.Values) (string, error) {
	storageURL := ""
	vars := ""
	if len(params) > 0 {
		vars = fmt.Sprintf("?%s", params.Encode())
	}
	switch t {
	case apis.StorageHDFS:
		storageURL = fmt.Sprintf("hdfs://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageCPFS:
		storageURL = fmt.Sprintf("cpfs://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageGit, apis.StorageHTTPS:
		storageURL = fmt.Sprintf("https://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageHTTP:
		storageURL = fmt.Sprintf("http://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageHive:
		storageURL = fmt.Sprintf("hive://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageCode:
		storageURL = fmt.Sprintf("code://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageTask:
		storageURL = fmt.Sprintf("task://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageModel:
		storageURL = fmt.Sprintf("model://%s%s%s", domain, path.Join("/", p), vars)
	case apis.StorageDataSet:
		storageURL = fmt.Sprintf("dataset://%s%s%s", domain, path.Join("/", p), vars)
	default:
		return "", fmt.Errorf("unknown storage type %s", t)
	}
	return storageURL, nil
}

// ParseStorageURL 解析存储URL
func ParseStorageURL(storageURL string) (*URL, error) {
	u, err := url.Parse(storageURL)
	if err != nil {
		return nil, err
	}

	// if http or https
	if u.Scheme != "" {
		return &URL{
			Scheme: apis.StorageType(u.Scheme),
			Domain: u.Host,
			Path:   u.Path,
			Params: u.Query(),
		}, nil
	} else if strings.HasPrefix(storageURL, "git@") {
		words := strings.Split(strings.TrimPrefix(storageURL, "git@"), "/")
		domain := words[0]
		path := strings.TrimSuffix(strings.TrimPrefix(u.Path, "git@"+domain), ".git")
		params := u.Query()
		return &URL{
			Scheme: apis.StorageGit,
			Domain: domain,
			Path:   path,
			Params: params,
		}, nil
	}
	return nil, fmt.Errorf("invalid storage url %s", storageURL)
}

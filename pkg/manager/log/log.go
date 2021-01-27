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

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/olivere/elastic/v7"
)

type logManager struct {
	esClient  *elastic.Client
	cmdServer cmd.Server
}

// NewManager build a new log manager
func NewManager(esURL string, cmdServer cmd.Server) Manager {
	return logManager{
		esClient:  newESClient(esURL),
		cmdServer: cmdServer,
	}
}

func newESClient(esURL string) *elastic.Client {
	cfg := []elastic.ClientOptionFunc{
		elastic.SetURL(esURL),
		elastic.SetHealthcheckTimeout(1 * time.Minute),
		//elastic.SetBasicAuth(UserName, Password)),
	}
	client, err := elastic.NewClient(cfg...)
	if err != nil {
		alog.Errorf("new elastic search client err: %v", err)
	}
	return client
}

// GetLog return logs matched by params
func (m logManager) GetLog(params map[string]string) ([]string, error) {
	if m.esClient == nil {
		return nil, fmt.Errorf("es client not initialized")
	}
	var messages []string
	//defer m.esClient.Stop()
	query := elastic.NewBoolQuery()
	query = query.Must(elastic.NewMatchPhraseQuery("kubernetes.pod_name", params["pod_name"]))
	query = query.Must(elastic.NewMatchPhraseQuery("kubernetes.namespace_name", params["namespace"]))
	if _, ok := params["container"]; ok {
		query = query.Must(elastic.NewMatchPhraseQuery("kubernetes.container_name", params["container"]))
	}
	lines, _ := strconv.Atoi(params["lines"])
	ascending := true
	if params["order"] == "ASC" {
		ascending = false
	} else {
		ascending = true
	}
	esResponse, _ := m.esClient.Search().
		Query(query).Sort("@timestamp", ascending).Size(lines).Do(context.Background())
	for _, value := range esResponse.Hits.Hits {
		var mapResult map[string]interface{}
		if err := json.Unmarshal(value.Source, &mapResult); err != nil {
			alog.Errorf("Unmarshal es result failed: %v", err)
			return messages, err
		}
		messages = append(messages, mapResult["message"].(string))
	}
	return messages, nil
}

// GetLogStream return stream of log
func (m logManager) GetLogStream(params map[string]string, executor string) (io.ReadCloser, error) {
	req := m.cmdServer.NewCmdReq(cmd.GetContainerLog, params, executor)
	resp, err := m.cmdServer.SendSync(req, 0)
	if err != nil {
		return nil, err
	}
	if resp.Code != apis.SuccessCode {
		return nil, fmt.Errorf("resp err: %s", resp.Msg)
	}
	return resp, err
}

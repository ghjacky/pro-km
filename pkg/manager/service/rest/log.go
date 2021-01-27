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

package rest

import (
	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type logService struct {
	cmdServer cmd.Server
}

// NewLogService build a log service
func NewLogService(cmdServer cmd.Server) RestfulService {
	return &logService{
		cmdServer: cmdServer,
	}
}

// RestfulService only to use swagger doc
func (ls *logService) RestfulService() *restful.WebService {
	ws := new(restful.WebService)

	tags := []string{"日志服务"}
	ws.Path("/api/v1/logs").
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/stream").To(ls.getContainerLog).
		Doc("获取日志流").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("cluster", "集群名称，根据该名称获取容器")).
		Param(ws.QueryParameter("namespace", "命名空间")).
		Param(ws.QueryParameter("pod", "POD名称")).
		Param(ws.QueryParameter("container", "容器名称")).
		Param(ws.QueryParameter("follow", "是否跟随日志").DataType("bool")).
		Param(ws.QueryParameter("previous", "是否显示上次容器日志").DataType("bool")).
		Param(ws.QueryParameter("timestamps", "是否显示日志时间戳").DataType("bool")).
		Param(ws.QueryParameter("since", "日志开始时间").DataType("string")).
		Writes(apis.Page{}))

	return ws
}

func (ls *logService) getContainerLog(req *restful.Request, resp *restful.Response) {
	args := cmd.Args{
		"namespace":  req.QueryParameter("namespace"),
		"pod_name":   req.QueryParameter("pod"),
		"container":  req.QueryParameter("container"),
		"previous":   req.QueryParameter("previous"),
		"timestamps": req.QueryParameter("timestamps"),
		"since":      req.QueryParameter("since"),
		"tail":       req.QueryParameter("tail"),
		"follow":     "true",
	}

	reqCmd := ls.cmdServer.NewCmdReq(cmd.GetContainerLog, args, req.QueryParameter("cluster"))
	respCmd, err := ls.cmdServer.SendSync(reqCmd, 0)
	if err != nil {
		apis.RespWebsocket(resp, req, err.Error(), nil)
		alog.Errorf("Send err: %v", err)
		return
	}
	if respCmd.Code != apis.SuccessCode {
		apis.RespWebsocket(resp, req, respCmd.Msg, nil)
		alog.Errorf("Resp err: %s", respCmd.Msg)
		return
	}

	apis.RespWebsocket(resp, req, "", respCmd)
}

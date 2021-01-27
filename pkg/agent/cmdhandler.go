package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/syncer"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/utils"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

/*
DefaultServerTimeout default request timeout
*/
const (
	DefaultServerTimeout = 30
)

// FileMetadataInfo file metadata
type FileMetadataInfo struct {
	Namespace   string `namespace`
	Name        string `name`
	OperateType string `operate_type`
	Digest      string `digest`
	LoadType    string `load_type`

	Extends map[string]string `extends`
}

// configInstanceUpdateInfo .
type configInstanceUpdateInfo struct {
	SiteID    string            `json:"site_id"`
	App       string            `json:"app"`
	Hostname  string            `json:"hostname"`
	IP        string            `json:"ip"`
	Namespace string            `json:"namespace"`
	Filenames map[string]string `json:"filenames"`
}

func (ca *Agent) addCmdHandlers() {
	ca.cmdClient.AddCmdHandler(cmd.HelloWorld, ca.CmdHelloWorldHandler)

	ca.cmdClient.AddCmdHandler(cmd.CmdNSPackageHandler, ca.CmdPackageHandler)
	ca.cmdClient.AddCmdHandler(cmd.CmdNSFileHandler, ca.CmdNSFileHandler)

	ca.cmdServer.AddCmdHandler(cmd.CmdFileRefreshHandler, ca.CmdFileRefreshHandler)
	ca.cmdServer.AddCmdHandler(cmd.CmdContentHandler, ca.CmdContentHandler)
}

// CmdHelloWorldHandler .
func (ca *Agent) CmdHelloWorldHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {
	name := req.Args.Get("name")
	alog.Infof("Receive hello from %v", name)
	if name == "" {
		return cmd.RespError(fmt.Errorf("i don't know your name")), nil
	}
	data := fmt.Sprintf("hello %v, i am %v", name, ca.config.ID)

	return cmd.RespSucceed(data), nil
}

// CmdPackageHandler .
func (ca *Agent) CmdPackageHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {
	namespace := req.Args.Get("namespace")
	digest := req.Args.Get("digest")

	ca.vm.AddDownloadPackageTask(namespace, digest)

	return cmd.RespSucceed(""), nil
}

// CmdNSFileHandler .
func (ca *Agent) CmdNSFileHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {
	namespace := req.Args.Get("namespace")
	filename := req.Args.Get("filename")
	version := req.Args.Get("version") // latest

	if version == "" || filename == "" || namespace == "" {
		return &cmd.Resp{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid Args: ns: %v, filename: %v, version: %v", namespace, filename, version),
		}, nil
	}
	ca.vm.AddDownloadFileTask(namespace, filename, version)

	return cmd.RespSucceed(""), nil
}

// CmdFileRefreshHandler .
func (ca *Agent) CmdFileRefreshHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {
	fileContents := req.Args.Get("filecontents")

	infos := map[string]map[string]string{}
	if err := json.Unmarshal([]byte(fileContents), &infos); err != nil {
		return &cmd.Resp{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}, nil
	}

	differentFiles := map[string][]string{}
	missedFiles := map[string][]string{}

	for ns, diffed := range infos {
		df, mf := ca.fm.Diff(ns, diffed)
		if len(df) > 0 {
			differentFiles[ns] = []string{}
			for name := range df {
				differentFiles[ns] = append(differentFiles[ns], name)
			}

		}
		if len(mf) > 0 {
			missedFiles[ns] = []string{}
			for name := range mf {
				missedFiles[ns] = append(missedFiles[ns], name)
			}
		}

	}

	go func(conn string) {
		app := ca.syncer.GetAppByConn(req.Caller)
		if app == nil {
			alog.Errorf("report updated info failed because app is empty, conn: %v ", req.Caller)
			return
		}
		// report to server
		// TODO: consider better async exec
		for ns, diffed := range infos {
			ca.reportUpdatedInfoToRemote(app, ns, diffed)
		}
	}(req.Caller)

	if len(missedFiles) > 0 {
		// TODO, consider better async exec
		required := map[string]map[string]string{}
		for ns, files := range missedFiles {
			required[ns] = map[string]string{}
			for _, f := range files {
				required[ns][f] = ""
			}
		}

		go ca.refreshAgentFDBFromRemote(required, false)
	}

	for ns, files := range missedFiles {
		if _, ok := differentFiles[ns]; !ok {
			differentFiles[ns] = files
		} else {
			for _, name := range files {
				differentFiles[ns] = append(differentFiles[ns], name)
			}
		}
	}

	data, err := json.Marshal(differentFiles)
	if err != nil {
		return &cmd.Resp{
			Code: 200,
		}, nil
	}

	return cmd.RespSucceed(string(data)), nil
}

func (ca *Agent) refreshFilemapperFromLocal() map[string]map[string]string {
	current := ca.fm.GetAll()
	missiedOrNotCurrent := map[string]map[string]string{}
	for ns, files := range current {
		for file, dg := range files {
			extend, err := ca.fdb.ExistsFile(ca.fdb.GetFilePath(utils.FdbSite, ns, file, utils.FdbVersion))
			if err == nil && dg == extend["digest"] {
				continue
			}
			if _, ok := missiedOrNotCurrent[ns]; !ok {
				missiedOrNotCurrent[ns] = map[string]string{}
			}
			missiedOrNotCurrent[ns][file] = extend["digest"]
		}
	}
	return missiedOrNotCurrent
}

func (ca *Agent) refreshAgentFDBFromRemote(required map[string]map[string]string, full bool) error {
	filesData, err := json.Marshal(required)
	if err != nil {
		alog.Warningf("Marshal miss failed:%v", err)
		return err
	}

	var resp *cmd.Resp
	if full {
		resp, err = ca.cmdClient.SendSync(ca.cmdClient.NewCmdReq(cmd.FileSyncHandler, map[string]string{
			"siteid":       ca.config.ID,
			"filecontents": string(filesData),
		}), DefaultServerTimeout)
	} else {
		resp, err = ca.cmdClient.SendSync(ca.cmdClient.NewCmdReq(cmd.CmdFileRefreshHandler, map[string]string{
			"siteid":       ca.config.ID,
			"filecontents": string(filesData),
		}), DefaultServerTimeout)
	}

	if err != nil {
		alog.Errorf("request upstream failed: %v", err)
		return err
	}

	data := map[string]map[string]string{}
	if err := json.Unmarshal([]byte(resp.Data), &data); err != nil {
		alog.Errorf("unmarashal resp failed: %v", err)
		// wait for next time sync refresh
		return err
	}

	for ns, files := range data {
		for filename, version := range files {
			ca.vm.AddDownloadFileTask(ns, filename, version)
		}
	}

	for ns, files := range required {
		missedFiles := []string{}
		if existedData, ok := data[ns]; ok {
			for f := range files {
				if _, ok := existedData[f]; !ok {
					missedFiles = append(missedFiles, f)
				}
			}
		} else {
			for f := range files {
				missedFiles = append(missedFiles, f)
			}
		}

		for _, file := range missedFiles {
			_, err := ca.fdb.StoreConfig(utils.FdbSite, ns, file, utils.FdbVersion, map[string]string{
				"_status": strconv.Itoa(http.StatusNotFound),
			}, bytes.NewBuffer([]byte{}))
			if err != nil {
				alog.Errorf("Store Not Found File Failed:%v", err)
			}
		}
	}

	return nil
}

func (ca *Agent) reportUpdatedInfoToRemote(app *syncer.AppDescribe, ns string, diffed map[string]string) {
	data, err := json.Marshal(configInstanceUpdateInfo{
		SiteID:    ca.config.ID,
		App:       app.AppID,
		IP:        app.PodIP,
		Hostname:  app.Hostname,
		Namespace: ns,
		Filenames: diffed,
	})

	if err != nil {
		alog.Error("marshal update info failed, when refresh")
		return
	}
	reportReq := ca.cmdClient.NewCmdReq(cmd.CmdUpdatedHandler, cmd.Args{
		"updateinfo": string(data),
	})

	_, err = ca.cmdClient.SendSync(reportReq, DefaultServerTimeout)
	if err != nil {
		alog.Infof("report updated Info To Remote Failed: %v", err)
		// TODO: report again
	}
}

// FileContent .
type FileContent struct {
	Extend  map[string]string `json:"extend"`
	Content string            `json:"content"`
	Digest  string            `json:"digest"`
}

// CmdContentHandler provide file content in response data
func (ca *Agent) CmdContentHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {
	namespace := req.Args.Get("namespace")
	filename := req.Args.Get("filename")
	digest := req.Args.Get("digest")

	ca.syncer.RegisterFileToApp(req.Caller, namespace, filename)

	extend, r, err := ca.fdb.VisitConfig(utils.FdbSite, namespace, filename, utils.FdbVersion)
	if err != nil {
		alog.Errorf("Config Visited Failed:%v", err)

		if err := ca.refreshAgentFDBFromRemote(map[string]map[string]string{namespace: {filename: digest}}, false); err != nil {
			alog.Errorf("Refresh Agent FDB Failed when content not exist:%v", err)
		}
		return &cmd.Resp{
			Code: 204,
		}, nil
	}
	defer r.Close()
	if status, ok := extend["_status"]; ok && status == strconv.Itoa(http.StatusNotFound) {
		return &cmd.Resp{
			Code: http.StatusNotFound,
		}, nil
	}

	if localDigest, ok := extend["digest"]; ok && localDigest == digest {
		return &cmd.Resp{
			Code: http.StatusNotModified,
		}, nil
	}

	fileData, err := ioutil.ReadAll(r)
	if err != nil {
		alog.Errorf("read content failed: %v", err)
		return &cmd.Resp{
			Code: 204,
		}, nil
	}

	rtnData, err := json.Marshal(FileContent{
		Extend:  extend,
		Content: string(fileData),
		Digest:  extend["digest"],
	})
	if err != nil {
		alog.Error("marshal content failed: %v", err)
		return &cmd.Resp{
			Code: 501,
		}, nil
	}

	return cmd.RespSucceed(string(rtnData)), func(err error) {
		if err != nil {
			alog.Errorf("Content Download failed: %v", err)
			return
		}

		describe := ca.syncer.GetAppByConn(req.Caller)
		if describe == nil {
			return
		}

		ca.reportUpdatedInfoToRemote(describe, namespace, map[string]string{
			filename: extend["digest"],
		})
	}
}

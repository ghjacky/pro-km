package configserver

import (
	"encoding/json"
	"fmt"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	model "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"
)

func (cm *configManager) addCmdHandlers() {
	cm.cmdServer.AddCmdHandler(cmd.AppCancelHandler, cm.appCancelHandler)
	cm.cmdServer.AddCmdHandler(cmd.FileUpdateHandler, cm.fileUpdateHandler)
	cm.cmdServer.AddCmdHandler(cmd.FileRefreshHandler, cm.fileRefreshHandler)
	cm.cmdServer.AddCmdHandler(cmd.FileSyncHandler, cm.fileSyncHandler)

}

// appCancelHandler cancel app instance
func (cm *configManager) appCancelHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {

	siteID, err := cm.getSiteIDFromConnKey(req.Caller)
	if err != nil {
		err := fmt.Errorf("invalid request caller: %v", req.Caller)
		return cmd.RespError(err), nil
	}
	app := req.Args.Get("app")
	hostname := req.Args.Get("hostname")

	if err := model.DeleteConfigInstance(app, hostname, siteID, req.Caller, nil); err != nil {
		alog.Errorf("AppCancelHandler: delete config instance %v err: %v", app, err)
		return cmd.RespError(err), nil
	}

	return cmd.RespSucceed("success"), nil

}

// fileUpdateHandler update app instance detail, including digest
func (cm *configManager) fileUpdateHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {

	updateInfo := &model.ConfigInstanceUpdateInfo{}
	infoJSON := req.Args.Get("updateinfo")
	if err := json.Unmarshal([]byte(infoJSON), updateInfo); err != nil {
		return cmd.RespError(err), nil
	}
	var errList []string
	for filename, digest := range updateInfo.Filenames {
		configInfo, err := model.GetConfigInfoByNamespaceNameSiteID(updateInfo.Namespace, filename, updateInfo.SiteID)
		if err != nil {
			alog.Errorf("FileUpdateHandler: get config %v-%v-%v err: %v", updateInfo.SiteID, updateInfo.Namespace, filename, err)
			errList = append(errList, filename)
			continue
		}
		instance := &model.ConfigInstance{
			Name:         updateInfo.App,
			SiteID:       updateInfo.SiteID,
			Hostname:     updateInfo.Hostname,
			IP:           updateInfo.IP,
			Digest:       digest,
			ConfigInfoID: configInfo.ID,
			ConnKey:      req.Caller,
		}
		if err := model.SaveConfigInstance(instance, nil); err != nil {
			alog.Errorf("FileUpdateHandler: save config instance %v-%v / %v err: %v", updateInfo.App, updateInfo.Namespace, filename, err)
			errList = append(errList, filename)
			continue
		}
	}

	if len(errList) != 0 {
		err := fmt.Errorf("FileUpdateHandler: update instance errList: %v", errList)
		return cmd.RespError(err), nil
	}

	return cmd.RespSucceed("success"), nil

}

// fileRefreshHandler diff file digest
func (cm *configManager) fileRefreshHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {

	siteID, err := cm.getSiteIDFromConnKey(req.Caller)
	if err != nil {
		return cmd.RespError(err), nil
	}
	contentJSON := req.Args.Get("filecontents")
	content := make(map[string]map[string]string)

	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return cmd.RespError(err), nil
	}

	diff := make(map[string]map[string]string)
	for ns, file := range content {
		for filename, digest := range file {
			configInfo, err := model.GetConfigInfoByNamespaceNameSiteID(ns, filename, siteID)
			if err != nil {
				alog.Errorf("FileRefreshHandler: get config info %v-%v-%v err: %v", siteID, ns, filename, err)
				continue
			}
			configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
			if err != nil {
				alog.Errorf("FileRefreshHandler: get config current %v err: %v", configInfo.Name, err)
				continue
			}
			release, err := model.GetReleaseByID(configCurrent.LatestReleaseID)
			if err != nil {
				alog.Errorf("FileRefreshHandler: get config release %v err: %v", configInfo.Name, err)
				continue
			}
			configVersion, err := model.GetConfigVersionByID(release.ConfigVersionID)
			if err != nil {
				alog.Errorf("FileRefreshHandler: get config version %v err: %v", configInfo.Name, err)
				continue
			}

			if release.Digest != digest {
				if _, ok := diff[ns]; !ok {
					diff[ns] = make(map[string]string)
				}
				diff[ns][filename] = fmt.Sprintf("%v", configVersion.Version)
			}
		}
	}

	diffResult, err := json.Marshal(diff)
	if err != nil {
		return cmd.RespError(err), nil
	}

	return cmd.RespSucceed(string(diffResult)), nil
}

// fileSyncHandler file sync digest
func (cm *configManager) fileSyncHandler(req *cmd.Req) (*cmd.Resp, cmd.OnComplete) {

	siteID, err := cm.getSiteIDFromConnKey(req.Caller)
	if err != nil {
		return cmd.RespError(err), nil
	}
	contentJSON := req.Args.Get("filecontents")
	content := make(map[string]map[string]string)

	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return cmd.RespError(err), nil
	}

	query := fmt.Sprintf("site_id=%q AND status != 'deleted'", siteID)
	configInfos, err := model.GetAllConfigInfos(query)
	if err != nil {
		return cmd.RespError(err), nil
	}

	alog.Infof("fileSyncHandler: site %v current: %+v", siteID, content)

	releasedDigest := make(map[string]map[string]string)
	releasedVersion := make(map[string]map[string]string)

	for _, configInfo := range configInfos {
		configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
		if err != nil {
			alog.Errorf("fileSyncHandler: get config current %v err: %v", configInfo.Name, err)
			continue
		}
		if configCurrent.LatestReleaseID == 0 {
			continue
		}
		release, err := model.GetReleaseByID(configCurrent.LatestReleaseID)
		if err != nil {
			alog.Errorf("fileSyncHandler: get config release %v err: %v", configInfo.Name, err)
			continue
		}
		configVersion, err := model.GetConfigVersionByID(release.ConfigVersionID)
		if err != nil {
			alog.Errorf("fileSyncHandler: get config version %v err: %v", configInfo.Name, err)
			continue
		}
		if _, ok := releasedDigest[configInfo.Namespace]; !ok {
			releasedDigest[configInfo.Namespace] = make(map[string]string)
		}
		releasedDigest[configInfo.Namespace][configInfo.Name] = release.Digest

		if _, ok := releasedVersion[configInfo.Namespace]; !ok {
			releasedVersion[configInfo.Namespace] = make(map[string]string)
		}
		releasedVersion[configInfo.Namespace][configInfo.Name] = fmt.Sprintf("%v", configVersion.Version)
	}
	alog.Infof("fileSyncHandler: site %v releasedDigest: %+v", siteID, releasedDigest)

	for ns, file := range content {
		if _, ok := releasedDigest[ns]; !ok {
			continue
		}
		for filename, digest := range file {
			if releasedDigest[ns][filename] == digest {
				delete(releasedVersion[ns], filename)
			}
		}
	}

	diffResult, err := json.Marshal(releasedVersion)
	if err != nil {
		return cmd.RespError(err), nil
	}
	alog.Infof("fileSyncHandler: site %v Diff: %v", siteID, string(diffResult))

	return cmd.RespSucceed(string(diffResult)), nil
}

package application

import (
	"fmt"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

type manager struct {
	cmdServer cmd.Server
}

// NewManager create a new appManager
func NewManager(cmdServer cmd.Server, stopCh <-chan struct{}) Manager {
	m := manager{
		cmdServer: cmdServer,
	}
	m.addCmdHandlers()
	return m
}

// GrpcExample .
func (m manager) GrpcExample(cluster string) (string, error) {
	args := cmd.Args{"name": "server"}
	c := m.cmdServer.NewCmdReq(cmd.HelloWorld, args, cluster)
	resp, err := m.cmdServer.SendSync(c, 3)
	if err != nil {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.HelloWorld, err)
		return "", err
	}
	if resp.Code == apis.FailCode {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.HelloWorld, resp.Msg)
		return "", fmt.Errorf(resp.Msg)
	}
	return resp.Data, nil
}

// AddAppInfo add appInfo
func (m manager) AddAppInfo(appInfo *model.AppInfo) (*model.AppInfo, error) {

	if valid, err := m.checkAppInfoValidity(appInfo); !valid {
		return nil, err
	}

	return model.CreateAppInfo(appInfo)
}

// ListAppInfos get AppInfo list matched query, and order by orders, select a page by offset, limit
func (m manager) ListAppInfos(query string, orders []string, offset, limit int) ([]*model.AppInfo, error) {
	appInfos, err := model.ListAppInfo(query, orders, offset, limit)
	if err != nil {
		return nil, err
	}

	return appInfos, nil
}

// CountAppInfos count all AppInfos matched query
func (m manager) CountAppInfos(query string) (int64, error) {
	return model.CountAppInfos(query)
}

// UpdateAppInfo update AppInfo
func (m manager) UpdateAppInfo(AppInfo *model.AppInfo) error {

	if valid, err := m.checkAppInfoValidity(AppInfo); !valid {
		return err
	}

	return model.UpdateAppInfo(AppInfo)
}

// DeleteAppInfo delete AppInfo
func (m manager) DeleteAppInfo(id uint64) error {
	return model.DeleteAppInfo(id)
}

// AddApp add app
func (m manager) AddApp(app *model.App) (*model.App, error) {

	if valid, err := m.checkAppValidity(app); !valid {
		return nil, err
	}

	return model.CreateApp(app)
}

// ListApps get App list matched query, and order by orders, select a page by offset, limit
func (m manager) ListApps(query string, orders []string, offset, limit int) ([]*model.App, error) {
	apps, err := model.ListApp(query, orders, offset, limit)
	if err != nil {
		return nil, err
	}

	return apps, nil
}

// CountApps count all Apps matched query
func (m manager) CountApps(query string) (int64, error) {
	return model.CountApps(query)
}

// UpdateApp update App
func (m manager) UpdateApp(App *model.App) error {

	if valid, err := m.checkAppValidity(App); !valid {
		return err
	}

	return model.UpdateApp(App)
}

// DeleteApp delete App
func (m manager) DeleteApp(id uint64) error {
	return model.DeleteApp(id)
}

func (m manager) checkAppInfoValidity(appInfo *model.AppInfo) (bool, error) {
	return true, nil
}

func (m manager) checkAppValidity(app *model.App) (bool, error) {
	return true, nil
}

package application

import "code.xxxxx.cn/platform/galaxy/pkg/manager/model"

// Manager define the manager of application
type Manager interface {
	// GrpcExample
	GrpcExample(cluster string) (string, error)
	// AddAppInfo add appInfo
	AddAppInfo(appInfo *model.AppInfo) (*model.AppInfo, error)
	// ListAppInfos get AppInfo list matched query, and order by orders, select a page by offset, limit
	ListAppInfos(query string, orders []string, offset, limit int) ([]*model.AppInfo, error)
	// CountAppInfos count all appInfo matched query
	CountAppInfos(query string) (int64, error)
	// UpdateAppInfo update AppInfo
	UpdateAppInfo(appInfo *model.AppInfo) error
	// DeleteAppInfo delete AppInfo
	DeleteAppInfo(id uint64) error

	// AddApp add App
	AddApp(App *model.App) (*model.App, error)
	// ListApps get App list matched query, and order by orders, select a page by offset, limit
	ListApps(query string, orders []string, offset, limit int) ([]*model.App, error)
	// CountApps count all App matched query
	CountApps(query string) (int64, error)
	// UpdateApp update App
	UpdateApp(App *model.App) error
	// DeleteApp delete App
	DeleteApp(id uint64) error
}

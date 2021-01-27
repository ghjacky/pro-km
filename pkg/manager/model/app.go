package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
)

func init() {
	db.RegisterDBTable(&AppInfo{})
	db.RegisterDBTable(&App{})
}

// AppInfo Describe an app
type AppInfo struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);unique_index:Name_Namespace;not null" json:"name" description:"应用名称（必填）"`
	Namespace string `gorm:"type:varchar(100);unique_index:Name_Namespace;not null" json:"name" description:"应用命名空间（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"应用描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"应用状态: 已创建(created), 已删除(deleted)"`

	Replicas    int         `gorm:"type:int(10)" json:"replicas"`
	Labels      StringArray `gorm:"type:varchar(200)" json:"labels" description:"标签集合，[key=value]"`
	Annotations StringArray `gorm:"type:varchar(200)" json:"annotations" description:"标注集合, [key=value]"`
	Ports       string      `gorm:"type:varchar(1000);" json:"ports" description:"端口信息的json串"`
	Configs     string      `gorm:"type:text" json:"configs" description:"配置文件集合的json串（选填）"`
	Network     string      `gorm:"type:varchar(100)" json:"network" description:"网络类型"`
}

// App Describe an app with version
type App struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);unique_index:Name_Namespace_Version;not null" json:"name" description:"应用名称（必填）"`
	Namespace string `gorm:"type:varchar(100);unique_index:Name_Namespace_Version;not null" json:"name" description:"应用命名空间（必填）"`
	Version   string `gorm:"type:varchar(100);unique_index:Name_Namespace_Version;not null" json:"version" description:"应用版本（必填）"`
	Commit    string `gorm:"type:varchar(100);not null" json:"commit" description:"应用版本commit id（必填）"`
	Image     string `gorm:"type:varchar(100);not null" json:"image" description:"应用版本镜像号（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"应用描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"应用状态: 已创建(created), 已删除(deleted)"`

	Replicas    int         `gorm:"type:int(10)" json:"replicas"`
	Labels      StringArray `gorm:"type:varchar(200)" json:"labels" description:"标签，逗号隔开,[key=value, ]"`
	Annotations StringArray `gorm:"type:varchar(200)" json:"annotations" description:"标注，逗号隔开,[key=value, ]"`
	Ports       string      `gorm:"type:varchar(1000);" json:"ports" description:"端口信息的json串"`
	Configs     string      `gorm:"type:text" json:"configs" description:"配置文件集合的json串（选填）"`
	Network     string      `gorm:"type:varchar(100)" json:"network" description:"网络类型"`

	AppInfoID uint64 `gorm:"type:bigint;not null" json:"-"`

	//TODO: Design a form which can generalize application for YAML

}

// AppPort .
type AppPort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int    `json:"target_port"`
	NodePort   int32  `json:"node_port"`
}

// AppConfig .
type AppConfig struct {
	Path    string
	Name    string
	Content string
	Type    string
}

// CreateAppInfo create a new appInfo
func CreateAppInfo(appInfo *AppInfo) (*AppInfo, error) {
	dbResult := db.Get().Create(appInfo)

	return appInfo, dbResult.Error
}

// ListAppInfo get page of AppInfo list
func ListAppInfo(query string, orders []string, offset int, limit int) ([]*AppInfo, error) {
	appInfos := []*AppInfo{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&appInfos).Error; err != nil {
		return nil, err
	}
	return appInfos, nil
}

// CountAppInfos get total size of apps
func CountAppInfos(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&AppInfo{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count appInfos failed: %v", err)
		return 0, err
	}
	return count, nil
}

// UpdateAppInfo update a appInfo
func UpdateAppInfo(appInfo *AppInfo) error {
	return db.Get().Model(&AppInfo{}).Updates(appInfo).Error
}

// DeleteAppInfo soft delete app info
func DeleteAppInfo(id uint64) error {
	appInfo := &AppInfo{}
	if err := db.Get().Model(appInfo).Where("id = ?", id).First(appInfo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update appInfo name and deleteAt
	return db.Get().
		Model(&AppInfo{}).
		Where("id = ?", id).
		Updates(&AppInfo{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", appInfo.Name, uuid.NewUUID()),
		}).Error
}

// CreateApp create a new app
func CreateApp(app *App) (*App, error) {
	dbResult := db.Get().Create(app)

	return app, dbResult.Error
}

// ListApp get page of AppInfo list
func ListApp(query string, orders []string, offset int, limit int) ([]*App, error) {
	apps := []*App{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// CountApps get total size of apps
func CountApps(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&App{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count apps failed: %v", err)
		return 0, err
	}
	return count, nil
}

// UpdateApp update a app
func UpdateApp(app *App) error {
	return db.Get().Model(&App{}).Updates(app).Error
}

// DeleteApp soft delete app
func DeleteApp(id uint64) error {
	app := &App{}
	if err := db.Get().Model(app).Where("id = ?", id).First(app).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	now := time.Now()

	// update app name and deleteAt
	return db.Get().
		Model(&App{}).
		Where("id = ?", id).
		Updates(&App{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", app.Name, uuid.NewUUID()),
		}).Error
}

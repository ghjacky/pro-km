package model

import (
	"fmt"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&ConfigInfo{})
}

// ConfigInfo config info
type ConfigInfo struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name              string       `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_SiteID;not null" json:"name" description:"配置名称（必填）"`
	Namespace         string       `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_SiteID;not null" json:"namespace" description:"配置命名空间（必填）"`
	Desc              string       `gorm:"type:varchar(512)" json:"desc" description:"配置描述，不超过200个汉字（选填）"`
	CreatedBy         string       `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group             string       `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`
	SiteID            string       `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_SiteID;not null" json:"site_id" description:"项目ID"`
	Format            ConfigFormat `gorm:"type:varchar(20);not null" json:"format" description:"文件类型，property，text, json, yaml, csv, jpg"`
	Status            ConfigStatus `gorm:"type:varchar(20)" json:"status" description:"配置状态: 已发布(released), 未发布(created), 有更新(updated), 已删除(removed), 软删除(deleted)"`
	TemplateVersionID uint64       `gorm:"type:bigint;not null" json:"template_version_id" description:"配置模板ID"`
	TemplateName      string       `gorm:"type:varchar(100);not null" json:"template_name" description:"配置模板名称"`
	TemplateNamespace string       `gorm:"type:varchar(100);not null" json:"template_namespace" description:"配置模板命名空间"`

	Data             string `gorm:"-" json:"data" description:"配置文件内容"`
	TemplateData     string `gorm:"-" json:"template_data" description:"配置模板内容"`
	ReplaceData      string `gorm:"-" json:"replace_data" description:"替换内容"`
	IsTemplateChange bool   `gorm:"-" json:"is_template_change" description:"模板文件是否已更改"`
}

// ConfigStatus the type of config status
type ConfigStatus string

/** all config status */
const (
	ConfigStatusCreated  ConfigStatus = "created"
	ConfigStatusReleased ConfigStatus = "released"
	ConfigStatusUpdated  ConfigStatus = "updated"
	ConfigStatusRemoved  ConfigStatus = "removed"
	ConfigStatusDeleted  ConfigStatus = "deleted"
)

// ConfigFormat the format of config
type ConfigFormat string

/** all config format */
const (
	ConfigFormatProperties ConfigFormat = "property"
	ConfigFormatTxt        ConfigFormat = "txt"
	ConfigFormatJSON       ConfigFormat = "json"
	ConfigFormatYAML       ConfigFormat = "yaml"
	ConfigFormatCSV        ConfigFormat = "csv"
	ConfigFormatJPG        ConfigFormat = "jpg"
)

// ConfigType the type of config
type ConfigType string

/** all config type */
const (
	ConfigCameraInfo ConfigType = "CameraInfos"
	ConfigStoreInfo  ConfigType = "StoreInfos"
	ConfigApp        ConfigType = "app"
)

// ConfigSource the type of config source
type ConfigSource string

/** all config source type */
const (
	ConfigSourceImage ConfigSource = "image"
	ConfigSourceTar   ConfigSource = "tar"
)

// ConfigAndInstance .
type ConfigAndInstance struct {
	Config    *ConfigInfo       `json:"config"`
	Instances []*ConfigInstance `json:"instances"`
}

// CreateConfigInfo create a new configInfo
func CreateConfigInfo(configInfo *ConfigInfo, tx *gorm.DB) (*ConfigInfo, error) {

	if tx == nil {
		tx = db.Get()
	}

	dbResult := tx.Create(configInfo)

	return configInfo, dbResult.Error
}

// ListConfigInfo get page of ConfigInfo list
func ListConfigInfo(query string, orders []string, offset int, limit int) ([]*ConfigInfo, error) {
	configInfos := []*ConfigInfo{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&configInfos).Error; err != nil {
		return nil, err
	}
	return configInfos, nil
}

// CountConfigInfos get total size of apps
func CountConfigInfos(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&ConfigInfo{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count configInfos failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetConfigInfoByID get the configInfo by id
func GetConfigInfoByID(id uint64) (*ConfigInfo, error) {
	configInfo := &ConfigInfo{}
	db := db.Get().Model(&ConfigInfo{}).Where("id = ? AND status != 'deleted'", id)
	if err := db.First(configInfo).Error; err != nil {
		return nil, err
	}
	return configInfo, nil
}

// GetConfigInfoByNamespaceNameSiteID get the configInfo by id
func GetConfigInfoByNamespaceNameSiteID(ns, name, siteID string) (*ConfigInfo, error) {
	configInfo := &ConfigInfo{}
	db := db.Get().Model(&ConfigInfo{}).Where("namespace = ? AND name = ? AND site_id = ? AND status != 'deleted'", ns, name, siteID)
	if err := db.First(configInfo).Error; err != nil {
		return nil, err
	}
	return configInfo, nil
}

// GetAllConfigInfos get all configInfos
func GetAllConfigInfos(query string) ([]*ConfigInfo, error) {

	configInfos := []*ConfigInfo{}
	db := db.Get().Model(&ConfigInfo{}).Where(query)
	if err := db.Find(&configInfos).Error; err != nil {
		return nil, err
	}
	return configInfos, nil
}

// UpdateConfigInfo update a configInfo
func UpdateConfigInfo(configInfo *ConfigInfo, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&ConfigInfo{}).Where(&ConfigInfo{ID: configInfo.ID}).Updates(configInfo).Error
}

// DeleteConfigInfo soft delete app info
func DeleteConfigInfo(id uint64) error {
	configInfo := &ConfigInfo{}
	if err := db.Get().Model(configInfo).Where("id = ?", id).First(configInfo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update configInfo name and deleteAt
	return db.Get().
		Model(&ConfigInfo{}).
		Where("id = ?", id).
		Updates(&ConfigInfo{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", configInfo.Name, uuid.NewUUID()),
		}).Error
}

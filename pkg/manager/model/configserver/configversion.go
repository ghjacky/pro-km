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
	db.RegisterDBTable(&ConfigVersion{})
}

// ConfigVersion the specific version of the config
type ConfigVersion struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string              `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_Version_ConfigInfoID;not null" json:"name" description:"配置名称（必填）"`
	Namespace string              `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_Version_ConfigInfoID;not null" json:"namespace" description:"配置命名空间（必填）"`
	Version   uint64              `gorm:"type:bigint;uniqueIndex:Name_Namespace_Version_ConfigInfoID;not null" json:"version" description:"配置版本（必填）"`
	Desc      string              `gorm:"type:varchar(512)" json:"desc" description:"配置描述，不超过200个汉字（选填）"`
	CreatedBy string              `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group     string              `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`
	SiteID    string              `gorm:"type:varchar(100);not null" json:"site_id" description:"项目ID"`
	Digest    string              `gorm:"type:varchar(100);not null" json:"-" description:"配置的digest值"`
	Status    ConfigVersionStatus `gorm:"type:varchar(20)" json:"-" description:"配置状态: 新建(create), 更新(update), 删除(remove), 回滚(rollback), 软删除(deleted)"`

	ConfigInfoID uint64 `gorm:"type:bigint;uniqueIndex:Name_Namespace_Version_ConfigInfoID;not null" json:"-"`

	TemplateVersionID uint64 `gorm:"type:bigint;not null" json:"-"`
	TemplateName      string `gorm:"type:varchar(100);not null" json:"template_name" description:"配置模板名称"`
	TemplateNamespace string `gorm:"type:varchar(100);not null" json:"template_namespace" description:"配置模板命名空间"`
	ReplaceDigest     string `gorm:"type:varchar(100);not null" json:"-" description:"配置替换值的digest值"`
}

// ConfigVersionStatus the type of config version status
type ConfigVersionStatus string

/** all config status */
const (
	ConfigVersionCreate   ConfigVersionStatus = "create"
	ConfigVersionUpdate   ConfigVersionStatus = "update"
	ConfigVersionRemove   ConfigVersionStatus = "remove"
	ConfigVersionRollback ConfigVersionStatus = "rollback"
	ConfigVersionDeleted  ConfigVersionStatus = "deleted"
)

// CreateConfigVersion create a new configVersion
func CreateConfigVersion(configVersion *ConfigVersion, tx *gorm.DB) (*ConfigVersion, error) {
	if tx == nil {
		tx = db.Get()
	}
	dbResult := tx.Create(configVersion)

	return configVersion, dbResult.Error
}

// ListConfigVersion get page of ConfigVersion list
func ListConfigVersion(query string, orders []string, offset int, limit int) ([]*ConfigVersion, error) {
	configVersions := []*ConfigVersion{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&configVersions).Error; err != nil {
		return nil, err
	}
	return configVersions, nil
}

// CountConfigVersions get total size of apps
func CountConfigVersions(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&ConfigVersion{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count configVersions failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetConfigVersionByID get the the specific version of the  configInfo by id
func GetConfigVersionByID(id uint64) (*ConfigVersion, error) {
	configVersion := &ConfigVersion{}
	db := db.Get().Model(&ConfigVersion{}).Where("id = ? AND status != 'deleted'", id)
	if err := db.First(configVersion).Error; err != nil {
		return nil, err
	}
	return configVersion, nil
}

// GetConfigVersionByConfigInfoIDAndVersion get the the specific version of the  configInfo by id
func GetConfigVersionByConfigInfoIDAndVersion(id, version uint64) (*ConfigVersion, error) {
	configVersion := &ConfigVersion{}
	db := db.Get().Model(&ConfigVersion{}).Where("config_info_id = ? AND version = ?", id, version)
	if err := db.First(configVersion).Error; err != nil {
		return nil, err
	}
	return configVersion, nil
}

// UpdateConfigVersion update a configVersion
func UpdateConfigVersion(configVersion *ConfigVersion, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&ConfigVersion{}).Where(&ConfigVersion{ID: configVersion.ID}).Updates(configVersion).Error
}

// DeleteConfigVersion soft delete app info
func DeleteConfigVersion(id uint64) error {
	configVersion := &ConfigVersion{}
	if err := db.Get().Model(configVersion).Where("id = ?", id).First(configVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update configVersion name and deleteAt
	return db.Get().
		Model(&ConfigVersion{}).
		Where("id = ?", id).
		Updates(&ConfigVersion{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", configVersion.Name, uuid.NewUUID()),
		}).Error
}

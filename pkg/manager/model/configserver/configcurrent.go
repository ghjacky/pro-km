package model

import (
	"time"

	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&ConfigCurrent{})
}

// ConfigCurrent 配置当前状态
type ConfigCurrent struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Status string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已创建(created), 已删除(deleted)"`

	ConfigInfoID          uint64 `gorm:"type:bigint;uniqueIndex:ConfigInfoID;not null" json:"-"`
	LatestConfigVersionID uint64 `gorm:"type:bigint;not null" json:"-"`
	LatestReleaseID       uint64 `gorm:"type:bigint;not null" json:"-"`
	PreReleaseID          uint64 `gorm:"type:bigint;not null" json:"-"`
}

// CreateConfigCurrent create a new configCurrent
func CreateConfigCurrent(configCurrent *ConfigCurrent, tx *gorm.DB) (*ConfigCurrent, error) {

	if tx == nil {
		tx = db.Get()
	}
	dbResult := tx.Create(configCurrent)

	return configCurrent, dbResult.Error
}

// GetConfigCurrentByConfigID get the configCurrent of the specific configInfo
func GetConfigCurrentByConfigID(id uint64) (*ConfigCurrent, error) {
	configCurrent := &ConfigCurrent{}
	db := db.Get().Model(&ConfigCurrent{}).Where("config_info_id = ? AND status != 'deleted'", id)
	if err := db.First(configCurrent).Error; err != nil {
		return nil, err
	}
	return configCurrent, nil
}

// UpdateConfigCurrent update a configCurrent
func UpdateConfigCurrent(configCurrent *ConfigCurrent, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&ConfigCurrent{}).Where(&ConfigCurrent{ID: configCurrent.ID}).Updates(configCurrent).Error
}

// DeleteConfigCurrent soft delete app info
func DeleteConfigCurrent(id uint64) error {
	configCurrent := &ConfigCurrent{}
	if err := db.Get().Model(configCurrent).Where("id = ?", id).First(configCurrent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update configCurrent name and deleteAt
	return db.Get().
		Model(&ConfigCurrent{}).
		Where("id = ?", id).
		Updates(&ConfigCurrent{
			DeletedAt: &now,
			Status:    "deleted",
		}).Error
}

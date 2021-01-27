package model

import (
	"time"

	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&ConfigInstance{})
}

// ConfigInstance config instance
type ConfigInstance struct {
	ID        uint64    `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time `json:"updated_at,omitempty" description:"更新时间（不填）"`

	Name     string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID_Hostname_ConfigInfoID_ConnKey;not null" json:"name" description:"实例名称"`
	SiteID   string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID_Hostname_ConfigInfoID_ConnKey;not null" json:"site_id" description:"项目ID"`
	Hostname string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID_Hostname_ConfigInfoID_ConnKey;not null" json:"hostname" description:"主机名"`
	IP       string `gorm:"type:varchar(100);not null" json:"ip" description:"主机IP"`
	Digest   string `gorm:"type:varchar(100);not null" json:"digest" description:"当前文件Digest"`

	ConfigInfoID uint64 `gorm:"type:bigint;uniqueIndex:Name_SiteID_Hostname_ConfigInfoID_ConnKey;not null" json:"-"`
	IsSync       bool   `gorm:"-" json:"is_sync" description:"是否是最新"`
	ConnKey      string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID_Hostname_ConfigInfoID_ConnKey;" json:"-" description:"agent connection key"`
}

// ConfigInstanceUpdateInfo .
type ConfigInstanceUpdateInfo struct {
	SiteID    string            `json:"site_id"`
	App       string            `json:"app"`
	Hostname  string            `json:"hostname"`
	IP        string            `json:"ip"`
	Namespace string            `json:"namespace"`
	Filenames map[string]string `json:"filenames"`
}

// CreateConfigInstance create a new configInstance
func CreateConfigInstance(configInstance *ConfigInstance, tx *gorm.DB) (*ConfigInstance, error) {

	if tx == nil {
		tx = db.Get()
	}

	dbResult := tx.Create(configInstance)

	return configInstance, dbResult.Error
}

// ListConfigInstance get page of ConfigInstance list
func ListConfigInstance(query string, orders []string, offset int, limit int) ([]*ConfigInstance, error) {
	configInstances := []*ConfigInstance{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&configInstances).Error; err != nil {
		return nil, err
	}
	return configInstances, nil
}

// CountConfigInstances get total size of apps
func CountConfigInstances(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&ConfigInstance{}).Where(query).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetConfigInstanceByID get the configInstance by id
func GetConfigInstanceByID(id uint64) (*ConfigInstance, error) {
	configInstance := &ConfigInstance{}
	db := db.Get().Model(&ConfigInstance{}).Where("id = ?", id)
	if err := db.First(configInstance).Error; err != nil {
		return nil, err
	}
	return configInstance, nil
}

// GetConfigInstanceByConfigID get the configInstance by configInfo id
func GetConfigInstanceByConfigID(id uint64, connKey string) ([]*ConfigInstance, error) {
	configInstances := []*ConfigInstance{}
	db := db.Get().Model(&ConfigInstance{}).Where("config_info_id = ? AND conn_key = ?", id, connKey)
	if err := db.Find(&configInstances).Error; err != nil {
		return nil, err
	}
	return configInstances, nil
}

// GetConfigInstanceByNameHostnameSiteID get the configInstance by id
func GetConfigInstanceByNameHostnameSiteID(name, hostname, siteID string) ([]*ConfigInstance, error) {
	configInstances := []*ConfigInstance{}
	db := db.Get().Model(&ConfigInstance{}).Where("name = ? AND hostname = ? AND site_id = ?", name, hostname, siteID)
	if err := db.Find(configInstances).Error; err != nil {
		return nil, err
	}
	return configInstances, nil
}

// GetConfigInstanceByNameHostnameSiteIDConnAndConfig get the configInstance by id
func GetConfigInstanceByNameHostnameSiteIDConnAndConfig(name, hostname, siteID, connKey string, configID uint64) (*ConfigInstance, error) {
	configInstance := &ConfigInstance{}
	db := db.Get().Model(&ConfigInstance{}).Where("name = ? AND hostname = ? AND site_id = ? AND conn_key = ? AND config_info_id = ?", name, hostname, siteID, connKey, configID)
	if err := db.First(configInstance).Error; err != nil {
		return nil, err
	}
	return configInstance, nil
}

// UpdateConfigInstance update a configInstance
func UpdateConfigInstance(configInstance *ConfigInstance, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&ConfigInstance{}).Where(&ConfigInstance{ID: configInstance.ID}).Updates(configInstance).Error
}

// SaveConfigInstance insert or update configInstance
func SaveConfigInstance(configInstance *ConfigInstance, tx *gorm.DB) error {

	if tx == nil {
		tx = db.Get()
	}

	cur, err := GetConfigInstanceByNameHostnameSiteIDConnAndConfig(configInstance.Name, configInstance.Hostname, configInstance.SiteID, configInstance.ConnKey, configInstance.ConfigInfoID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return tx.Create(configInstance).Error
		}
		return err
	}

	cur.IP = configInstance.IP
	cur.Digest = configInstance.Digest

	return tx.Model(&ConfigInstance{}).Where(&ConfigInstance{ID: cur.ID}).Updates(cur).Error
}

// DeleteConfigInstance .
func DeleteConfigInstance(name, hostname, siteID, connKey string, tx *gorm.DB) error {

	if tx == nil {
		tx = db.Get()
	}

	configInstance := &ConfigInstance{Name: name, Hostname: hostname, SiteID: siteID, ConnKey: connKey}
	if err := tx.Model(&ConfigInstance{}).Where("name = ? AND hostname = ? AND site_id = ? AND conn_key = ?", name, hostname, siteID, connKey).Delete(configInstance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}

// DeleteConfigInstanceByConnKey .
func DeleteConfigInstanceByConnKey(connKey string) error {

	configInstance := &ConfigInstance{ConnKey: connKey}
	if err := db.Get().Model(&ConfigInstance{}).Where("conn_key = ?", connKey).Delete(configInstance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}

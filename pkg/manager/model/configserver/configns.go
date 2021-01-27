package model

import (
	"time"

	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&ConfigNamespace{})
}

// ConfigNamespace config namespace
type ConfigNamespace struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID;not null" json:"name" description:"命名空间名称（必填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group     string `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`
	SiteID    string `gorm:"type:varchar(100);uniqueIndex:Name_SiteID;not null" json:"site_id" description:"项目ID"`

	Status string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已发布(released), 未发布(created), 有更新(updated), 已删除(removed), 软删除(deleted)"`
}

// FindOrCreateConfigNamespace find or create config namespace
func FindOrCreateConfigNamespace(ns string, siteID string, user string, tx *gorm.DB) (*ConfigNamespace, error) {
	if tx == nil {
		tx = db.Get()
	}

	configNs := &ConfigNamespace{}

	dbResult := tx.Where(ConfigNamespace{Name: ns, SiteID: siteID}).Attrs(ConfigNamespace{CreatedBy: user, Status: "created"}).FirstOrCreate(configNs)

	return configNs, dbResult.Error
}

// ListConfigNamespace get page of config namespace list
func ListConfigNamespace(query string, orders []string, offset int, limit int) ([]*ConfigNamespace, error) {
	configNs := []*ConfigNamespace{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&configNs).Error; err != nil {
		return nil, err
	}
	return configNs, nil
}

// CountConfigNamespace get total size of config namespace
func CountConfigNamespace(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&ConfigNamespace{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count configNs failed: %v", err)
		return 0, err
	}
	return count, nil
}

package model

import (
	"time"

	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

func init() {
	db.RegisterDBTable(&TemplateNamespace{})
}

// TemplateNamespace config namespace
type TemplateNamespace struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);unique;not null" json:"name" description:"命名空间名称（必填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group     string `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`

	Status string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已发布(released), 未发布(created), 有更新(updated), 已删除(removed), 软删除(deleted)"`
}

// FindOrCreateTemplateNamespace find or create template namespace
func FindOrCreateTemplateNamespace(ns string, user string, tx *gorm.DB) (*TemplateNamespace, error) {
	if tx == nil {
		tx = db.Get()
	}
	templateNs := &TemplateNamespace{}
	dbResult := tx.Where(TemplateNamespace{Name: ns}).Attrs(TemplateNamespace{CreatedBy: user, Status: "created"}).FirstOrCreate(templateNs)

	return templateNs, dbResult.Error
}

// ListTemplateNamespace get page of template namespace list
func ListTemplateNamespace(query string, orders []string, offset int, limit int) ([]*TemplateNamespace, error) {
	templateNs := []*TemplateNamespace{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&templateNs).Error; err != nil {
		return nil, err
	}
	return templateNs, nil
}

// CountTemplateNamespace get total size of template namespace
func CountTemplateNamespace(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&TemplateNamespace{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count templateNs failed: %v", err)
		return 0, err
	}
	return count, nil
}

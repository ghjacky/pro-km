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
	db.RegisterDBTable(&Template{})
}

// Template 配置模板
type Template struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);uniqueIndex:Name_Namespace;not null" json:"name" description:"模板名称（必填）"`
	Namespace string `gorm:"type:varchar(100);uniqueIndex:Name_Namespace;not null" json:"namespace" description:"模板命名空间（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"模板描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group     string `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已创建(created), 已删除(deleted)"`

	LatestVersionID uint64 `gorm:"type:bigint;not null" json:"latest_version_id"`
	Data            string `gorm:"-" json:"data" description:"模板文件内容"`
}

// CreateTemplate create a new template
func CreateTemplate(template *Template, tx *gorm.DB) (*Template, error) {
	if tx == nil {
		tx = db.Get()
	}
	dbResult := tx.Create(template)

	return template, dbResult.Error
}

// ListTemplate get page of Template list
func ListTemplate(query string, orders []string, offset int, limit int) ([]*Template, error) {
	templates := []*Template{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// ListAllTemplate list all template
func ListAllTemplate() ([]*Template, error) {
	templates := []*Template{}

	if err := db.Get().Model(&Template{}).Where("status != 'deleted'").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// CountTemplates get total size of apps
func CountTemplates(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&Template{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count templates failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetTemplateByID get the template by id
func GetTemplateByID(id uint64) (*Template, error) {
	template := &Template{}
	db := db.Get().Model(&Template{}).Where("id = ? AND status != 'deleted'", id)
	if err := db.First(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

// GetTemplateByNameNamespace get a template by name and namespace
func GetTemplateByNameNamespace(name, ns string) (*Template, error) {
	template := &Template{}
	db := db.Get().Model(&Template{}).Where("name = ? AND namespace = ? AND status != 'deleted'", name, ns)
	if err := db.First(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

// UpdateTemplate update a template
func UpdateTemplate(template *Template, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&Template{}).Where(&Template{ID: template.ID}).Updates(template).Error
}

// DeleteTemplate soft delete app info
func DeleteTemplate(id uint64) error {
	template := &Template{}
	if err := db.Get().Model(template).Where("id = ?", id).First(template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update template name and deleteAt
	return db.Get().
		Model(&Template{}).
		Where("id = ?", id).
		Updates(&Template{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", template.Name, uuid.NewUUID()),
		}).Error
}

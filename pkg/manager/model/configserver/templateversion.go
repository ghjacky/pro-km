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
	db.RegisterDBTable(&TemplateVersion{})
}

// TemplateVersion 配置模板版本，真实的数据载体
type TemplateVersion struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_Version;not null" json:"name" description:"模板名称（必填）"`
	Namespace string `gorm:"type:varchar(100);uniqueIndex:Name_Namespace_Version;not null" json:"namespace" description:"模板命名空间（必填）"`
	Version   uint64 `gorm:"type:bigint;uniqueIndex:Name_Namespace_Version;not null" json:"version" description:"模板版本（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"配置描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Group     string `gorm:"type:varchar(100)" json:"group" description:"创建者所属团队（不填）"`
	Digest    string `gorm:"type:varchar(100);not null" json:"-" description:"配置的digest值"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已创建(created), 已删除(deleted)"`

	TemplateID uint64 `gorm:"type:bigint;not null" json:"-"`
}

// CreateTemplateVersion create a new templateVersion
func CreateTemplateVersion(templateVersion *TemplateVersion, tx *gorm.DB) (*TemplateVersion, error) {
	if tx == nil {
		tx = db.Get()
	}
	dbResult := tx.Create(templateVersion)

	return templateVersion, dbResult.Error
}

// ListTemplateVersion get page of TemplateVersion list
func ListTemplateVersion(query string, orders []string, offset int, limit int) ([]*TemplateVersion, error) {
	templateVersions := []*TemplateVersion{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&templateVersions).Error; err != nil {
		return nil, err
	}
	return templateVersions, nil
}

// CountTemplateVersions get total size of apps
func CountTemplateVersions(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&TemplateVersion{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count templateVersions failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetTemplateVersionByID get the the specific version of the template by id
func GetTemplateVersionByID(id uint64) (*TemplateVersion, error) {
	templateVersion := &TemplateVersion{}
	db := db.Get().Model(&TemplateVersion{}).Where("id = ? AND status != 'deleted'", id)
	if err := db.First(templateVersion).Error; err != nil {
		return nil, err
	}
	return templateVersion, nil
}

// GetTemplateVersionByTemplateIDAndVersion get the the specific version of the template version
func GetTemplateVersionByTemplateIDAndVersion(id, version uint64) (*TemplateVersion, error) {
	templateVersion := &TemplateVersion{}
	db := db.Get().Model(&TemplateVersion{}).Where("template_id = ? AND version = ?", id, version)
	if err := db.First(templateVersion).Error; err != nil {
		return nil, err
	}
	return templateVersion, nil
}

// UpdateTemplateVersion update a templateVersion
func UpdateTemplateVersion(templateVersion *TemplateVersion, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&TemplateVersion{}).Where(&TemplateVersion{ID: templateVersion.ID}).Updates(templateVersion).Error
}

// DeleteTemplateVersion soft delete app info
func DeleteTemplateVersion(id uint64) error {
	templateVersion := &TemplateVersion{}
	if err := db.Get().Model(templateVersion).Where("id = ?", id).First(templateVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update templateVersion name and deleteAt
	return db.Get().
		Model(&TemplateVersion{}).
		Where("id = ?", id).
		Updates(&TemplateVersion{
			DeletedAt: &now,
			Status:    "deleted",
			Name:      fmt.Sprintf("%s-%s", templateVersion.Name, uuid.NewUUID()),
		}).Error
}

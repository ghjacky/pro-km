package model

import (
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"gorm.io/gorm"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&Release{})
}

// Release release version
type Release struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Version   uint64 `gorm:"type:bigint;uniqueIndex:Version_ConfigInfoID;not null" json:"version" description:"发布版本（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"发布描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"配置状态: 已创建(created), 已删除(deleted)"`
	Digest    string `gorm:"type:varchar(100);not null" json:"-" description:"配置的digest值"`

	ConfigInfoID    uint64 `gorm:"type:bigint;uniqueIndex:Version_ConfigInfoID;not null" json:"-"`
	ConfigVersionID uint64 `gorm:"type:bigint;not null" json:"-"`
}

// CreateRelease create a new release
func CreateRelease(release *Release, tx *gorm.DB) (*Release, error) {
	if tx == nil {
		tx = db.Get()
	}
	dbResult := tx.Create(release)

	return release, dbResult.Error
}

// ListRelease get page of Release list
func ListRelease(query string, orders []string, offset int, limit int) ([]*Release, error) {
	releases := []*Release{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&releases).Error; err != nil {
		return nil, err
	}
	return releases, nil
}

// CountReleases get total size of apps
func CountReleases(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&Release{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count releases failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetReleaseByID get the the release version of the configInfo by id
func GetReleaseByID(id uint64) (*Release, error) {
	release := &Release{}
	db := db.Get().Model(&Release{}).Where("id = ? AND status != 'deleted'", id)
	if err := db.First(release).Error; err != nil {
		return nil, err
	}
	return release, nil
}

// GetReleaseByConfigInfoIDAndVersion get the the release version of the configInfo by config id and version
func GetReleaseByConfigInfoIDAndVersion(id uint64, version uint64) (*Release, error) {
	release := &Release{}
	db := db.Get().Model(&Release{}).Where("config_info_id = ? AND version = ? AND status != 'deleted'", id, version)
	if err := db.First(release).Error; err != nil {
		return nil, err
	}
	return release, nil
}

// UpdateRelease update a release
func UpdateRelease(release *Release, tx *gorm.DB) error {
	if tx == nil {
		tx = db.Get()
	}
	return tx.Model(&Release{}).Where(&Release{ID: release.ID}).Updates(release).Error
}

// DeleteRelease soft delete app info
func DeleteRelease(id uint64) error {
	release := &Release{}
	if err := db.Get().Model(release).Where("id = ?", id).First(release).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	now := time.Now()
	// update release name and deleteAt
	return db.Get().
		Model(&Release{}).
		Where("id = ?", id).
		Updates(&Release{
			DeletedAt: &now,
			Status:    "deleted",
		}).Error
}

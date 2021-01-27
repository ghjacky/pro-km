package model

import (
	"fmt"
	"strings"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"gorm.io/gorm"
)

func init() {
	db.RegisterDBTable(&Cluster{})
}

// Cluster Describe a cluster
type Cluster struct {
	ID          uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt   time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `sql:"index" json:"-"`
	HeartbeatAt *time.Time `json:"heartbeat_at" description:"最近心跳时间"`

	SiteID       string        `gorm:"type:varchar(100);unique;not null" json:"site_id" description:"项目ID（必填）"`
	Name         string        `gorm:"type:varchar(100);not null" json:"name" description:"项目名称（必填）"`
	Desc         string        `gorm:"type:varchar(512)" json:"desc" description:"应用描述，不超过200个汉字（选填）"`
	BusinessLine string        `gorm:"type:varchar(100);not null" json:"business_line" description:"业务线"`
	Group        string        `gorm:"type:varchar(100);not null" json:"group" description:"用户组，不填"`
	CreatedBy    string        `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status       ClusterStatus `gorm:"type:varchar(10);not null" json:"status" description:"状态"`
	Env          string        `gorm:"type:varchar(512)" json:"env" description:"环境变量json字串"`
	IP           string        `gorm:"type:varchar(100)" json:"ip" description:"ip地址"`
	SyncID       string        `gorm:"type:varchar(100)" json:"sync_id" description:"从外部平台同步数据的唯一ID"`
	ConnKey      string        `gorm:"type:varchar(100);not null" json:"-" description:"agent connection key"`
}

// ClusterStatus the type of cluster status
type ClusterStatus string

/** all cluster status */
const (
	ClusterStatusReady    ClusterStatus = "Ready"
	ClusterStatusNotReady ClusterStatus = "NotReady"
)

// CreateCluster create a new cluster
func CreateCluster(cluster *Cluster) (*Cluster, error) {
	dbResult := db.Get().Create(cluster)

	return cluster, dbResult.Error
}

// UpdateCluster update a cluster
func UpdateCluster(cluster *Cluster) error {
	return db.Get().Model(&cluster).Updates(cluster).Error
}

// UpdateClusterStatus update status of a cluster
func UpdateClusterStatus(connKey string, status ClusterStatus) error {
	return db.Get().Transaction(func(tx *gorm.DB) error {

		connInfos := strings.Split(connKey, apis.ConnectionSplit)
		if len(connInfos) != 2 {
			alog.Errorf("When connected, connection key is invalid: %v", connKey)
			return fmt.Errorf("invalid conn key")
		}

		siteID := connInfos[0]
		timeStamp := connInfos[1]

		cluster := &Cluster{}
		if err := tx.Model(&Cluster{}).Where("site_id = ? AND status != 'deleted'", siteID).First(cluster).Error; err != nil {
			alog.Errorf("When connected, get cluster err: %v", err)
			return err
		}
		if cluster.ConnKey == "" {
			if err := tx.Model(&Cluster{}).Where("site_id = ?", siteID).Updates(Cluster{ConnKey: connKey, Status: status}).Error; err != nil {
				alog.Errorf("When connected, update cluster %v error: %v", connKey, err)
				return err
			}
			return nil

		}
		oldConnInfos := strings.Split(cluster.ConnKey, apis.ConnectionSplit)
		if len(oldConnInfos) != 2 {
			alog.Errorf("When connected, old connection key is invalid: %v", cluster.ConnKey)
			return fmt.Errorf("invalid old conn key")
		}

		if strings.Compare(timeStamp, oldConnInfos[1]) > -1 {
			if err := tx.Model(&Cluster{}).Where("site_id = ?", siteID).Updates(Cluster{ConnKey: connKey, Status: status}).Error; err != nil {
				alog.Errorf("When connected, update cluster %v error: %v", connKey, err)
				return err
			}
		}
		return nil
	})
}

// ListCluster get page of AppInfo list
func ListCluster(query string, orders []string, offset int, limit int) ([]*Cluster, error) {
	clusters := []*Cluster{}
	db := db.Get().Where(query).Offset(offset).Limit(limit)
	for _, order := range orders {
		db = db.Order(order)
	}
	if err := db.Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

// CountCluster get total size of cluster
func CountCluster(query string) (int64, error) {
	var count int64
	if err := db.Get().Model(&Cluster{}).Where(query).Count(&count).Error; err != nil {
		alog.Errorf("Count Cluster failed: %v", err)
		return 0, err
	}
	return count, nil
}

// GetClusterBySiteID get the cluster by site id
func GetClusterBySiteID(siteID string) (*Cluster, error) {
	cluster := &Cluster{}
	db := db.Get().Model(&Cluster{}).Where("site_id = ? AND status != 'deleted'", siteID)
	if err := db.First(cluster).Error; err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteCluster delete a cluster
func DeleteCluster(id uint64) error {
	err := db.Get().First(&Cluster{}, "id = ?", id).Error
	if err != nil {
		return err
	}
	return db.Get().Where("id = ?", id).Delete(&Cluster{}).Error
}

// GetClusterEnvs get a cluster env
func GetClusterEnvs(cluster *Cluster) error {
	return db.Get().Model(Cluster{}).Select("env").Find(&cluster).Where("id = ?", cluster.ID).Error
}

// UpdateClusterEnvs get a cluster env
func UpdateClusterEnvs(cluster *Cluster) error {
	return db.Get().Model(Cluster{}).Where("id = ?", cluster.ID).Select("env").Update("env", cluster.Env).Error
}

// GetClusterEnvBySiteID get cluster env by site id
func GetClusterEnvBySiteID(siteID string) (string, error) {
	cluster := &Cluster{}
	dbResult := db.Get().Model(Cluster{}).Select("env").Where("site_id = ?", siteID).Find(&cluster)
	if dbResult.Error != nil {
		return "", dbResult.Error
	}

	return cluster.Env, nil
}

// RsyncClusterBySyncID create or update clusters by syncid
func RsyncClusterBySyncID(cluster *Cluster) error {
	mydata := &Cluster{
		SiteID: cluster.SiteID,
		Name:   cluster.Name,
		IP:     cluster.IP,
	}
	dbResult := db.Get().Where("sync_id = ?", cluster.SyncID).First(&cluster)
	if dbResult.Error != nil {
		if dbResult.Error == gorm.ErrRecordNotFound {
			results := db.Get().Create(&cluster)
			if results.Error != nil {
				return results.Error
			}
		}
	} else {
		results := db.Get().Model(&cluster).Where("sync_id = ?", cluster.SyncID).Updates(mydata)
		if results.Error != nil {
			return results.Error
		}
	}
	return nil
}

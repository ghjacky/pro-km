package model

import "time"

// Flow 流水线
type Flow struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);not null" json:"name" description:"流水线名称（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"流水线描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Phase     string `gorm:"type:text;" json:"phase" description:"phase表示的json串"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"流水线状态"`

	ClusterName string `gorm:"type:string;not null" json:"cluster_name" description:"集群项目ID"`
	ProductName string `gorm:"type:string;not null" json:"product_name" description:"产品名称"`
	AppName     string `gorm:"type:string;not null" json:"app_name" description:"应用名称"`
}

// FlowStatus the type of flow status
type FlowStatus string

/** all flow status */
const (
	FlowStatusCreated FlowStatus = "created"
	FlowStatusPending FlowStatus = "pending"
	FlowStatusRunning FlowStatus = "running"
	FlowStatusFailed  FlowStatus = "failed"
	FlowStatusSucceed FlowStatus = "succeed"
	FlowStatusDeleted FlowStatus = "deleted"
)

// Phase 定义流水线的各个阶段
type Phase struct {
}

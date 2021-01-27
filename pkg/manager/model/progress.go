package model

import (
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
)

func init() {
	db.RegisterDBTable(&ProgressInfo{})
	db.RegisterDBTable(&ProgressVersion{})
}

// ProgressInfo 进程信息，对应一个容器模板
type ProgressInfo struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string      `gorm:"type:varchar(100);not null" json:"name" description:"进程信息名称（必填）"`
	Code      string      `gorm:"type:varchar(100);" json:"code" description:"源码地址（选填）"`
	Command   StringArray `gorm:"type:varchar(200)" json:"command" description:"程序启动命令, 逗号隔开"`
	WorkDir   string      `gorm:"type:varchar(100);" json:"work_dir" description:"程序工作路径"`
	Env       StringArray `gorm:"type:varchar(200)" json:"env" description:"环境变量，逗号隔开"`
	CPU       float64     `gorm:"type:decimal(10,3) unsigned" json:"cpu" description:"cpu capacity of cluster, unit is core"`
	GPU       float64     `gorm:"type:decimal(10,0) unsigned" json:"gpu" description:"gpu capacity of cluster, unit is card"`
	Memory    float64     `gorm:"type:decimal(10,3) unsigned" json:"memory" description:"memory capacity of cluster, unit is GiB"`
	Volume    string      `gorm:"type:varchar(1000);" json:"volume" description:"volume的json串"`
	CreatedBy string      `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string      `gorm:"type:varchar(20)" json:"-" description:"进程信息状态: 已创建(created), 已删除(deleted)"`

	AppInfoID uint64 `gorm:"type:bigint;not null" json:"-" description:"应用ID"`
}

// ProgressVersion 进程版本信息，对应一个容器
type ProgressVersion struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string      `gorm:"type:varchar(100);not null" json:"name" description:"进程信息名称（必填）"`
	Code      string      `gorm:"type:varchar(100);" json:"code" description:"源码地址（选填）"`
	Commit    string      `gorm:"type:varchar(100);" json:"commit" description:"源码commit号（选填）"`
	Image     string      `gorm:"type:varchar(100);" json:"image" description:"源码镜像号（必填）"`
	Command   StringArray `gorm:"type:varchar(200)" json:"command" description:"程序启动命令, 逗号隔开"`
	WorkDir   string      `gorm:"type:varchar(100);" json:"work_dir" description:"程序工作路径"`
	Env       StringArray `gorm:"type:varchar(200)" json:"env" description:"环境变量，逗号隔开"`
	CPU       float64     `gorm:"type:decimal(10,3) unsigned" json:"cpu" description:"cpu capacity of cluster, unit is core"`
	GPU       float64     `gorm:"type:decimal(10,0) unsigned" json:"gpu" description:"gpu capacity of cluster, unit is card"`
	Memory    float64     `gorm:"type:decimal(10,3) unsigned" json:"memory" description:"memory capacity of cluster, unit is GiB"`
	Volume    string      `gorm:"type:varchar(1000);" json:"volume" description:"volume的json串"`
	CreatedBy string      `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string      `gorm:"type:varchar(20)" json:"-" description:"进程信息状态: 已创建(created), 已删除(deleted)"`

	AppID          uint64 `gorm:"type:bigint;not null" json:"-" description:"应用版本ID"`
	ProgressInfoID uint64 `gorm:"type:bigint;not null" json:"-" description:"进程信息ID"`
}

// Volume .
type Volume struct {
	Name      string
	MouthPath string
	HostPath  string
	Type      string
}

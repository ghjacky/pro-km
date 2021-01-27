package model

import "time"

// Product 产品
type Product struct {
	ID        uint64     `gorm:"primary_key" json:"id,omitempty" description:"唯一id（不填）"`
	CreatedAt time.Time  `json:"created_at,omitempty" description:"创建时间（不填）"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name      string `gorm:"type:varchar(100);unique_index:Name_Version;not null" json:"name" description:"产品名称（必填）"`
	Version   string `gorm:"type:varchar(100);unique_index:Name_Version;not null" json:"version" description:"产品版本（必填）"`
	Desc      string `gorm:"type:varchar(512)" json:"desc" description:"产品描述，不超过200个汉字（选填）"`
	CreatedBy string `gorm:"type:varchar(100);not null" json:"created_by" description:"创建者（不填）"`
	Status    string `gorm:"type:varchar(20)" json:"-" description:"Application状态: 已创建(created), 已删除(deleted)"`

	AppIDs    IntArray    `gorm:"type:varchar(1024)" json:"app_ids" description:"应用版本ID的集合"`
	DependOns StringArray `gorm:"type:varchar(200)" json:"depend_ons" description:"应用之间的依赖，[id1:id2,id2]"`
}

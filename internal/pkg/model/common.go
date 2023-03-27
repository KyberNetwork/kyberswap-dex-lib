package model

import "gorm.io/gorm"

type Common struct {
	Key   string `gorm:"primaryKey;type:varchar(255);auto_increment:false"`
	Value string `gorm:"column:value" sql:"type:json"`
}

func CommonTable() func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table("common")
	}
}

func (Common) TableName() string {
	return "common"
}

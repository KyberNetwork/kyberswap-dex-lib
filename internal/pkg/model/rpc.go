package model

import "gorm.io/gorm"

type RPC struct {
	URL       string `gorm:"primaryKey;type:varchar(256);column:url"`
	Block     uint64
	Status    bool
	Active    bool
	UpdatedAt int64 `gorm:"column:updatedAt"`
}

func RPCTable() func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table("rpcs")
	}
}

func (RPC) TableName() string {
	return "rpcs"
}

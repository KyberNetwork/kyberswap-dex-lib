package model

import (
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

const TokenKey = "tokens"

type Token struct {
	Address     string `gorm:"primaryKey;type:varchar(255);auto_increment:false" json:"address"`
	Symbol      string `gorm:"column:symbol;type:varchar(255)" json:"symbol"`
	Name        string `gorm:"column:name;type:varchar(255)" json:"name"`
	Decimals    uint8  `gorm:"column:decimals" json:"decimals"`
	CgkID       string `gorm:"column:cgkId;type:varchar(255)" json:"cgkId"`
	Type        string `gorm:"column:type;type:varchar(255)" json:"type"`
	PoolAddress string `gorm:"column:type;type:varchar(255)" json:"poolAddress"`
}

func (t Token) EncodeToken() string {
	return utils.Join(t.Symbol, t.Name, t.Decimals, t.CgkID, t.Type, t.PoolAddress)
}

func DecodeToken(key, member string) Token {
	var t Token
	split := strings.Split(member, ":")
	t.Address = key
	t.Symbol = split[0]
	t.Name = split[1]
	decimals, _ := strconv.ParseUint(split[2], 10, 64)
	t.Decimals = uint8(decimals)
	t.CgkID = split[3]
	t.Type = split[4]
	if len(split) >= 6 {
		t.PoolAddress = split[5]
	}
	return t
}
func TokenTable() func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table("tokens")
	}
}

func (Token) TableName() string {
	return "tokens"
}

package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const GormDataType = "json"
const PoolKey = "pools"
const WhiteListKey = "whitelist"
const PairKey = "pairs"
const Prices = "price"
const AmountKey = "amounts"
const AmplifiedTvlKey = "amplifiedTvl"
const PoolReserveKey = "pools:reserves"
const PoolTokenKey = "pools:tokens"
const UpdatePoolKey = "pools:update"
const ExchangesKey = "exchanges"
const Route = "route"
const CachePoint = "token"
const CacheRange = "usd"

type PoolReserves []string

func (j PoolReserves) Encode() string {
	return strings.Join(j, ":")
}

type PoolToken struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Decimals  uint   `json:"decimals"`
	Weight    uint   `json:"weight"`
	Swappable bool   `json:"swappable"`
	//PrecisionMultiplier string `json:"precisionMultiplier,omitempty"`
	//Rate                string `json:"rate,omitempty"`
}

type PoolTokens []*PoolToken

type Pool struct {
	Address      string       `gorm:"primaryKey;type:varchar(255);auto_increment:false" json:"-"`
	ReserveUsd   float64      `gorm:"column:reserveUsd" json:"reserveUsd,omitempty"`
	AmplifiedTvl float64      `json:"amplifiedTvl,omitempty"`
	SwapFee      float64      `gorm:"column:swapFee" json:"swapFee,omitempty"`
	Exchange     string       `gorm:"column:exchange;type:varchar(255);index:idx_exchange" json:"exchange,omitempty"`
	Type         string       `gorm:"column:type;type:varchar(255)" json:"type,omitempty"`
	Timestamp    int64        `gorm:"column:timestamp" json:"timestamp,omitempty"`
	Reserves     PoolReserves `gorm:"column:reserves" sql:"type:json" json:"reserves,omitempty"`
	Tokens       PoolTokens   `gorm:"column:tokens" sql:"type:json" json:"tokens,omitempty"`
	Extra        string       `gorm:"column:extra" json:"extra,omitempty"`
	StaticExtra  string       `gorm:"column:staticExtra" json:"staticExtra,omitempty"`
	TotalSupply  string       `json:"totalSupply,omitempty"`
}

func (p Pool) EncodePool() (string, error) {
	bytes, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func DecodePool(key, member string) (p Pool, err error) {
	err = json.Unmarshal([]byte(member), &p)
	if err != nil {
		return
	}
	p.Address = key
	return
}

func (p Pool) LpToken() string {

	var staticExtra = struct {
		LpToken string `json:"lpToken"`
	}{}
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if len(staticExtra.LpToken) > 0 {
		return strings.ToLower(staticExtra.LpToken)
	}
	return p.Address
}

func PoolTable() func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table("pools")
	}
}

func (Pool) TableName() string {
	return "pools"
}

func (p Pool) HasReserves() bool {
	if (len(p.Reserves)) == 0 {
		return false
	}
	for _, reserve := range p.Reserves {
		if len(reserve) == 0 || reserve == "0" {
			return false
		}
	}
	return true
}

func (p Pool) HasAmplifiedTvl() bool {
	return p.AmplifiedTvl != 0
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *PoolReserves) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	err := json.Unmarshal(bytes, j)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j PoolReserves) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (PoolReserves) GormDataType() string {
	return GormDataType
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *PoolTokens) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	err := json.Unmarshal(bytes, j)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j PoolTokens) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (PoolTokens) GormDataType() string {
	return GormDataType
}

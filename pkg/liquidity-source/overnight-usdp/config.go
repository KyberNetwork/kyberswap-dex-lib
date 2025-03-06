package overnightusdp

import (
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	DexID    string         `json:"dexID"`
	Exchange string         `json:"exchange"`
	Usdc     common.Address `json:"usdc"`
	UsdPlus  common.Address `json:"usdPlus"`
}

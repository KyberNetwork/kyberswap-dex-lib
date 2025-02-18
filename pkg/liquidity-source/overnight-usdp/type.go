package overnightusdp

import "math/big"

type PoolMeta struct {
	Asset       string `json:"asset"`
	UsdPlus     string `json:"usdPlus"`
	BlockNumber uint64 `json:"blockNumber"`
}

type Extra struct {
	IsPaused  bool     `json:"isPaused"`
	BuyFee    *big.Int `json:"buyFee"`
	RedeemFee *big.Int `json:"redeemFee"`
}

type StaticExtra struct {
	AssetDecimals   int64 `json:"assetDecimals"`
	UsdPlusDecimals int64 `json:"usdPlusDecimals"`
}

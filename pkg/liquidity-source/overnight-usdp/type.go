package overnightusdp

import "math/big"

type PoolMeta struct {
	Exchange    string `json:"exchange"`
	BlockNumber uint64 `json:"blockNumber"`
}

type Extra struct {
	IsPaused  bool     `json:"isPaused"`
	BuyFee    *big.Int `json:"buyFee"`
	RedeemFee *big.Int `json:"redeemFee"`
}

type StaticExtra struct {
	Exchange        string `json:"exchange"`
	AssetDecimals   int64  `json:"assetDecimals"`
	UsdPlusDecimals int64  `json:"usdPlusDecimals"`
}

package ethervista

import "math/big"

type RPCStateData struct {
	Reserve0             *big.Int
	Reserve1             *big.Int
	RouterAddress        string
	BuyTotalFee          uint
	USDCToETHBuyTotalFee *big.Int
}

type Extra struct {
	RouterAddress        string   `json:"routerAddress"`
	BuyTotalFee          uint     `json:"buyTotalFee"`
	USDCToETHBuyTotalFee *big.Int `json:"usdcToETHBuyTotalFee"`
}

type PoolMeta struct {
	RouterAddress string `json:"routerAddress"`
	BlockNumber   uint64 `json:"blockNumber"`
}

package ethervista

import "math/big"

type RPCStateData struct {
	Reserve0              *big.Int
	Reserve1              *big.Int
	RouterAddress         string
	BuyTotalFee           uint
	SellTotalFee          uint
	USDCToETHBuyTotalFee  *big.Int
	USDCToETHSellTotalFee *big.Int
}

type Extra struct {
	RouterAddress         string   `json:"routerAddress"`
	BuyTotalFee           uint     `json:"buyTotalFee"`
	SellTotalFee          uint     `json:"sellTotalFee"`
	USDCToETHBuyTotalFee  *big.Int `json:"usdcToETHBuyTotalFee"`
	USDCToETHSellTotalFee *big.Int `json:"usdcToETHSellTotalFee"`
}

type PoolMeta struct {
	RouterAddress   string `json:"routerAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type Gas struct {
	Swap int64
}

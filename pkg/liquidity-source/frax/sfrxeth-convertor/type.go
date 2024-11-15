package sfrxeth_convertor

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolItem struct {
	FrxETHAddress  string `json:"frxETH"`
	SfrxETHAddress string `json:"sfrxETH"`
}

type Gas struct {
	Deposit int64
	Redeem  int64
}

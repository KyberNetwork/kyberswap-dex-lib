package sfrxeth

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	SubmitPaused bool `json:"submitPaused"`
}

type Gas struct {
	SubmitAndDeposit int64
}

type PoolItem struct {
	FrxETHMinterAddress string `json:"frxETHMinter"`
	FrxETHAddress       string `json:"frxETH"`
	SfrxETHAddress      string `json:"sfrxETH"`
}

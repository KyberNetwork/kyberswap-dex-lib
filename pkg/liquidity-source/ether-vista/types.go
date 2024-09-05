package ethervista

import "math/big"

type ReserveData struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Extra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

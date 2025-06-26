package v2

import "math/big"

type Extra struct {
	ExchangeRateStored *big.Int `json:"exchangeRateStored"`
}

type PoolMeta struct {
	BlockNumber uint64
}

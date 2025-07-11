package v2

import "math/big"

type Extra struct {
	ExchangeRateStored *big.Int `json:"exchangeRateStored,omitempty"`
	IsMintPaused       bool     `json:"isMintPaused,omitempty"`
}

type PoolMeta struct {
	BlockNumber uint64
}

package ringswap

import "math/big"

type ReserveData struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Extra struct {
	Fee           uint64 `json:"fee"`
	FeePrecision  uint64 `json:"feePrecision"`
	WrappedToken0 string `json:"wrappedToken0"`
	WrappedToken1 string `json:"wrappedToken1"`
}

type SwapInfo struct {
	WrappedTokenIn  string `json:"wrappedTokenIn"`
	WrappedTokenOut string `json:"wrappedTokenOut"`
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}

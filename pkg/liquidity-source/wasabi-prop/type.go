package wasabiprop

import "math/big"

type Extra struct {
	Samples [][][2]*big.Int `json:"samples"` // [tokenInIndex][]{amountIn, amountOut}
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	BlockNumber   uint64 `json:"blockNumber"`
	RouterAddress string `json:"routerAddress"`
}

type getReservesResult struct {
	BaseTokenReserves  *big.Int `json:"baseTokenReserves"`
	QuoteTokenReserves *big.Int `json:"quoteTokenReserves"`
}

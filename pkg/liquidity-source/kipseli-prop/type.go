package kipseliprop

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

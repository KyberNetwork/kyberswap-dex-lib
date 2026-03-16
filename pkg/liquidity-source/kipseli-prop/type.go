package kipseliprop

import "math/big"

type Extra struct {
	Samples [][][2]*big.Int `json:"samples"`        // [tokenInIndex][]{amountIn, amountOut}
	Caps    []*big.Int      `json:"caps,omitempty"` // per-token reserve caps, same order as pool tokens
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	BlockNumber   uint64 `json:"blockNumber"`
	RouterAddress string `json:"routerAddress"`
}

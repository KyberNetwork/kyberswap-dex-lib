package uniswapv4

import "github.com/ethereum/go-ethereum/common"

type SubgraphPool struct {
	ID             string `json:"id"`
	PoolId         string `json:"poolId"`
	Currency0      string `json:"currency0"`
	Currency1      string `json:"currency1"`
	Fee            int    `json:"fee"`
	TickSpacing    int    `json:"tickSpacing"`
	Hooks          string `json:"hooks"`
	BlockTimestamp string `json:"blockTimestamp"`
}

type StaticExtra struct {
	PoolId      string `json:"poolId"`
	Currency0   string `json:"currency0"`
	Currency1   string `json:"currency1"`
	Fee         int    `json:"fee"`
	TickSpacing int    `json:"tickSpacing"`

	HooksAddress           common.Address `json:"hooksAddress"`
	UniversalRouterAddress common.Address `json:"universalRouterAddress"`
	Permit2Address         common.Address `json:"permit2Address"`
	Multicall3Address      common.Address `json:"multicall3Address"`
	NativeTokenAddress     common.Address `json:"nativeTokenAddress"`
}

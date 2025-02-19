package uniswapv4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Token struct {
	ID       string `json:"id"`
	Decimals string `json:"decimals"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
}

type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         uint32
	TickSpacing int32
	Hooks       common.Address
}

type SubgraphPool struct {
	ID                 string `json:"id"`
	Token0             Token  `json:"token0"`
	Token1             Token  `json:"token1"`
	Fee                string `json:"feeTier"`
	TickSpacing        string `json:"tickSpacing"`
	Hooks              string `json:"hooks"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}

type StaticExtra struct {
	PoolId       string         `json:"poolId"`
	Currency0    string         `json:"currency0"`
	Currency1    string         `json:"currency1"`
	Fee          int            `json:"fee"`
	TickSpacing  int            `json:"tickSpacing"`
	HooksAddress common.Address `json:"hooksAddress"`

	UniversalRouterAddress common.Address `json:"universalRouterAddress"`
	Permit2Address         common.Address `json:"permit2Address"`
	Multicall3Address      common.Address `json:"multicall3Address"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64   `json:"tickSpacing"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`
}

type Slot0Data struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	ProtocolFee  *big.Int `json:"protocolFee"`
	LpFee        *big.Int `json:"lpFee"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int  `json:"liquidity"`
	Slot0       Slot0Data `json:"slot0"`
	TickSpacing int       `json:"tickSpacing"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type PoolMetaInfo struct {
	Router      common.Address `json:"router"`
	Permit2Addr common.Address `json:"permit2Addr"`
	TokenIn     common.Address `json:"tokenIn"`
	TokenOut    common.Address `json:"tokenOut"`
	Fee         int            `json:"fee"`
	TickSpacing int            `json:"tickSpacing"`
	HookAddress common.Address `json:"hookAddress"`
	HookData    []byte         `json:"hookData"`
}

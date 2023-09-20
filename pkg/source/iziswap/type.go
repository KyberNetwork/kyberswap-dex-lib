package iziswap

import (
	"math/big"

	"github.com/izumiFinance/iZiSwap-SDK-go/swap"
)

type PoolInfo struct {
	Fee            int    `json:"fee"`
	TokenX         string `json:"tokenX"`
	TokenY         string `json:"tokenY"`
	Address        string `json:"address"`
	Timestamp      int    `json:"timestamp"`
	TokenXAddress  string `json:"tokenX_address"`
	TokenYAddress  string `json:"tokenY_address"`
	TokenXDecimals int    `json:"tokenX_decimals"`
	TokenYDecimals int    `json:"tokenY_decimals"`
	Version        string `json:"version"`
}

type ListPoolsParams struct {
	ChainId int
	// v1 or v2
	Version string
	// timestamp in second
	TimeStart int
	// response size
	Limit int
}

type ListPoolsResponse struct {
	Data  []PoolInfo `json:"data,omitempty"`
	Total int        `json:"total"`
}

type State struct {
	SqrtPrice_96            *big.Int `abi:"sqrtPrice_96"`
	CurrentPoint            *big.Int `abi:"currentPoint"`
	ObservationCurrentIndex uint16   `abi:"observationCurrentIndex"`
	ObservationQueueLen     uint16   `abi:"observationQueueLen"`
	ObservationNextQueueLen uint16   `abi:"observationNextQueueLen"`
	Locked                  bool     `abi:"locked"`
	Liquidity               *big.Int `abi:"liquidity"`
	LiquidityX              *big.Int `abi:"liquidityX"`
}

type Extra swap.PoolInfo

type FetchRPCResult struct {
	state    State
	reserve0 *big.Int
	reserve1 *big.Int
}

type LimitOrder struct {
	SellingX *big.Int `abi:"sellingX"`
	EarnY    *big.Int `abi:"earnY"`
	AccEarnY *big.Int `abi:"accEarnY"`

	SellingY *big.Int `abi:"sellingY"`
	EarnX    *big.Int `abi:"earnX"`
	AccEarnX *big.Int `abi:"accEarnX"`
}

type iZiSwapInfo struct {
	nextPoint      int
	nextLiquidity  *big.Int
	nextLiquidityX *big.Int
}

type Metadata struct {
	// a unix-timestamp counted in Second
	LastCreatedAtTimestamp int `json:"lastCreatedAtTimestamp"`
}

type Meta struct {
	LimitPoint int `json:"limitPoint"`
}

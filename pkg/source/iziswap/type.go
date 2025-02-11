package iziswap

import (
	"math/big"

	"github.com/KyberNetwork/iZiSwap-SDK-go/swap"

	iziswapclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/client"
)

type PoolInfo = iziswapclient.PoolInfo
type ListPoolsParams = iziswapclient.ListPoolsParams
type ListPoolsResponse = iziswapclient.ListPoolsResponse

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

type Gas struct {
	Swap int64
}

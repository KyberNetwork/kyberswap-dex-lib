package iziswap

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/swap"

	iziswapclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/client"
)

type PoolInfo = iziswapclient.PoolInfo
type ListPoolsParams = iziswapclient.ListPoolsParams
type ListPoolsResponse = iziswapclient.ListPoolsResponse

type State struct {
	SqrtPrice_96            *uint256.Int `abi:"sqrtPrice_96"`
	CurrentPoint            *big.Int     `abi:"currentPoint"`
	ObservationCurrentIndex uint16       `abi:"observationCurrentIndex"`
	ObservationQueueLen     uint16       `abi:"observationQueueLen"`
	ObservationNextQueueLen uint16       `abi:"observationNextQueueLen"`
	Locked                  bool         `abi:"locked"`
	Liquidity               *uint256.Int `abi:"liquidity"`
	LiquidityX              *uint256.Int `abi:"liquidityX"`
}

type Extra = swap.PoolInfo
type ExtraU256 = swap.PoolInfoU256

type FetchRPCResult struct {
	state       State
	reserve0    *uint256.Int
	reserve1    *uint256.Int
	blockNumber *big.Int
}

type LimitOrder struct {
	SellingX *uint256.Int `abi:"sellingX"`
	EarnY    *uint256.Int `abi:"earnY"`
	AccEarnY *uint256.Int `abi:"accEarnY"`

	SellingY *uint256.Int `abi:"sellingY"`
	EarnX    *uint256.Int `abi:"earnX"`
	AccEarnX *uint256.Int `abi:"accEarnX"`
}

type iZiSwapInfo struct {
	nextPoint      int
	nextLiquidity  *uint256.Int
	nextLiquidityX *uint256.Int
}

type Metadata struct {
	// a unix-timestamp counted in Second
	LastCreatedAtTimestamp int `json:"lastCreatedAtTimestamp"`
}

type Meta struct {
	LimitPoint  int    `json:"limitPoint"`
	BlockNumber uint64 `json:"blockNumber"`
}

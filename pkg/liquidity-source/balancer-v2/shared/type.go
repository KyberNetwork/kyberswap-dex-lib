package shared

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

type IBasePool interface {
	pool.IPoolSimulator

	GetPoolId() string
	OnJoin(tokenIn string, amountIn *uint256.Int) (*uint256.Int, error)
	OnExit(tokenOut string, amountIn *uint256.Int) (*uint256.Int, error)
	OnSwap(tokenIn, tokenOut string, amountIn *uint256.Int) (*uint256.Int, error)
}

type SwapInfo struct {
	Hops []Hop `json:"hops,omitempty"`
}

type Hop struct {
	PoolId        string
	Pool          string
	TokenIn       string
	TokenOut      string
	AmountIn      *big.Int
	AmountOut     *big.Int
	JoinExitIndex *big.Int `json:"joinExitIndex,omitempty"`
}

// indexes of the pools to exit or join in ascending order,
// each value is a packed uint256 with the following structure [kind(uint24) 0 for exiting pool 1 for joining pool, pool index(uint232)]
func PackJoinExitIndex(kind JoinExitKind, poolIndex int) *big.Int {
	kindBig := big.NewInt(int64(kind))

	kindBig.Lsh(kindBig, 232) // shift kind to the top 24 bits

	tokenIndexBig := big.NewInt(int64(poolIndex))
	return new(big.Int).Or(kindBig, tokenIndexBig)
}

package shared

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type IBasePool interface {
	pool.IPoolSimulator

	GetPoolId() string
}

type SwapInfo struct {
	Hops []Hop `json:"hops,omitempty"`
}

type Hop struct {
	PoolId    string
	Pool      string
	TokenIn   string
	TokenOut  string
	AmountIn  *big.Int
	AmountOut *big.Int
}

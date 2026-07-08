package caliberprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Address string `json:"a"`
}

type Extra struct {
	Ladders [2][]LadderPoint `json:"l"`
}

type LadderPoint struct {
	AmountIn  *uint256.Int `json:"in"`
	AmountOut *uint256.Int `json:"out"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
	Address     string `json:"a"`
}

type quoteCallArg struct {
	PairId   [32]byte
	TokenIn  common.Address
	TokenOut common.Address
	AmountIn *big.Int
}

type quoteCallResult struct {
	AmountOut *big.Int
	Success   bool
}

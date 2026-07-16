package caliberprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	Address string `json:"a"`
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

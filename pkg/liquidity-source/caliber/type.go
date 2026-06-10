package caliber

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Contract string `json:"contract"`
	PairID   string `json:"pairId"`
}

type Extra struct {
	Unquoteable bool `json:"unquoteable,omitempty"`

	Ladder0 []LadderPoint `json:"ladder0"`
	Ladder1 []LadderPoint `json:"ladder1"`
}

type LadderPoint struct {
	AmountIn  *uint256.Int `json:"in"`
	AmountOut *uint256.Int `json:"out"`
}

type SwapInfo struct {
	Reserve0 *uint256.Int `json:"r0"`
	Reserve1 *uint256.Int `json:"r1"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
	Contract    string `json:"contract"`
	PairID      string `json:"pairId"`
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

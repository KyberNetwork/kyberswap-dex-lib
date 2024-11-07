package balancerv1

import "github.com/holiman/uint256"

type (
	PoolExtra struct {
		Records    map[string]Record `json:"records"`
		PublicSwap bool              `json:"publicSwap"`
		SwapFee    *uint256.Int      `json:"swapFee"`
	}

	Record struct {
		Bound   bool         `json:"bound"`
		Denorm  *uint256.Int `json:"denorm"`
		Balance *uint256.Int `json:"balance"`
	}

	PoolMeta struct {
		BlockNumber uint64
	}

	Gas struct {
		SwapExactAmountIn int64
	}

	PoolData struct {
		Tokens       []string
		SwapFee      *uint256.Int
		Records      map[string]Record
		IsPublicSwap bool
	}
)

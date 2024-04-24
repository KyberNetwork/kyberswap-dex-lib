//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Record Gas
//msgp:ignore PoolExtra PoolMeta PoolData
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256

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

//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple StaticExtra Extra
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim uint256.Int as:[]byte using:msgpencode.EncodeUint256NonPtr/msgpencode.DecodeUint256NonPtr

package stableng

import "github.com/holiman/uint256"

type (
	StaticExtra struct {
		APrecision          *uint256.Int
		OffpegFeeMultiplier *uint256.Int
		// which coins are originally native (before being converted to wrapped)
		IsNativeCoins []bool
	}

	Extra struct {
		InitialA     *uint256.Int
		FutureA      *uint256.Int
		InitialATime int64
		FutureATime  int64
		SwapFee      *uint256.Int
		AdminFee     *uint256.Int

		RateMultipliers []uint256.Int `json:",omitempty"`
	}
)

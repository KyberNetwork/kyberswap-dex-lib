//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple StaticExtra Extra
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim uint256.Int as:[]byte using:msgpencode.EncodeUint256NonPtr/msgpencode.DecodeUint256NonPtr

package plain

import "github.com/holiman/uint256"

type (
	StaticExtra struct {
		APrecision *uint256.Int
		LpToken    string

		// some Plain pools have oracle for 2nd coin's rate
		// (not to be confused with Plain2Price that use multiple oracles for all of its coins)
		Oracle string `json:",omitempty"`

		// which coins are originally native (before being converted to wrapped)
		IsNativeCoin []bool
	}

	Extra struct {
		InitialA     *uint256.Int
		FutureA      *uint256.Int
		InitialATime int64
		FutureATime  int64
		SwapFee      *uint256.Int
		AdminFee     *uint256.Int

		// some Plain pools have non-standard rates
		// for example https://etherscan.io/address/0xA96A65c051bF88B4095Ee1f2451C2A9d43F53Ae2#readContract
		// or https://etherscan.io/address/0xF9440930043eb3997fc70e1339dBb11F341de7A8#readContract
		// or Plain2Price, or old plain-oracle pools
		// those rates are not fixed (10^(36-decimal)) like other Plain pools so need to be put in Extra
		// note that this will be empty for standard rates pool (will be init by NewSimulator)
		RateMultipliers []uint256.Int `json:",omitempty"`
	}
)

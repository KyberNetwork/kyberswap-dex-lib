//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Factory Gas
//msgp:ignore Metadata StaticExtra Extra Pair Meta
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

package camelot

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Metadata struct {
	Offset uint64
}

type StaticExtra struct {
	FeeDenominator *big.Int `json:"feeDenominator"`
}
type Factory struct {
	FeeTo         common.Address `json:"feeTo"`
	OwnerFeeShare *big.Int       `json:"ownerFeeShare"`
}

type Extra struct {
	StableSwap           bool     `json:"stableSwap"`
	Token0FeePercent     *big.Int `json:"token0FeePercent"`
	Token1FeePercent     *big.Int `json:"token1FeePercent"`
	PrecisionMultiplier0 *big.Int `json:"precisionMultiplier0"`
	PrecisionMultiplier1 *big.Int `json:"precisionMultiplier1"`

	Factory *Factory `json:"factory"`
}

type Pair struct {
	Reserve0             *big.Int
	Reserve1             *big.Int
	StableSwap           bool
	Token0FeePercent     uint16
	Token1FeePercent     uint16
	PrecisionMultiplier0 *big.Int
	PrecisionMultiplier1 *big.Int
}

type Gas struct {
	Swap int64
}

type Meta struct {
	SwapFee      uint32 `json:"swapFee"`
	FeePrecision uint32 `json:"feePrecision"`
}

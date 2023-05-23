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

	Factory *Factory
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

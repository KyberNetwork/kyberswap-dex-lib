package dexT1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	CollateralReserves CollateralReserves
	DebtReserves       DebtReserves
	DexLimits          DexLimits
}

type CollateralReserves struct {
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

type DebtReserves struct {
	Token0Debt              *big.Int `json:"token0Debt"`
	Token1Debt              *big.Int `json:"token1Debt"`
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

type TokenLimit struct {
	Available      *big.Int `json:"available"`      // maximum available swap amount
	ExpandsTo      *big.Int `json:"expandsTo"`      // maximum amount the available swap amount expands to
	ExpandDuration *big.Int `json:"expandDuration"` // duration for `available` to grow to `expandsTo`
}

type DexLimits struct {
	WithdrawableToken0 TokenLimit `json:"withdrawableToken0"`
	WithdrawableToken1 TokenLimit `json:"withdrawableToken1"`
	BorrowableToken0   TokenLimit `json:"borrowableToken0"`
	BorrowableToken1   TokenLimit `json:"borrowableToken1"`
}

type PoolWithReserves struct {
	PoolAddress        common.Address     `json:"poolAddress"`
	Token0Address      common.Address     `json:"token0Address"`
	Token1Address      common.Address     `json:"token1Address"`
	Fee                *big.Int           `json:"fee"`
	CollateralReserves CollateralReserves `json:"collateralReserves"`
	DebtReserves       DebtReserves       `json:"debtReserves"`
	Limits             DexLimits          `json:"limits"`
}

type Gas struct {
	Swap int64
}

type StaticExtra struct {
	DexReservesResolver string `json:"dexReservesResolver"`
	HasNative           bool   `json:"hasNative"`
}

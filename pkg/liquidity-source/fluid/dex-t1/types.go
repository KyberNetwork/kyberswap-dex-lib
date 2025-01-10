package dexT1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	CollateralReserves       CollateralReserves
	DebtReserves             DebtReserves
	IsSwapAndArbitragePaused bool
	DexLimits                DexLimits
	CenterPrice              *big.Int
}

type CollateralReserves struct {
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

func (r CollateralReserves) Clone() CollateralReserves {
	return CollateralReserves{
		Token0RealReserves:      new(big.Int).Set(r.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(r.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(r.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(r.Token1ImaginaryReserves),
	}
}

type DebtReserves struct {
	Token0Debt              *big.Int `json:"token0Debt"`
	Token1Debt              *big.Int `json:"token1Debt"`
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

func (r DebtReserves) Clone() DebtReserves {
	return DebtReserves{
		Token0Debt:              new(big.Int).Set(r.Token0Debt),
		Token1Debt:              new(big.Int).Set(r.Token1Debt),
		Token0RealReserves:      new(big.Int).Set(r.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(r.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(r.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(r.Token1ImaginaryReserves),
	}
}

type TokenLimit struct {
	Available      *big.Int `json:"available"`      // maximum available swap amount
	ExpandsTo      *big.Int `json:"expandsTo"`      // maximum amount the available swap amount expands to
	ExpandDuration *big.Int `json:"expandDuration"` // duration for `available` to grow to `expandsTo`
}

func (t TokenLimit) Clone() TokenLimit {
	return TokenLimit{
		Available:      new(big.Int).Set(t.Available),
		ExpandsTo:      new(big.Int).Set(t.ExpandsTo),
		ExpandDuration: new(big.Int).Set(t.ExpandDuration),
	}
}

type DexLimits struct {
	WithdrawableToken0 TokenLimit `json:"withdrawableToken0"`
	WithdrawableToken1 TokenLimit `json:"withdrawableToken1"`
	BorrowableToken0   TokenLimit `json:"borrowableToken0"`
	BorrowableToken1   TokenLimit `json:"borrowableToken1"`
}

func (d DexLimits) Clone() DexLimits {
	return DexLimits{
		WithdrawableToken0: d.WithdrawableToken0.Clone(),
		WithdrawableToken1: d.WithdrawableToken1.Clone(),
		BorrowableToken0:   d.BorrowableToken0.Clone(),
		BorrowableToken1:   d.BorrowableToken1.Clone(),
	}
}

type PoolWithReserves struct {
	PoolAddress        common.Address     `json:"poolAddress"`
	Token0Address      common.Address     `json:"token0Address"`
	Token1Address      common.Address     `json:"token1Address"`
	Fee                *big.Int           `json:"fee"`
	CenterPrice        *big.Int           `json:"centerPrice"`
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

type SwapInfo struct {
	HasNative             bool               `json:"hasNative"`
	NewCollateralReserves CollateralReserves `json:"-"`
	NewDebtReserves       DebtReserves       `json:"-"`
	NewDexLimits          DexLimits          `json:"-"`
}

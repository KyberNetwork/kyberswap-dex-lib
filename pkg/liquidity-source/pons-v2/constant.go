package ponsv2

import "errors"

const (
	DexType = "pons-v2"

	// basisPoints is the fee/tax denominator used throughout PonsV2BondingCurve
	// and PonsV2BondingCurveMath (BASIS_POINTS in the Solidity source).
	basisPoints = 10_000
	// maxTotalTradeFeeBps mirrors PonsV2BondingCurve.MAX_TOTAL_TRADE_FEE_BPS,
	// the on-chain cap enforced at curve construction time (feeBps +
	// creatorTaxBps can never exceed this on a real curve; checked
	// defensively here rather than trusted blindly).
	maxTotalTradeFeeBps = 2_000

	// buyGas/sellGas are rough estimates pending a live gas-metered
	// transaction trace (no Tenderly/vnet access was available when this
	// package was written); dex-verify should replace these with measured
	// values.
	buyGas  = 150_000
	sellGas = 120_000

	curveMethodGetReserves = "getReserves"
	curveMethodFeeBps      = "feeBps"
	curveMethodCreatorTax  = "creatorTaxBps"
	curveMethodIsNative    = "isNativeQuote"
	curveMethodGraduated   = "graduated"
	curveMethodReserved    = "reservedTokens"
)

var (
	ErrPoolGraduated            = errors.New("curve graduated, no longer tradeable via bonding curve")
	ErrInvalidToken             = errors.New("invalid token")
	ErrZeroAmount               = errors.New("zero amount")
	ErrInsufficientInputAmount  = errors.New("insufficient input amount")
	ErrInsufficientOutputAmount = errors.New("insufficient output amount")
	ErrInsufficientLiquidity    = errors.New("insufficient liquidity")
	ErrOverflow                 = errors.New("overflow")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidFeeConfig         = errors.New("feeBps + creatorTaxBps exceeds basis points")
)

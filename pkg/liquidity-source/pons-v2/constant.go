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

	// buyGas/sellGas are measured gas_used from live buy()/sell() calls
	// against the reference curve on a Tenderly fork of Robinhood chain
	// (173,801 and 111,589 respectively), rounded up slightly for margin.
	buyGas  = 180_000
	sellGas = 115_000

	// defaultMaxBlockRangePerScan is used when Config.MaxBlockRangePerScan
	// is unset (0). See Config.MaxBlockRangePerScan's doc.
	defaultMaxBlockRangePerScan = 5_000

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

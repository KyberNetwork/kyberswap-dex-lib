package synthereum

import (
	"errors"
)

const (
	DexType = "synthereum"

	// pool types
	poolTypeMultiLP = "multi-lp"
	poolTypeWrapper = "wrapper"

	reserveZero = "0"
	// placeholder reserve for the synthetic side of the fixed-rate wrapper (wrap capacity is unbounded)
	defaultSynthReserve = "100000000000000000000000000"

	// SynthereumMultiLpLiquidityPool methods
	poolMethodGetMintTradeInfo     = "getMintTradeInfo"
	poolMethodGetRedeemTradeInfo   = "getRedeemTradeInfo"
	poolMethodMaxTokensCapacity    = "maxTokensCapacity"
	poolMethodTotalSyntheticTokens = "totalSyntheticTokens"
	poolMethodFeePercentage        = "feePercentage"

	// ERC4626 vault methods (Morpho vault holding the wrapper's collateral)
	vaultMethodBalanceOf     = "balanceOf"
	vaultMethodPreviewRedeem = "previewRedeem"
)

var defaultGas = Gas{
	Mint:   450000,
	Redeem: 450000,
	Wrap:   300000,
	Unwrap: 300000,
}

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrUnsupportedPoolType     = errors.New("unsupported pool type")
	ErrTradeUnavailable        = errors.New("trade info unavailable")
	ErrExceedsMaxCapacity      = errors.New("exceeds max synthetic tokens capacity")
	ErrExceedsRedeemCapacity   = errors.New("exceeds total synthetic tokens available for redeem")
	ErrInsufficientWrapReserve = errors.New("insufficient wrapper collateral reserve")
	ErrZeroAmountOut           = errors.New("zero amount out")
	ErrOverflow                = errors.New("overflow")
)

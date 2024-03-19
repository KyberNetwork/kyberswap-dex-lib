package tricryptong

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "curve-tricrypto-ng"

	poolMethodD                   = "D"
	poolMethodFeeGamma            = "fee_gamma"
	poolMethodMidFee              = "mid_fee"
	poolMethodOutFee              = "out_fee"
	poolMethodInitialAGamma       = "initial_A_gamma"
	poolMethodInitialAGammaTime   = "initial_A_gamma_time"
	poolMethodFutureAGamma        = "future_A_gamma"
	poolMethodFutureAGammaTime    = "future_A_gamma_time"
	poolMethodLastPricesTimestamp = "last_prices_timestamp"
	poolMethodXcpProfit           = "xcp_profit"
	poolMethodVirtualPrice        = "virtual_price"
	poolMethodAllowedExtraProfit  = "allowed_extra_profit"
	poolMethodAdjustmentStep      = "adjustment_step"
	poolMethodMaTime              = "ma_time"
	poolMethodBalances            = "balances"
	poolMethodPriceScale          = "price_scale"
	poolMethodPriceOracle         = "price_oracle"
	poolMethodLastPrices          = "last_prices"

	MaxLoopLimit = 256
)

var (
	Precision      = uint256.MustFromDecimal("1000000000000000000")
	FeeDenominator = uint256.MustFromDecimal("10000000000")

	PriceMask = uint256.MustFromHex("0xffffffffffffffffffffffffffffffff")
)

var (
	ErrInvalidReserve               = errors.New("invalid reserve")
	ErrInvalidStoredRates           = errors.New("invalid stored rates")
	ErrInvalidNumToken              = errors.New("invalid number of token")
	ErrInvalidAValue                = errors.New("invalid A value")
	ErrZero                         = errors.New("zero")
	ErrBalancesMustMatchMultipliers = errors.New("balances must match multipliers")
	ErrDDoesNotConverge             = errors.New("d does not converge")
	ErrTokenFromEqualsTokenTo       = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange       = errors.New("token index out of range")
	ErrAmountOutNotConverge         = errors.New("approximation did not converge")
)

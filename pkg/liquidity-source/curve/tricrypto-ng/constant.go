package tricryptong

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

const (
	DexType = "curve-tricrypto-ng"

	DefaultGas = 240000

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
	NumTokens    = 3
)

var (
	Precision = uint256.MustFromDecimal("1000000000000000000")

	PriceMask = uint256.MustFromHex("0xffffffffffffffffffffffffffffffff")

	TenPow10 = uint256.MustFromDecimal("10000000000")
	TenPow14 = uint256.MustFromDecimal("100000000000000")
	TwoE18   = uint256.MustFromDecimal("2000000000000000000")

	AMultiplier = uint256.MustFromDecimal("10000")
	MinGamma    = uint256.MustFromDecimal("10000000000")
	MaxGamma    = uint256.MustFromDecimal("50000000000000000")
	MinA        = number.Div(number.Mul(uint256.NewInt(27), AMultiplier), uint256.NewInt(100)) // 27 = NCoins ** NCoins, NCoins = 3
	MaxA        = number.Mul(number.Mul(uint256.NewInt(27), AMultiplier), uint256.NewInt(1000))
	MinD        = uint256.MustFromDecimal("100000000000000000")
	MaxD        = uint256.MustFromDecimal("1000000000000000000000000000000000")
	MinFrac     = uint256.MustFromDecimal("10000000000000000")
	MaxFrac     = uint256.MustFromDecimal("100000000000000000000")

	NumTokensU256 = uint256.NewInt(NumTokens)
)

var (
	ErrInvalidReserve               = errors.New("invalid reserve")
	ErrInvalidStoredRates           = errors.New("invalid stored rates")
	ErrInvalidNumToken              = errors.New("invalid number of token")
	ErrInvalidAValue                = errors.New("invalid A value")
	ErrZero                         = errors.New("zero")
	ErrBalancesMustMatchMultipliers = errors.New("balances must match multipliers")
	ErrDDoesNotConverge             = errors.New("d does not converge")
	ErrYDoesNotConverge             = errors.New("y does not converge")
	ErrTokenFromEqualsTokenTo       = errors.New("can't compare token to itself")
	ErrTokenIndexesOutOfRange       = errors.New("token index out of range")
	ErrAmountOutNotConverge         = errors.New("approximation did not converge")
	ErrUnsafeY                      = errors.New("unsafe value for y")
)

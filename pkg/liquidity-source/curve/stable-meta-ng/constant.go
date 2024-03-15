package stablemetang

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "curve-stable-meta-ng"

	poolMethodA            = "A"
	poolMethodAPrecise     = "A_precise"
	poolMethodOffpegFeeMul = "offpeg_fee_multiplier"
	poolMethodInitialA     = "initial_A"
	poolMethodInitialATime = "initial_A_time"
	poolMethodFutureA      = "future_A"
	poolMethodFutureATime  = "future_A_time"
	poolMethodFee          = "fee"
	poolMethodAdminFee     = "admin_fee"
	poolMethodGetBalances  = "get_balances"
	poolMethodStoredRates  = "stored_rates"

	MaxLoopLimit = 256

	N_COINS                 = 2
	MAX_METAPOOL_COIN_INDEX = N_COINS - 1
)

var (
	DefaultGasUnderlying int64 = 260000

	Precision      = uint256.MustFromDecimal("1000000000000000000")
	FeeDenominator = uint256.MustFromDecimal("10000000000")
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

	ErrTokenToUnderLyingNotSupported = errors.New("not support exchange from base pool token to its underlying")
	ErrAllBasePoolTokens             = errors.New("base pool swap should be done at base pool")
	ErrAllMetaPoolTokens             = errors.New("meta pool swap should be done using GetDy")
)

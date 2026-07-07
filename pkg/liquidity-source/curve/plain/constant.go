package plain

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "curve-stable-plain"

	poolMethodA            = "A"
	poolMethodAPrecise     = "A_precise"
	poolMethodInitialA     = "initial_A"
	poolMethodInitialATime = "initial_A_time"
	poolMethodFutureA      = "future_A"
	poolMethodFutureATime  = "future_A_time"
	poolMethodFee          = "fee"
	poolMethodAdminFee     = "admin_fee"
	poolMethodBalances     = "balances"
	poolMethodStoredRates  = "stored_rates"
	poolMethodOracle       = "oracle"
	poolMethodLatestAnswer = "latestAnswer"

	mainRegistryMethodGetRates = "get_rates"

	MaxLoopLimit = 256
)

var (
	DefaultGas     = Gas{Exchange: 128000}
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
	ErrTokenNotFound                = errors.New("token not found")
	ErrWithdrawMoreThanAvailable    = errors.New("cannot withdraw more than available")
	ErrD1LowerThanD0                = errors.New("d1 <= d0")
	ErrDenominatorZero              = errors.New("denominator should not be 0")
	ErrReserveTooSmall              = errors.New("reserve too small")
	ErrInvalidFee                   = errors.New("invalid fee")
	ErrNewReserveInvalid            = errors.New("invalid new reserve")
)

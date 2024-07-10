package stableng

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "curve-stable-ng"

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
	ErrExecutionReverted            = errors.New("execution reverted")
)

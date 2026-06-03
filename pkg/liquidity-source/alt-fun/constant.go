package altfun

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "alt-fun"

	buyGas  = 350000
	sellGas = 300000
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrPoolGraduated         = errors.New("pool is graduated or graduating")
	ErrZeroAmount            = errors.New("zero amount")
	ErrZeroExchangeRate      = errors.New("zero exchange rate")
	ErrZeroK                 = errors.New("zero K invariant")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInsufficientBalance   = errors.New("insufficient base asset balance")
	ErrOverflow              = errors.New("overflow")
	ErrBelowMinAmount        = errors.New("below min amount")
	ErrMintPaused            = errors.New("mint is paused")
	ErrBasePoolNotFound      = errors.New("base pool (bounce-tech LT) not found")

	scaleUp   = big256.TenPow(12) // USDC 6-dec → 18-dec
	precision = big256.BONE       // ScaledNumber precision
	bpsDenomU = big256.UBasisPoint
	minUSDCU  = big256.TenPow(7)
	// memeTokenTotalSupply = Token.TOTAL_SUPPLY = 1_000_000_000 ether (constant across all alt.fun tokens).
	// Used to derive launchTimeVirtualLtReserve from K: virtualLt = K / TOTAL_SUPPLY.
	memeTokenTotalSupply = big256.TenPow(27)
)

package hiddenocean

import (
	"errors"
	"time"
)

const (
	DexType = "hidden-ocean"

	defaultGas int64 = 150000

	// Pool methods
	methodSlot0     = "slot0"
	methodLiquidity = "liquidity"
	methodFee       = "fee"
	methodGetRange  = "getRange"
	methodToken0    = "token0"
	methodToken1    = "token1"

	// ERC20 method
	erc20MethodBalanceOf = "balanceOf"

	// Registry methods
	registryMethodPoolCount = "poolCount"
	registryMethodGetPool   = "getPool"

	defaultNewPoolLimit = 100
)

var (
	// MIN_SQRT_RATIO from TickMath
	minSqrtRatio = "4295128739"
	// MAX_SQRT_RATIO from TickMath
	maxSqrtRatio = "1461446703485210103287273052203988822378723970342"
)

var (
	ErrZeroLiquidity  = errors.New("zero liquidity")
	ErrInvalidToken   = errors.New("invalid token")
	ErrZeroAmountIn   = errors.New("zero amount in")
	ErrNoSwapLimit    = errors.New("no swap at price limit")
	ErrPoolStateStale = errors.New("pool state is stale")
)

const (
	// MaxAge is the maximum acceptable age for pool state before it's considered stale.
	// Oracle price can change every block (~2s), so we keep this tight.
	MaxAge = 30 * time.Second
)

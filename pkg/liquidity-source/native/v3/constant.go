package v3

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType         = "native-v3"
	graphFirstLimit = 1000
	rpcChunkSize    = 100

	poolMethodGetLiquidity = "liquidity"
	poolMethodGetSlot0     = "slot0"
	poolMethodTickSpacing  = "tickSpacing"

	erc20MethodBalanceOf = "balanceOf"

	lpTokenMethodUnderlying = "underlying"

	WrapGasCost   = 80000 // Gas cost for wrapping token
	UnwrapGasCost = 40000 // Gas cost for unwrapping token
)

var (
	defaultGas = Gas{BaseGas: 85000, CrossInitTickGas: 24000}

	ErrPoolLocked      = errors.New("pool is locked")
	ErrOverflow        = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier  = errors.New("invalid feeTier")
	ErrTickNil         = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty    = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
	ErrTokenInInvalid  = errors.New("tokenIn is not correct")
	ErrTokenOutInvalid = errors.New("tokenOut is not correct")
	ErrAmountInZero    = errors.New("amountIn is 0")
	ErrAmountOutZero   = errors.New("amountOut is 0")
)

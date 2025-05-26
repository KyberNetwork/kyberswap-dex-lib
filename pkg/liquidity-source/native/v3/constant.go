package v3

import (
	"errors"
	"math/big"
)

const (
	DexType              = "native-v3"
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	defaultTokenWeight   = 50
	rpcChunkSize         = 100
)

const (
	poolMethodGetLiquidity = "liquidity"
	poolMethodGetSlot0     = "slot0"
	poolMethodTickSpacing  = "tickSpacing"

	erc20MethodBalanceOf = "balanceOf"

	lpTokenMethodUnderlying = "underlying"
)

const (
	WrapGasCost   = 150000 // Gas cost for wrapping token
	UnwrapGasCost = 100000 // Gas cost for unwrapping token
)

var (
	Zero = big.NewInt(0)

	defaultGas = Gas{BaseGas: 85000, CrossInitTickGas: 24000}

	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
	ErrTickNil        = errors.New("tick is nil")
	ErrV3TicksEmpty   = errors.New("v3Ticks empty")
)

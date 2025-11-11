package uniswapv3

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexTypeUniswapV3 = "uniswapv3"

	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	rpcChunkSize         = 100
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodTickSpacing  = "tickSpacing"
	methodTicks        = "ticks"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}

	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
	ErrTickNil        = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty   = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)

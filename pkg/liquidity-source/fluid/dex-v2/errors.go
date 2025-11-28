package dexv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"
)

var (
	ErrOverflow                 = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier           = errors.New("invalid feeTier")
	ErrFluidLiquidityCalcsError = errors.New("fluidLiquidityCalcsError")
	ErrTickNil                  = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty             = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)

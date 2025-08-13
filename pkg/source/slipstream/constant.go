package slipstream

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType              = "slipstream"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	zeroString           = "0"
	emptyString          = ""
)

const (
	methodGetLiquidity   = "liquidity"
	methodGetSlot0       = "slot0"
	methodTickSpacing    = "tickSpacing"
	methodFee            = "fee"
	erc20MethodBalanceOf = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 85000, CrossInitTickGas: 24000}
)

var (
	ErrOverflow           = errors.New("bigInt overflow int/uint256")
	ErrInvalidTickSpacing = errors.New("invalid tickSpacing")
	ErrTickNil            = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty       = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)

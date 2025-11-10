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
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodTickSpacing  = "tickSpacing"
	methodFee          = "fee"
	methodTicks        = "ticks"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}
)

var (
	ErrOverflow           = errors.New("bigInt overflow int/uint256")
	ErrInvalidTickSpacing = errors.New("invalid tickSpacing")
	ErrTickNil            = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty       = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)

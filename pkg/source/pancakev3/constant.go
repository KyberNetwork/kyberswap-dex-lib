package pancakev3

import (
	"math/big"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexTypePancakeV3     = "pancake-v3"
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
	erc20MethodBalanceOf = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}

	TickSpacings = lo.Assign(constants.TickSpacings, map[constants.FeeAmount]int{
		constants.Fee2500: 50,
	})

	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
	ErrTickNil        = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty   = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)

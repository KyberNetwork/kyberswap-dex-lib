package pancakev3

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/samber/lo"
)

const (
	DexTypePancakeV3      = "pancake-v3"
	graphSkipLimit        = 5000
	graphFirstLimit       = 1000
	defaultTokenDecimals  = 18
	defaultTokenWeight    = 50
	zeroString            = "0"
	emptyString           = ""
	graphQLRequestTimeout = 20 * time.Second
)

const (
	methodGetLiquidity   = "liquidity"
	methodGetSlot0       = "slot0"
	methodTickSpacing    = "tickSpacing"
	erc20MethodBalanceOf = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 85000, CrossInitTickGas: 24000}

	TickSpacings = lo.Assign(constants.TickSpacings, map[constants.FeeAmount]int{
		constants.Fee2500: 50,
	})
)

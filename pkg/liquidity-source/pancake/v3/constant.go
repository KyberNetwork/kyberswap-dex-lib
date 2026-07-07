package pancakev3

import (
	"math/big"

	"github.com/samber/lo"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

const (
	DexTypePancakeV3     = "pancake-v3"
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
	methodTicks        = "ticks"
)

var (
	zeroBI = big.NewInt(0)

	PancakeTickSpacings = lo.Assign(uniswapv3.TickSpacings, map[uniswapv3.FeeAmount]int{
		uniswapv3.Fee2500: 50,
	})
)

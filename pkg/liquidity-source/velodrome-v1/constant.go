package velodromev1

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "velodrome"

	pairFactoryMethodIsPaused = "isPaused"
	pairMethodMetadata        = "metadata"
	pairMethodStable          = "stable"
	pairMethodToken0          = "token0"
	pairMethodToken1          = "token1"
	pairMethodGetReserves     = "getReserves"

	genericMethodAllPairs       = "allPairs"
	genericMethodAllPairsLength = "allPairsLength"
	genericMethodFee            = "fee"
	genericTemplatePool         = "pool"
	genericTemplateFactory      = "factory"
	genericTemplateIsStable     = "isStable"

	defaultMethodAllPairs       = 0x1e3dd18b // allPairs(uint256)
	defaultMethodAllPairsLength = 0x574f2ba3 // allPairsLength()

	defaultGas = 227000
)

var (
	routerAddressByExchange = map[string]string{ // used both as router and approval address
		valueobject.ExchangeHyperpieV2: "0xdfBAf8C8d60FBdDc906f95810ffC62e511CB2150",
		valueobject.ExchangeAeonV2:     "0x4d188106175De919a971B0cB6F8A0e3E885a3410",
	}
	extraGasByExchange = map[string]int64{
		valueobject.ExchangeHyperpieV2: 297667 - defaultGas,
	}

	ErrPoolIsPaused             = errors.New("pool is paused")
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidAmountIn          = errors.New("invalid amountIn")
	ErrInvalidAmountOut         = errors.New("invalid amountOut")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
)

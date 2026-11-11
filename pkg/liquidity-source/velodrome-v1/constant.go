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

	// exchangeVelodromeV1 is the DexID of Velodrome V1 on Optimism (the "velodrome" source of the router config)
	exchangeVelodromeV1 = "velodrome"

	defaultGas = 227000

	// maxAmountInRoundingUnits bounds the units added to the amount in of a stable pair (see getAmountIn)
	maxAmountInRoundingUnits = 8
)

var (
	routerAddressByExchange = map[string]string{ // used both as router and approval address
		valueobject.ExchangeHyperpieV2: "0xdfBAf8C8d60FBdDc906f95810ffC62e511CB2150",
		valueobject.ExchangeAeonV2:     "0x4d188106175De919a971B0cB6F8A0e3E885a3410",
	}
	// solidlyCurveExchanges are the exchanges whose pair computes the stable curve like Solidly's (see _fSolidly),
	// checked against pair.getAmountOut on-chain. The others keep the form of _k (aero-v2 is verified to be one).
	solidlyCurveExchanges = map[string]struct{}{
		exchangeVelodromeV1:          {},
		valueobject.ExchangeDystopia: {},
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
	ErrInsufficientBalance      = errors.New("INSUFFICIENT_BALANCE") // safeTransfer of the amount out fails
	ErrK                        = errors.New("K")
)

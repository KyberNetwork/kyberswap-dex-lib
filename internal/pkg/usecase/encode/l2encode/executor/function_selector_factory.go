package executor

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	functionSelectorRegistry           = map[valueobject.Exchange]FunctionSelector{}
	ErrFunctionSelectorIsNotRegistered = errors.New("function selector is not registered")
)

func RegisterFunctionSelector(exchange valueobject.Exchange, functionSelector FunctionSelector) {
	functionSelectorRegistry[exchange] = functionSelector
}

func GetFunctionSelector(exchange valueobject.Exchange) (FunctionSelector, error) {
	functionSelector, ok := functionSelectorRegistry[exchange]
	if !ok {
		return FunctionSelector{}, errors.Wrapf(
			ErrFunctionSelectorIsNotRegistered,
			"exchange: [%s]",
			exchange,
		)
	}

	return functionSelector, nil
}

func init() {
	// executeUniswap
	RegisterFunctionSelector(valueobject.ExchangePancake, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSushiSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSwapr, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeUniSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeZipSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSpartaDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeArbiDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeBaseSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeAlienBase, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSwapBased, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSynthSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeRocketSwapV2, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeDackieV2, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeMoonBase, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeBalDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeMMF, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeArbswapAMM, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeKokonutCpmm, FunctionSelectorUniswap)

	// executeKyberClassic
	RegisterFunctionSelector(valueobject.ExchangeKyberSwap, FunctionSelectorKSClassic)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapStatic, FunctionSelectorKSClassic)

	// executeCamelotSwap
	RegisterFunctionSelector(valueobject.ExchangeCamelot, FunctionSelectorCamelotSwap)

	// executeFraxSwap
	RegisterFunctionSelector(valueobject.ExchangeFraxSwap, FunctionSelectorFraxSwap)

	// executeStableSwap
	RegisterFunctionSelector(valueobject.ExchangeSaddle, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeSynapse, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeAlienBaseStableSwap, FunctionSelectorStableSwap)

	// executeCurveSwap
	RegisterFunctionSelector(valueobject.ExchangeCurve, FunctionSelectorCurveSwap)

	// executeUniV3KSElastic
	RegisterFunctionSelector(valueobject.ExchangeChronosV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeKyberswapElastic, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangePancakeV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeRamsesV2, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeSushiSwapV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeUniSwapV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeArbiDexV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeMMFV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeHorizonDex, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeDackieV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeBaseSwapV3, FunctionSelectorUniV3KSElastic)

	// executeBalV2Swap
	RegisterFunctionSelector(valueobject.ExchangeBalancer, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBalancerComposableStable, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBeethovenX, FunctionSelectorBalancerV2)

	// executeDODOSwap
	RegisterFunctionSelector(valueobject.ExchangeDodo, FunctionSelectorDODO)

	// executeGMXSwap
	RegisterFunctionSelector(valueobject.ExchangeGMX, FunctionSelectorGMX)
	RegisterFunctionSelector(valueobject.ExchangeBMX, FunctionSelectorGMX)
	RegisterFunctionSelector(valueobject.ExchangeBMXGLP, FunctionSelectorGmxGlp)
	RegisterFunctionSelector(valueobject.ExchangeSynthSwapPerp, FunctionSelectorGMX)
	RegisterFunctionSelector(valueobject.ExchangeSwapBasedPerp, FunctionSelectorGMX)

	// executeVelodromeSwap
	RegisterFunctionSelector(valueobject.ExchangeChronos, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeRamses, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelodrome, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelodromeV2, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeBvm, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeBaso, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeAerodrome, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeScale, FunctionSelectorVelodrome)

	// executeKyberLimitOrder
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapLimitOrder, FunctionSelectorLimitOrder)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapLimitOrderDS, FunctionSelectorLimitOrderDS)

	// executeMaverick
	RegisterFunctionSelector(valueobject.ExchangeMaverickV1, FunctionSelectorMaverickV1)

	// executeAlgebraV1
	RegisterFunctionSelector(valueobject.ExchangeCamelotV3, FunctionSelectorAlgebraV1)
	RegisterFunctionSelector(valueobject.ExchangeZyberSwapV3, FunctionSelectorAlgebraV1)
	RegisterFunctionSelector(valueobject.ExchangeSynthSwapV3, FunctionSelectorAlgebraV1)
	RegisterFunctionSelector(valueobject.ExchangeSwapBasedV3, FunctionSelectorAlgebraV1)

	// executeIziSwap
	RegisterFunctionSelector(valueobject.ExchangeIZiSwap, FunctionSelectorIZiSwap)

	// executeWombat
	RegisterFunctionSelector(valueobject.ExchangeWombat, FunctionSelectorWombat)

	// executeWooFiV2
	RegisterFunctionSelector(valueobject.ExchangeWooFiV2, FunctionSelectorWooFiV2)
}

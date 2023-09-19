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

	// executeCurveSwap
	RegisterFunctionSelector(valueobject.ExchangeCurve, FunctionSelectorCurveSwap)

	// executeUniV3KSElastic
	RegisterFunctionSelector(valueobject.ExchangeChronosV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeKyberswapElastic, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangePancakeV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeRamsesV2, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeSushiSwapV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeUniSwapV3, FunctionSelectorUniV3KSElastic)

	// executeBalV2Swap
	RegisterFunctionSelector(valueobject.ExchangeBalancer, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBalancerComposableStable, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBeethovenX, FunctionSelectorBalancerV2)

	// executeDODOSwap
	RegisterFunctionSelector(valueobject.ExchangeDodo, FunctionSelectorDODO)

	// executeGMXSwap
	RegisterFunctionSelector(valueobject.ExchangeGMX, FunctionSelectorGMX)

	// executeVelodromeSwap
	RegisterFunctionSelector(valueobject.ExchangeChronos, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeRamses, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelodrome, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelodromeV2, FunctionSelectorVelodrome)

	// executeKyberLimitOrder
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapLimitOrder, FunctionSelectorLimitOrder)
}

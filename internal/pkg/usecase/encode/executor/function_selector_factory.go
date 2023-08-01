package executor

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	functionSelectorRegistry = map[valueobject.Exchange]FunctionSelector{}

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
	// executeUniSwap
	RegisterFunctionSelector(valueobject.ExchangeSushiSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeTrisolaris, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeWannaSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeNearPad, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePangolin, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeTraderJoe, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeLydia, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeYetiSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeApeSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeJetSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeMDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePancake, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeWault, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePancakeLegacy, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeBiSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePantherSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeVVS, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeCronaSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeCrodex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeMMF, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeEmpireDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePhotonSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeUniSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeShibaSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeDefiSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSpookySwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSpiritSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePaintSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeMorpheus, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeValleySwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeYuzuSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeGemKeeper, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeLizard, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeValleySwapV2, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeZipSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeQuickSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePolycat, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeDFYN, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangePolyDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeGravity, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeCometh, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeDinoSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeKrptoDex, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSafeSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeSwapr, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeWagyuSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeAstroSwap, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeVerse, FunctionSelectorUniswap)
	RegisterFunctionSelector(valueobject.ExchangeEchoDex, FunctionSelectorUniswap)

	// executeCamelotSwap
	RegisterFunctionSelector(valueobject.ExchangeCamelot, FunctionSelectorCamelotSwap)

	// executeFraxSwap
	RegisterFunctionSelector(valueobject.ExchangeFraxSwap, FunctionSelectorFraxSwap)

	// executeStableSwap
	RegisterFunctionSelector(valueobject.ExchangeOneSwap, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeNerve, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeIronStable, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeSynapse, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeSaddle, FunctionSelectorStableSwap)
	RegisterFunctionSelector(valueobject.ExchangeAxial, FunctionSelectorStableSwap)

	// executeCurveSwap
	RegisterFunctionSelector(valueobject.ExchangeCurve, FunctionSelectorCurveSwap)
	RegisterFunctionSelector(valueobject.ExchangeEllipsis, FunctionSelectorCurveSwap)

	// executeUniV3ProMMSwap
	RegisterFunctionSelector(valueobject.ExchangeUniSwapV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeKyberswapElastic, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangePancakeV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeChronosV3, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeRetroCL, FunctionSelectorUniV3KSElastic)
	RegisterFunctionSelector(valueobject.ExchangeHorizonDex, FunctionSelectorUniV3KSElastic)

	// executeBalV2Swap
	RegisterFunctionSelector(valueobject.ExchangeBalancer, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBalancerComposableStable, FunctionSelectorBalancerV2)
	RegisterFunctionSelector(valueobject.ExchangeBeethovenX, FunctionSelectorBalancerV2)

	// executeDODOSwap
	RegisterFunctionSelector(valueobject.ExchangeDodo, FunctionSelectorDODO)

	// executeGMXSwap
	RegisterFunctionSelector(valueobject.ExchangeGMX, FunctionSelectorGMX)
	RegisterFunctionSelector(valueobject.ExchangeMadMex, FunctionSelectorGMX)
	RegisterFunctionSelector(valueobject.ExchangeMetavault, FunctionSelectorGMX)

	// executeSynthetixSwap
	RegisterFunctionSelector(valueobject.ExchangeSynthetix, FunctionSelectorSynthetix)

	// executePSMSwap
	RegisterFunctionSelector(valueobject.ExchangeMakerPSM, FunctionSelectorPSM)

	// executeWrappedstETHSwap
	RegisterFunctionSelector(valueobject.ExchangeMakerLido, FunctionSelectorWSTETH)
	RegisterFunctionSelector(valueobject.ExchangeMakerLidoStETH, FunctionSelectorSTETH)

	// executeKyberDMMSwap
	RegisterFunctionSelector(valueobject.ExchangeDMM, FunctionSelectorKSClassic)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwap, FunctionSelectorKSClassic)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapStatic, FunctionSelectorKSClassic)

	// executeVelodromeSwap
	RegisterFunctionSelector(valueobject.ExchangeVelodrome, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeDystopia, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeChronos, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeRamses, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelocore, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeRetro, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeMuteSwitch, FunctionSelectorMuteSwitch)
	RegisterFunctionSelector(valueobject.ExchangeThena, FunctionSelectorVelodrome)

	// executePlatypusSwap
	RegisterFunctionSelector(valueobject.ExchangePlatypus, FunctionSelectorPlatypus)

	// executeSyncSwap
	RegisterFunctionSelector(valueobject.ExchangeSyncSwap, FunctionSelectorSyncSwap)

	// executeMaverickV1
	RegisterFunctionSelector(valueobject.ExchangeMaverickV1, FunctionSelectorMaverickV1)

	// executeKyberLimitOrder
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapLimitOrder, FunctionSelectorLimitOrder)

}

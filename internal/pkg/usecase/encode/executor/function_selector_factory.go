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
	RegisterFunctionSelector(valueobject.ExchangeSushiSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeTrisolaris, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeWannaSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeNearPad, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePangolin, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeTraderJoe, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeLydia, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeYetiSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeApeSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeJetSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeMDex, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePancake, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeWault, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePancakeLegacy, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeBiSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePantherSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeVVS, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeCronaSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeCrodex, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeMMF, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeEmpireDex, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePhotonSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeUniSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeShibaSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeDefiSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeSpookySwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeSpiritSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePaintSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeMorpheus, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeValleySwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeYuzuSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeGemKeeper, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeLizard, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeValleySwapV2, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeZipSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeQuickSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePolycat, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeDFYN, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangePolyDex, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeGravity, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeCometh, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeDinoSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeKrptoDex, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeSafeSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeSwapr, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeWagyuSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeAstroSwap, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeVerse, FunctionSelectorUniSwap)
	RegisterFunctionSelector(valueobject.ExchangeEchoDex, FunctionSelectorUniSwap)

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
	RegisterFunctionSelector(valueobject.ExchangeUniSwapV3, FunctionSelectorUniSwapV3ProMM)
	RegisterFunctionSelector(valueobject.ExchangeKyberswapElastic, FunctionSelectorUniSwapV3ProMM)

	// executeBalV2Swap
	RegisterFunctionSelector(valueobject.ExchangeBalancer, FunctionSelectorBalancerV2)
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

	// executeKyberDMMSwap
	RegisterFunctionSelector(valueobject.ExchangeDMM, FunctionSelectorKyberDMM)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwap, FunctionSelectorKyberDMM)
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapStatic, FunctionSelectorKyberDMM)

	// executeVelodromeSwap
	RegisterFunctionSelector(valueobject.ExchangeVelodrome, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeDystopia, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeChronos, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeRamses, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeVelocore, FunctionSelectorVelodrome)
	RegisterFunctionSelector(valueobject.ExchangeMuteSwitch, FunctionSelectorMuteSwitch)

	// executePlatypusSwap
	RegisterFunctionSelector(valueobject.ExchangePlatypus, FunctionSelectorPlatypus)

	// executeSyncSwap
	RegisterFunctionSelector(valueobject.ExchangeSyncSwap, FunctionSelectorSyncSwap)

	// executeKyberLimitOrder
	RegisterFunctionSelector(valueobject.ExchangeKyberSwapLimitOrder, FunctionSelectorLimitOrder)

}

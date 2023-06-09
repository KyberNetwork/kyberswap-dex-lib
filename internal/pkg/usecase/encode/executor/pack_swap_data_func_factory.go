package executor

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	packSwapDataFuncRegistry = map[valueobject.Exchange]PackSwapDataFunc{}

	ErrPackSwapDataFuncIsNotRegistered = errors.New("pack swap data function is not registered")
)

// PackSwapDataFunc is a function to pack swap data
type PackSwapDataFunc func(chainID valueobject.ChainID, swap types.EncodingSwap) ([]byte, error)

func RegisterPackSwapDataFunc(exchange valueobject.Exchange, fn PackSwapDataFunc) {
	packSwapDataFuncRegistry[exchange] = fn
}

func GetPackSwapDataFunc(exchange valueobject.Exchange) (PackSwapDataFunc, error) {
	fn, ok := packSwapDataFuncRegistry[exchange]
	if !ok {
		return nil, errors.Wrapf(ErrPackSwapDataFuncIsNotRegistered, "exchange: [%s]", exchange)
	}

	return fn, nil
}

func init() {
	// UniSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeSushiSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeTrisolaris, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeWannaSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeNearPad, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePangolin, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeTraderJoe, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeLydia, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeYetiSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeApeSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeJetSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePancake, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeWault, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePancakeLegacy, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBiSwap, swapdata.PackBiSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePantherSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVVS, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeCronaSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeCrodex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMMF, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeEmpireDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePhotonSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeUniSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeShibaSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDefiSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSpookySwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSpiritSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePaintSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMorpheus, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeValleySwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeYuzuSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeGemKeeper, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeLizard, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeValleySwapV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeZipSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeQuickSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePolycat, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDFYN, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePolyDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeGravity, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeCometh, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDinoSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKrptoDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSafeSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapr, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeWagyuSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAstroSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDMM, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapStatic, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelodrome, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDystopia, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeChronos, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRamses, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelocore, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVerse, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMuteSwitch, swapdata.PackUniSwap)

	RegisterPackSwapDataFunc(valueobject.ExchangeCamelot, swapdata.PackCamelot)

	RegisterPackSwapDataFunc(valueobject.ExchangeFraxSwap, swapdata.PackFraxSwap)

	// StableSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeOneSwap, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeNerve, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeIronStable, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynapse, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSaddle, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAxial, swapdata.PackStableSwap)

	// CurveSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeCurve, swapdata.PackCurveSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeEllipsis, swapdata.PackCurveSwap)

	// UniSwapV3ProMM
	RegisterPackSwapDataFunc(valueobject.ExchangeUniSwapV3, swapdata.PackUniSwapV3ProMM)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberswapElastic, swapdata.PackUniSwapV3ProMM)

	// BalancerV2
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancer, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBeethovenX, swapdata.PackBalancerV2)

	// DODO
	RegisterPackSwapDataFunc(valueobject.ExchangeDodo, swapdata.PackDODO)

	// GMX
	RegisterPackSwapDataFunc(valueobject.ExchangeGMX, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeMadMex, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeMetavault, swapdata.PackGMX)

	// Synthetix
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthetix, swapdata.PackSynthetix)

	// PSM
	RegisterPackSwapDataFunc(valueobject.ExchangeMakerPSM, swapdata.PackPSM)

	// WSTETH
	RegisterPackSwapDataFunc(valueobject.ExchangeMakerLido, swapdata.PackWSTETH)

	// Platypus
	RegisterPackSwapDataFunc(valueobject.ExchangePlatypus, swapdata.PackPlatypus)

	// KyberLimitOrder
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapLimitOrder, swapdata.PackKyberLimitOrder)
}

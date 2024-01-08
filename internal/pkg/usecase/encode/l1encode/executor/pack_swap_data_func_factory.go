package executor

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor/swapdata"
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
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwap, swapdata.PackUniSwap)
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
	RegisterPackSwapDataFunc(valueobject.ExchangeFvm, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBvm, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeThena, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDystopia, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeChronos, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRamses, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelocore, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVerse, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMuteSwitch, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeEchoDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRetro, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePearl, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePearlV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBaseSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAlienBase, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBased, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBaso, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRocketSwapV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelodromeV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAerodrome, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSpartaDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeArbiDex, swapdata.PackBiSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSpacefi, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeLyve, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeCrowdswapV2, swapdata.PackBiSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVesync, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDackieV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMoonBase, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeScale, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBalDex, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeUSDFi, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeZkSwapFinance, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeScrollSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSkydrome, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangePunkSwap, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMetavaultV2, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeNomiswap, swapdata.PackBiSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeArbswapAMM, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKokonutCpmm, swapdata.PackUniSwap)

	RegisterPackSwapDataFunc(valueobject.ExchangeZebra, swapdata.PackUniSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeZKSwap, swapdata.PackUniSwap)

	RegisterPackSwapDataFunc(valueobject.ExchangeCamelot, swapdata.PackCamelot)
	RegisterPackSwapDataFunc(valueobject.ExchangeEzkalibur, swapdata.PackCamelot)

	RegisterPackSwapDataFunc(valueobject.ExchangeFraxSwap, swapdata.PackFraxSwap)

	// StableSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeOneSwap, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeNerve, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeIronStable, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynapse, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSaddle, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAxial, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAlienBaseStableSwap, swapdata.PackStableSwap)

	// CurveSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeCurve, swapdata.PackCurveSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeEllipsis, swapdata.PackCurveSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKokonutCrypto, swapdata.PackKokonutCrypto)
	RegisterPackSwapDataFunc(valueobject.ExchangePancakeStable, swapdata.PackPancakeStableSwap)

	// UniSwapV3ProMM
	RegisterPackSwapDataFunc(valueobject.ExchangeUniSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberswapElastic, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangePancakeV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeChronosV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeRetroV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeHorizonDex, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeDoveSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeSushiSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeRamsesV2, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeEchoDexV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeDackieV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeHoriza, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeBaseSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeArbiDexV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeWagmi, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeMMFV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeMetavaultV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeSolidlyV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeZero, swapdata.PackUniswapV3KSElastic)

	// BalancerV2
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancerV2Weighted, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancerV2Stable, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancerV2ComposableStable, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBeethovenXWeighted, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBeethovenXStable, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBeethovenXComposableStable, swapdata.PackBalancerV2)

	// DODO
	RegisterPackSwapDataFunc(valueobject.ExchangeDodo, swapdata.PackDODO)

	// GMX
	RegisterPackSwapDataFunc(valueobject.ExchangeGMX, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeMadMex, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeMetavault, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeBMX, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwapPerp, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBasedPerp, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeFxdx, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeQuickPerps, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeMummyFinance, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeOpx, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeFulcrom, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeVodoo, swapdata.PackGMX)

	// Synthetix
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthetix, swapdata.PackSynthetix)

	// PSM
	RegisterPackSwapDataFunc(valueobject.ExchangeMakerPSM, swapdata.PackPSM)

	// WSTETH
	RegisterPackSwapDataFunc(valueobject.ExchangeMakerLido, swapdata.PackWSTETH)
	RegisterPackSwapDataFunc(valueobject.ExchangeMakerLidoStETH, swapdata.PackStETH)

	// Platypus
	RegisterPackSwapDataFunc(valueobject.ExchangePlatypus, swapdata.PackPlatypus)

	// KyberLimitOrder
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapLimitOrder, swapdata.PackKyberLimitOrder)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapLimitOrderDS, swapdata.PackKyberLimitOrderDS)

	// SyncSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeSyncSwap, swapdata.PackSyncSwap)

	// MaverickV1
	RegisterPackSwapDataFunc(valueobject.ExchangeMaverickV1, swapdata.PackMaverickV1)

	// AlgebraV1
	RegisterPackSwapDataFunc(valueobject.ExchangeQuickSwapV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwapV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBasedV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeLynex, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeCamelotV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeZyberSwapV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeThenaFusion, swapdata.PackAlgebraV1)

	// TraderJoeV20 and TraderJoeV21
	RegisterPackSwapDataFunc(valueobject.ExchangeTraderJoeV20, swapdata.PackTraderJoeV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeTraderJoeV21, swapdata.PackTraderJoeV2)

	// KyberPMM
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberPMM, swapdata.PackKyberRFQ)

	// IZiSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeIZiSwap, swapdata.PackIZiSwap)

	// Wombat
	RegisterPackSwapDataFunc(valueobject.ExchangeWombat, swapdata.PackWombat)
	RegisterPackSwapDataFunc(valueobject.ExchangeWooFiV2, swapdata.PackWombat)
	RegisterPackSwapDataFunc(valueobject.ExchangeMantisSwap, swapdata.PackWombat)

	// Vooi
	RegisterPackSwapDataFunc(valueobject.ExchangeVooi, swapdata.PackVooi)

	RegisterPackSwapDataFunc(valueobject.ExchangePolMatic, swapdata.PackMaticMigrate)

	// Smardex
	RegisterPackSwapDataFunc(valueobject.ExchangeSmardex, swapdata.PackSmardex)

	// BalancerV1
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancerV1, swapdata.PackBalancerV1)

	// VelocoreV2
	RegisterPackSwapDataFunc(valueobject.ExchangeVelocoreV2CPMM, swapdata.PackVelocoreV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelocoreV2WombatStable, swapdata.PackVelocoreV2)
}

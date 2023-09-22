package valueobject

type Exchange string

var (
	ExchangeSushiSwap     Exchange = "sushiswap"
	ExchangeTrisolaris    Exchange = "trisolaris"
	ExchangeWannaSwap     Exchange = "wannaswap"
	ExchangeNearPad       Exchange = "nearpad"
	ExchangePangolin      Exchange = "pangolin"
	ExchangeTraderJoe     Exchange = "traderjoe"
	ExchangeLydia         Exchange = "lydia"
	ExchangeYetiSwap      Exchange = "yetiswap"
	ExchangeApeSwap       Exchange = "apeswap"
	ExchangeJetSwap       Exchange = "jetswap"
	ExchangeMDex          Exchange = "mdex"
	ExchangePancake       Exchange = "pancake"
	ExchangePancakeV3     Exchange = "pancake-v3"
	ExchangePancakeStable Exchange = "pancake-stable"
	ExchangeEchoDexV3     Exchange = "echo-dex-v3"
	ExchangeWault         Exchange = "wault"
	ExchangePancakeLegacy Exchange = "pancake-legacy"
	ExchangeBiSwap        Exchange = "biswap"
	ExchangePantherSwap   Exchange = "pantherswap"
	ExchangeVVS           Exchange = "vvs"
	ExchangeCronaSwap     Exchange = "cronaswap"
	ExchangeCrodex        Exchange = "crodex"
	ExchangeMMF           Exchange = "mmf"
	ExchangeEmpireDex     Exchange = "empiredex"
	ExchangePhotonSwap    Exchange = "photonswap"
	ExchangeUniSwap       Exchange = "uniswap"
	ExchangeShibaSwap     Exchange = "shibaswap"
	ExchangeDefiSwap      Exchange = "defiswap"
	ExchangeSpookySwap    Exchange = "spookyswap"
	ExchangeSpiritSwap    Exchange = "spiritswap"
	ExchangePaintSwap     Exchange = "paintswap"
	ExchangeMorpheus      Exchange = "morpheus"
	ExchangeValleySwap    Exchange = "valleyswap"
	ExchangeYuzuSwap      Exchange = "yuzuswap"
	ExchangeGemKeeper     Exchange = "gemkeeper"
	ExchangeLizard        Exchange = "lizard"
	ExchangeValleySwapV2  Exchange = "valleyswap-v2"
	ExchangeZipSwap       Exchange = "zipswap"
	ExchangeQuickSwap     Exchange = "quickswap"
	ExchangeQuickSwapV3   Exchange = "quickswap-v3"
	ExchangeSynthSwap     Exchange = "synthswap"
	ExchangeSynthSwapV3   Exchange = "synthswap-v3"
	ExchangeSwapBasedV3   Exchange = "swapbased-v3"
	ExchangeLynex         Exchange = "lynex"
	ExchangePolycat       Exchange = "polycat"
	ExchangeDFYN          Exchange = "dfyn"
	ExchangePolyDex       Exchange = "polydex"
	ExchangeGravity       Exchange = "gravity"
	ExchangeCometh        Exchange = "cometh"
	ExchangeDinoSwap      Exchange = "dinoswap"
	ExchangeKrptoDex      Exchange = "kryptodex"
	ExchangeSafeSwap      Exchange = "safeswap"
	ExchangeSwapr         Exchange = "swapr"
	ExchangeWagyuSwap     Exchange = "wagyuswap"
	ExchangeAstroSwap     Exchange = "astroswap"
	ExchangeCamelot       Exchange = "camelot"
	ExchangeCamelotV3     Exchange = "camelot-v3"
	ExchangeFraxSwap      Exchange = "fraxswap"
	ExchangeVerse         Exchange = "verse"
	ExchangeEchoDex       Exchange = "echo-dex"
	ExchangeBaseSwap      Exchange = "baseswap"
	ExchangeAlienBase     Exchange = "alien-base"
	ExchangeSwapBased     Exchange = "swapbased"
	ExchangeRocketSwapV2  Exchange = "rocketswap-v2"
	ExchangeSpartaDex     Exchange = "sparta-dex"
	ExchangeArbiDex       Exchange = "arbi-dex"
	ExchangeZyberSwapV3   Exchange = "zyberswap-v3"

	ExchangeOneSwap             Exchange = "oneswap"
	ExchangeNerve               Exchange = "nerve"
	ExchangeIronStable          Exchange = "iron-stable"
	ExchangeSynapse             Exchange = "synapse"
	ExchangeSaddle              Exchange = "saddle"
	ExchangeAxial               Exchange = "axial"
	ExchangeAlienBaseStableSwap Exchange = "alien-base-stableswap"

	ExchangeCurve    Exchange = "curve"
	ExchangeEllipsis Exchange = "ellipsis"

	ExchangeUniSwapV3        Exchange = "uniswapv3"
	ExchangeKyberswapElastic Exchange = "kyberswap-elastic"
	ExchangeChronosV3        Exchange = "chronos-v3"
	ExchangeRetroV3          Exchange = "retro-v3"
	ExchangeHorizonDex       Exchange = "horizon-dex"
	ExchangeDoveSwapV3       Exchange = "doveswap-v3"
	ExchangeSushiSwapV3      Exchange = "sushiswap-v3"
	ExchangeRamsesV2         Exchange = "ramses-v2"
	ExchangeDackieV3         Exchange = "dackie-v3"

	ExchangeBalancer                 Exchange = "balancer"
	ExchangeBalancerComposableStable Exchange = "balancer-composable-stable"
	ExchangeBeethovenX               Exchange = "beethovenx"

	ExchangeDodo Exchange = "dodo"

	ExchangeGMX       Exchange = "gmx"
	ExchangeMadMex    Exchange = "madmex"
	ExchangeMetavault Exchange = "metavault"

	ExchangeSynthetix Exchange = "synthetix"

	ExchangeMakerPSM Exchange = "maker-psm"

	ExchangeMakerLido Exchange = "lido"

	ExchangeMakerLidoStETH Exchange = "lido-steth"

	ExchangeDMM             Exchange = "dmm"
	ExchangeKyberSwap       Exchange = "kyberswap"
	ExchangeKyberSwapStatic Exchange = "kyberswap-static"

	ExchangeVelodrome   Exchange = "velodrome"
	ExchangeVelodromeV2 Exchange = "velodrome-v2"
	ExchangeAerodrome   Exchange = "aerodrome"
	ExchangeFvm         Exchange = "fvm"
	ExchangeBvm         Exchange = "bvm"
	ExchangeDystopia    Exchange = "dystopia"
	ExchangeChronos     Exchange = "chronos"
	ExchangeRamses      Exchange = "ramses"
	ExchangeVelocore    Exchange = "velocore"
	ExchangeMuteSwitch  Exchange = "muteswitch"
	ExchangeRetro       Exchange = "retro"
	ExchangeThena       Exchange = "thena"
	ExchangePearl       Exchange = "pearl"
	ExchangePearlV2     Exchange = "pearl-v2"
	ExchangeBaso        Exchange = "baso"

	ExchangePlatypus Exchange = "platypus"
	ExchangeWombat   Exchange = "wombat"

	ExchangeSyncSwap Exchange = "syncswap"

	ExchangeMaverickV1 Exchange = "maverick-v1"

	ExchangeKyberSwapLimitOrder   Exchange = "kyberswap-limit-order"
	ExchangeKyberSwapLimitOrderDS Exchange = "kyberswap-limit-order-v2"

	ExchangeKyberPMM Exchange = "kyber-pmm"

	ExchangeTraderJoeV20 Exchange = "traderjoe-v20"
	ExchangeTraderJoeV21 Exchange = "traderjoe-v21"

	ExchangeIZiSwap Exchange = "iziswap"
)

var AMMSourceSet = map[Exchange]struct{}{
	ExchangeSushiSwap:                {},
	ExchangeTrisolaris:               {},
	ExchangeWannaSwap:                {},
	ExchangeNearPad:                  {},
	ExchangePangolin:                 {},
	ExchangeTraderJoe:                {},
	ExchangeLydia:                    {},
	ExchangeYetiSwap:                 {},
	ExchangeApeSwap:                  {},
	ExchangeJetSwap:                  {},
	ExchangeMDex:                     {},
	ExchangePancake:                  {},
	ExchangePancakeV3:                {},
	ExchangeEchoDexV3:                {},
	ExchangeWault:                    {},
	ExchangePancakeLegacy:            {},
	ExchangeBiSwap:                   {},
	ExchangePantherSwap:              {},
	ExchangeVVS:                      {},
	ExchangeCronaSwap:                {},
	ExchangeCrodex:                   {},
	ExchangeMMF:                      {},
	ExchangeEmpireDex:                {},
	ExchangePhotonSwap:               {},
	ExchangeUniSwap:                  {},
	ExchangeShibaSwap:                {},
	ExchangeDefiSwap:                 {},
	ExchangeSpookySwap:               {},
	ExchangeSpiritSwap:               {},
	ExchangePaintSwap:                {},
	ExchangeMorpheus:                 {},
	ExchangeValleySwap:               {},
	ExchangeYuzuSwap:                 {},
	ExchangeGemKeeper:                {},
	ExchangeLizard:                   {},
	ExchangeValleySwapV2:             {},
	ExchangeZipSwap:                  {},
	ExchangeQuickSwap:                {},
	ExchangeQuickSwapV3:              {},
	ExchangeSynthSwap:                {},
	ExchangeSynthSwapV3:              {},
	ExchangeSwapBasedV3:              {},
	ExchangeLynex:                    {},
	ExchangePolycat:                  {},
	ExchangeDFYN:                     {},
	ExchangePolyDex:                  {},
	ExchangeGravity:                  {},
	ExchangeCometh:                   {},
	ExchangeDinoSwap:                 {},
	ExchangeKrptoDex:                 {},
	ExchangeSafeSwap:                 {},
	ExchangeSwapr:                    {},
	ExchangeWagyuSwap:                {},
	ExchangeAstroSwap:                {},
	ExchangeCamelot:                  {},
	ExchangeCamelotV3:                {},
	ExchangeFraxSwap:                 {},
	ExchangeVerse:                    {},
	ExchangeEchoDex:                  {},
	ExchangeOneSwap:                  {},
	ExchangeNerve:                    {},
	ExchangeIronStable:               {},
	ExchangeSynapse:                  {},
	ExchangeSaddle:                   {},
	ExchangeAxial:                    {},
	ExchangeCurve:                    {},
	ExchangeEllipsis:                 {},
	ExchangePancakeStable:            {},
	ExchangeUniSwapV3:                {},
	ExchangeKyberswapElastic:         {},
	ExchangeChronosV3:                {},
	ExchangeRetroV3:                  {},
	ExchangeBalancer:                 {},
	ExchangeBalancerComposableStable: {},
	ExchangeBeethovenX:               {},
	ExchangeDodo:                     {},
	ExchangeGMX:                      {},
	ExchangeMadMex:                   {},
	ExchangeMetavault:                {},
	ExchangeSynthetix:                {},
	ExchangeMakerPSM:                 {},
	ExchangeMakerLido:                {},
	ExchangeMakerLidoStETH:           {},
	ExchangeDMM:                      {},
	ExchangeKyberSwap:                {},
	ExchangeKyberSwapStatic:          {},
	ExchangeVelodrome:                {},
	ExchangeVelodromeV2:              {},
	ExchangeAerodrome:                {},
	ExchangeFvm:                      {},
	ExchangeBvm:                      {},
	ExchangeThena:                    {},
	ExchangePearl:                    {},
	ExchangePearlV2:                  {},
	ExchangeDystopia:                 {},
	ExchangeChronos:                  {},
	ExchangeRamses:                   {},
	ExchangeVelocore:                 {},
	ExchangeMuteSwitch:               {},
	ExchangePlatypus:                 {},
	ExchangeWombat:                   {},
	ExchangeSyncSwap:                 {},
	ExchangeKyberSwapLimitOrder:      {},
	ExchangeKyberSwapLimitOrderDS:    {},
	ExchangeMaverickV1:               {},
	ExchangeHorizonDex:               {},
	ExchangeRetro:                    {},
	ExchangeDoveSwapV3:               {},
	ExchangeSushiSwapV3:              {},
	ExchangeRamsesV2:                 {},
	ExchangeBaseSwap:                 {},
	ExchangeAlienBase:                {},
	ExchangeSwapBased:                {},
	ExchangeBaso:                     {},
	ExchangeRocketSwapV2:             {},
	ExchangeDackieV3:                 {},
	ExchangeKyberPMM:                 {},
	ExchangeTraderJoeV20:             {},
	ExchangeTraderJoeV21:             {},
	ExchangeSpartaDex:                {},
	ExchangeArbiDex:                  {},
	ExchangeAlienBaseStableSwap:      {},
	ExchangeIZiSwap:                  {},
	ExchangeZyberSwapV3:              {},
}

func IsAMMSource(exchange Exchange) bool {
	_, contained := AMMSourceSet[exchange]

	return contained
}

var RFQSourceSet = map[Exchange]struct{}{
	ExchangeKyberPMM: {},
}

func IsRFQSource(exchange Exchange) bool {
	_, contained := RFQSourceSet[exchange]

	return contained
}

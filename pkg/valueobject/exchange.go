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
	ExchangeFraxSwap      Exchange = "fraxswap"

	ExchangeOneSwap    Exchange = "oneswap"
	ExchangeNerve      Exchange = "nerve"
	ExchangeIronStable Exchange = "iron-stable"
	ExchangeSynapse    Exchange = "synapse"
	ExchangeSaddle     Exchange = "saddle"
	ExchangeAxial      Exchange = "axial"

	ExchangeCurve         Exchange = "curve"
	ExchangeEllipsis      Exchange = "ellipsis"
	ExchangePancakeStable Exchange = "pancake-stable"

	ExchangeUniSwapV3        Exchange = "uniswapv3"
	ExchangeKyberswapElastic Exchange = "kyberswap-elastic"

	ExchangeBalancer   Exchange = "balancer"
	ExchangeBeethovenX Exchange = "beethovenx"

	ExchangeDodo Exchange = "dodo"

	ExchangeGMX       Exchange = "gmx"
	ExchangeMadMex    Exchange = "madmex"
	ExchangeMetavault Exchange = "metavault"

	ExchangeSynthetix Exchange = "synthetix"

	ExchangeMakerPSM Exchange = "maker-psm"

	ExchangeMakerLido Exchange = "lido"

	ExchangeDMM             Exchange = "dmm"
	ExchangeKyberSwap       Exchange = "kyberswap"
	ExchangeKyberSwapStatic Exchange = "kyberswap-static"

	ExchangeVelodrome Exchange = "velodrome"
	ExchangeDystopia  Exchange = "dystopia"
	ExchangeChronos   Exchange = "chronos"
	ExchangeRamses    Exchange = "ramses"
	ExchangeVelocore  Exchange = "velocore"
	ExchangePearl     Exchange = "pearl"

	ExchangePlatypus Exchange = "platypus"

	ExchangeKyberSwapLimitOrder Exchange = "kyberswap-limit-order"
)

var AMMSourceSet = map[Exchange]struct{}{
	ExchangeSushiSwap:           {},
	ExchangeTrisolaris:          {},
	ExchangeWannaSwap:           {},
	ExchangeNearPad:             {},
	ExchangePangolin:            {},
	ExchangeTraderJoe:           {},
	ExchangeLydia:               {},
	ExchangeYetiSwap:            {},
	ExchangeApeSwap:             {},
	ExchangeJetSwap:             {},
	ExchangeMDex:                {},
	ExchangePancake:             {},
	ExchangeWault:               {},
	ExchangePancakeLegacy:       {},
	ExchangeBiSwap:              {},
	ExchangePantherSwap:         {},
	ExchangeVVS:                 {},
	ExchangeCronaSwap:           {},
	ExchangeCrodex:              {},
	ExchangeMMF:                 {},
	ExchangeEmpireDex:           {},
	ExchangePhotonSwap:          {},
	ExchangeUniSwap:             {},
	ExchangeShibaSwap:           {},
	ExchangeDefiSwap:            {},
	ExchangeSpookySwap:          {},
	ExchangeSpiritSwap:          {},
	ExchangePaintSwap:           {},
	ExchangeMorpheus:            {},
	ExchangeValleySwap:          {},
	ExchangeYuzuSwap:            {},
	ExchangeGemKeeper:           {},
	ExchangeLizard:              {},
	ExchangeValleySwapV2:        {},
	ExchangeZipSwap:             {},
	ExchangeQuickSwap:           {},
	ExchangePolycat:             {},
	ExchangeDFYN:                {},
	ExchangePolyDex:             {},
	ExchangeGravity:             {},
	ExchangeCometh:              {},
	ExchangeDinoSwap:            {},
	ExchangeKrptoDex:            {},
	ExchangeSafeSwap:            {},
	ExchangeSwapr:               {},
	ExchangeWagyuSwap:           {},
	ExchangeAstroSwap:           {},
	ExchangeCamelot:             {},
	ExchangeFraxSwap:            {},
	ExchangeOneSwap:             {},
	ExchangeNerve:               {},
	ExchangeIronStable:          {},
	ExchangeSynapse:             {},
	ExchangeSaddle:              {},
	ExchangeAxial:               {},
	ExchangeCurve:               {},
	ExchangeEllipsis:            {},
	ExchangePancakeStable:       {},
	ExchangeUniSwapV3:           {},
	ExchangeKyberswapElastic:    {},
	ExchangeBalancer:            {},
	ExchangeBeethovenX:          {},
	ExchangeDodo:                {},
	ExchangeGMX:                 {},
	ExchangeMadMex:              {},
	ExchangeMetavault:           {},
	ExchangeSynthetix:           {},
	ExchangeMakerPSM:            {},
	ExchangeMakerLido:           {},
	ExchangeDMM:                 {},
	ExchangeKyberSwap:           {},
	ExchangeKyberSwapStatic:     {},
	ExchangeVelodrome:           {},
	ExchangePearl:               {},
	ExchangeDystopia:            {},
	ExchangeChronos:             {},
	ExchangeRamses:              {},
	ExchangeVelocore:            {},
	ExchangePlatypus:            {},
	ExchangeKyberSwapLimitOrder: {},
}

func IsAMMSource(exchange Exchange) bool {
	_, contained := AMMSourceSet[exchange]

	return contained
}

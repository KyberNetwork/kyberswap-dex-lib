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
	ExchangeUniSwapV2     Exchange = "uniswap-v2"
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

	ExchangePancakeV3     Exchange = "pancake-v3"
	ExchangeEchoDexV3     Exchange = "echo-dex-v3"
	ExchangeCrowdswapV2   Exchange = "crowdswap-v2"
	ExchangeQuickSwapV3   Exchange = "quickswap-v3"
	ExchangeSynthSwap     Exchange = "synthswap"
	ExchangeSynthSwapV3   Exchange = "synthswap-v3"
	ExchangeSwapBasedV3   Exchange = "swapbased-v3"
	ExchangeLynex         Exchange = "lynex"
	ExchangeCamelotV3     Exchange = "camelot-v3"
	ExchangeVerse         Exchange = "verse"
	ExchangeEchoDex       Exchange = "echo-dex"
	ExchangeBaseSwap      Exchange = "baseswap"
	ExchangeAlienBase     Exchange = "alien-base"
	ExchangeSwapBased     Exchange = "swapbased"
	ExchangeRocketSwapV2  Exchange = "rocketswap-v2"
	ExchangeSpartaDex     Exchange = "sparta-dex"
	ExchangeArbiDex       Exchange = "arbi-dex"
	ExchangeZyberSwapV3   Exchange = "zyberswap-v3"
	ExchangeSpacefi       Exchange = "spacefi"
	ExchangeEzkalibur     Exchange = "ezkalibur"
	ExchangeMoonBase      Exchange = "moonbase"
	ExchangeBalDex        Exchange = "baldex"
	ExchangeZkSwapFinance Exchange = "zkswap-finance"
	ExchangeScrollSwap    Exchange = "scrollswap"
	ExchangePunkSwap      Exchange = "punkswap"
	ExchangeMetavaultV2   Exchange = "metavault-v2"
	ExchangeNomiswap      Exchange = "nomiswap"
	ExchangeArbswapAMM    Exchange = "arbswap-amm"
	ExchangeKokonutCpmm   Exchange = "kokonut-cpmm"

	ExchangeKokonutCrypto Exchange = "kokonut-crypto"

	ExchangeChronosV3   Exchange = "chronos-v3"
	ExchangeRetroV3     Exchange = "retro-v3"
	ExchangeHorizonDex  Exchange = "horizon-dex"
	ExchangeDoveSwapV3  Exchange = "doveswap-v3"
	ExchangeSushiSwapV3 Exchange = "sushiswap-v3"
	ExchangeRamsesV2    Exchange = "ramses-v2"
	ExchangeDackieV3    Exchange = "dackie-v3"
	ExchangeHoriza      Exchange = "horiza"
	ExchangeBaseSwapV3  Exchange = "baseswap-v3"
	ExchangeArbiDexV3   Exchange = "arbidex-v3"
	ExchangeWagmi       Exchange = "wagmi"
	ExchangeMetavaultV3 Exchange = "metavault-v3"
	ExchangeSolidlyV3   Exchange = "solidly-v3"
	ExchangeZero        Exchange = "zero"
	ExchangeZebraV2     Exchange = "zebra-v2"

	ExchangeBalancerV2Weighted         Exchange = "balancer-v2-weighted"
	ExchangeBalancerV2Stable           Exchange = "balancer-v2-stable"
	ExchangeBalancerV2ComposableStable Exchange = "balancer-v2-composable-stable"
	ExchangeBeethovenXWeighted         Exchange = "beethovenx-weighted"
	ExchangeBeethovenXStable           Exchange = "beethovenx-stable"
	ExchangeBeethovenXComposableStable Exchange = "beethovenx-composable-stable"
	ExchangeGyroscope2CLP              Exchange = "gyroscope-2clp"
	ExchangeGyroscope3CLP              Exchange = "gyroscope-3clp"
	ExchangeGyroscopeECLP              Exchange = "gyroscope-eclp"

	ExchangeSynthSwapPerp Exchange = "synthswap-perp"
	ExchangeSwapBasedPerp Exchange = "swapbased-perp"
	ExchangeBMX           Exchange = "bmx"
	ExchangeBMXGLP        Exchange = "bmx-glp"
	ExchangeFxdx          Exchange = "fxdx"
	ExchangeQuickPerps    Exchange = "quickperps"
	ExchangeMummyFinance  Exchange = "mummy-finance"
	ExchangeOpx           Exchange = "opx"
	ExchangeFulcrom       Exchange = "fulcrom"
	ExchangeVodoo         Exchange = "vodoo"

	ExchangeMakerLidoStETH Exchange = "lido-steth"

	ExchangeVelodromeV2 Exchange = "velodrome-v2"
	ExchangeAerodrome   Exchange = "aerodrome"
	ExchangeFvm         Exchange = "fvm"
	ExchangeBvm         Exchange = "bvm"
	ExchangeMuteSwitch  Exchange = "muteswitch"
	ExchangeRetro       Exchange = "retro"
	ExchangeThena       Exchange = "thena"
	ExchangeThenaFusion Exchange = "thena-fusion"
	ExchangePearlV2     Exchange = "pearl-v2"
	ExchangeBaso        Exchange = "baso"
	ExchangeLyve        Exchange = "lyve"
	ExchangeScale       Exchange = "scale"
	ExchangeUSDFi       Exchange = "usdfi"
	ExchangeSkydrome    Exchange = "skydrome"

	ExchangeWombat     Exchange = "wombat"
	ExchangeMantisSwap Exchange = "mantisswap"

	ExchangeSyncSwap Exchange = "syncswap"

	ExchangeMaverickV1 Exchange = "maverick-v1"

	ExchangeKyberSwapLimitOrderDS Exchange = "kyberswap-limit-order-v2"

	ExchangeKyberPMM Exchange = "kyber-pmm"
	ExchangeSwaapV2  Exchange = "swaap-v2"

	ExchangeTraderJoeV20 Exchange = "traderjoe-v20"
	ExchangeTraderJoeV21 Exchange = "traderjoe-v21"

	ExchangeIZiSwap Exchange = "iziswap"

	ExchangeWooFiV2  Exchange = "woofi-v2"
	ExchangeVesync   Exchange = "vesync"
	ExchangeDackieV2 Exchange = "dackie-v2"

	ExchangeMMFV3 Exchange = "mmf-v3"

	ExchangeVooi Exchange = "vooi"

	ExchangePolMatic Exchange = "pol-matic"

	ExchangeSmardex Exchange = "smardex"

	ExchangeZebra  Exchange = "zebra"
	ExchangeZKSwap Exchange = "zkswap"

	ExchangeBalancerV1 Exchange = "balancer-v1"

	ExchangeVelocoreV2CPMM         Exchange = "velocore-v2-cpmm"
	ExchangeVelocoreV2WombatStable Exchange = "velocore-v2-wombat-stable"
	ExchangeAlienBaseStableSwap    Exchange = "alien-base-stableswap"
)

var AMMSourceSet = map[Exchange]struct{}{
	ExchangeSushiSwap:                  {},
	ExchangeTrisolaris:                 {},
	ExchangeWannaSwap:                  {},
	ExchangeNearPad:                    {},
	ExchangePangolin:                   {},
	ExchangeTraderJoe:                  {},
	ExchangeLydia:                      {},
	ExchangeYetiSwap:                   {},
	ExchangeApeSwap:                    {},
	ExchangeJetSwap:                    {},
	ExchangeMDex:                       {},
	ExchangePancake:                    {},
	ExchangeWault:                      {},
	ExchangePancakeLegacy:              {},
	ExchangeBiSwap:                     {},
	ExchangePantherSwap:                {},
	ExchangeVVS:                        {},
	ExchangeCronaSwap:                  {},
	ExchangeCrodex:                     {},
	ExchangeMMF:                        {},
	ExchangeEmpireDex:                  {},
	ExchangePhotonSwap:                 {},
	ExchangeUniSwap:                    {},
	ExchangeUniSwapV2:                  {},
	ExchangeShibaSwap:                  {},
	ExchangeDefiSwap:                   {},
	ExchangeSpookySwap:                 {},
	ExchangeSpiritSwap:                 {},
	ExchangePaintSwap:                  {},
	ExchangeMorpheus:                   {},
	ExchangeValleySwap:                 {},
	ExchangeYuzuSwap:                   {},
	ExchangeGemKeeper:                  {},
	ExchangeLizard:                     {},
	ExchangeValleySwapV2:               {},
	ExchangeZipSwap:                    {},
	ExchangeQuickSwap:                  {},
	ExchangePolycat:                    {},
	ExchangeDFYN:                       {},
	ExchangePolyDex:                    {},
	ExchangeGravity:                    {},
	ExchangeCometh:                     {},
	ExchangeDinoSwap:                   {},
	ExchangeKrptoDex:                   {},
	ExchangeSafeSwap:                   {},
	ExchangeSwapr:                      {},
	ExchangeWagyuSwap:                  {},
	ExchangeAstroSwap:                  {},
	ExchangeCamelot:                    {},
	ExchangeFraxSwap:                   {},
	ExchangeOneSwap:                    {},
	ExchangeNerve:                      {},
	ExchangeIronStable:                 {},
	ExchangeSynapse:                    {},
	ExchangeSaddle:                     {},
	ExchangeAxial:                      {},
	ExchangeCurve:                      {},
	ExchangeEllipsis:                   {},
	ExchangePancakeStable:              {},
	ExchangeUniSwapV3:                  {},
	ExchangeKyberswapElastic:           {},
	ExchangeBalancer:                   {},
	ExchangeBeethovenX:                 {},
	ExchangeDodo:                       {},
	ExchangeGMX:                        {},
	ExchangeMadMex:                     {},
	ExchangeMetavault:                  {},
	ExchangeSynthetix:                  {},
	ExchangeMakerPSM:                   {},
	ExchangeMakerLido:                  {},
	ExchangeDMM:                        {},
	ExchangeKyberSwap:                  {},
	ExchangeKyberSwapStatic:            {},
	ExchangeVelodrome:                  {},
	ExchangePearl:                      {},
	ExchangeDystopia:                   {},
	ExchangeChronos:                    {},
	ExchangeRamses:                     {},
	ExchangeVelocore:                   {},
	ExchangePlatypus:                   {},
	ExchangeKyberSwapLimitOrder:        {},
	ExchangePancakeV3:                  {},
	ExchangeEchoDexV3:                  {},
	ExchangeCrowdswapV2:                {},
	ExchangeQuickSwapV3:                {},
	ExchangeSynthSwap:                  {},
	ExchangeSynthSwapV3:                {},
	ExchangeSwapBasedV3:                {},
	ExchangeLynex:                      {},
	ExchangeCamelotV3:                  {},
	ExchangeVerse:                      {},
	ExchangeEchoDex:                    {},
	ExchangeBaseSwap:                   {},
	ExchangeAlienBase:                  {},
	ExchangeSwapBased:                  {},
	ExchangeRocketSwapV2:               {},
	ExchangeSpartaDex:                  {},
	ExchangeArbiDex:                    {},
	ExchangeZyberSwapV3:                {},
	ExchangeSpacefi:                    {},
	ExchangeEzkalibur:                  {},
	ExchangeMoonBase:                   {},
	ExchangeBalDex:                     {},
	ExchangeZkSwapFinance:              {},
	ExchangeScrollSwap:                 {},
	ExchangePunkSwap:                   {},
	ExchangeMetavaultV2:                {},
	ExchangeNomiswap:                   {},
	ExchangeArbswapAMM:                 {},
	ExchangeKokonutCpmm:                {},
	ExchangeKokonutCrypto:              {},
	ExchangeChronosV3:                  {},
	ExchangeRetroV3:                    {},
	ExchangeHorizonDex:                 {},
	ExchangeDoveSwapV3:                 {},
	ExchangeSushiSwapV3:                {},
	ExchangeRamsesV2:                   {},
	ExchangeDackieV3:                   {},
	ExchangeHoriza:                     {},
	ExchangeBaseSwapV3:                 {},
	ExchangeArbiDexV3:                  {},
	ExchangeWagmi:                      {},
	ExchangeMetavaultV3:                {},
	ExchangeSolidlyV3:                  {},
	ExchangeZero:                       {},
	ExchangeBalancerV2Weighted:         {},
	ExchangeBalancerV2Stable:           {},
	ExchangeBalancerV2ComposableStable: {},
	ExchangeBeethovenXWeighted:         {},
	ExchangeBeethovenXStable:           {},
	ExchangeBeethovenXComposableStable: {},
	ExchangeGyroscope2CLP:              {},
	ExchangeGyroscope3CLP:              {},
	ExchangeGyroscopeECLP:              {},
	ExchangeSynthSwapPerp:              {},
	ExchangeSwapBasedPerp:              {},
	ExchangeBMX:                        {},
	ExchangeBMXGLP:                     {},
	ExchangeFxdx:                       {},
	ExchangeQuickPerps:                 {},
	ExchangeMummyFinance:               {},
	ExchangeOpx:                        {},
	ExchangeFulcrom:                    {},
	ExchangeVodoo:                      {},
	ExchangeMakerLidoStETH:             {},
	ExchangeVelodromeV2:                {},
	ExchangeAerodrome:                  {},
	ExchangeFvm:                        {},
	ExchangeBvm:                        {},
	ExchangeMuteSwitch:                 {},
	ExchangeRetro:                      {},
	ExchangeThena:                      {},
	ExchangeThenaFusion:                {},
	ExchangePearlV2:                    {},
	ExchangeBaso:                       {},
	ExchangeLyve:                       {},
	ExchangeScale:                      {},
	ExchangeUSDFi:                      {},
	ExchangeSkydrome:                   {},
	ExchangeWombat:                     {},
	ExchangeMantisSwap:                 {},
	ExchangeSyncSwap:                   {},
	ExchangeMaverickV1:                 {},
	ExchangeKyberSwapLimitOrderDS:      {},
	ExchangeKyberPMM:                   {},
	ExchangeTraderJoeV20:               {},
	ExchangeTraderJoeV21:               {},
	ExchangeIZiSwap:                    {},
	ExchangeWooFiV2:                    {},
	ExchangeVesync:                     {},
	ExchangeDackieV2:                   {},
	ExchangeMMFV3:                      {},
	ExchangeVooi:                       {},
	ExchangePolMatic:                   {},
	ExchangeSmardex:                    {},
	ExchangeZebra:                      {},
	ExchangeZKSwap:                     {},
	ExchangeBalancerV1:                 {},
	ExchangeVelocoreV2CPMM:             {},
	ExchangeVelocoreV2WombatStable:     {},
	ExchangeAlienBaseStableSwap:        {},
	ExchangeZebraV2:                    {},
}

func IsAMMSource(exchange Exchange) bool {
	_, contained := AMMSourceSet[exchange]

	return contained
}

var RFQSourceSet = map[Exchange]struct{}{
	ExchangeKyberPMM: {},
	ExchangeSwaapV2:  {},

	ExchangeKyberSwapLimitOrderDS: {},
}

func IsRFQSource(exchange Exchange) bool {
	_, contained := RFQSourceSet[exchange]

	return contained
}

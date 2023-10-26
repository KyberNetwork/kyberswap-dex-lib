package valueobject

import (
	"hash/fnv"
	"sort"
)

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
	ExchangeCrowdswapV2   Exchange = "crowdswap-v2"
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
	ExchangeHoriza           Exchange = "horiza"
	ExchangeBaseSwapV3       Exchange = "baseswap-v3"
	ExchangeArbiDexV3        Exchange = "arbidex-v3"
	ExchangeWagmi            Exchange = "wagmi"
	ExchangeMetavaultV3      Exchange = "metavault-v3"

	ExchangeBalancer                 Exchange = "balancer"
	ExchangeBalancerComposableStable Exchange = "balancer-composable-stable"
	ExchangeBeethovenX               Exchange = "beethovenx"

	ExchangeDodo Exchange = "dodo"

	ExchangeSynthSwapPerp Exchange = "synthswap-perp"
	ExchangeSwapBasedPerp Exchange = "swapbased-perp"
	ExchangeGMX           Exchange = "gmx"
	ExchangeMadMex        Exchange = "madmex"
	ExchangeMetavault     Exchange = "metavault"
	ExchangeBMX           Exchange = "bmx"
	ExchangeBMXGLP        Exchange = "bmx-glp"

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
	ExchangeThenaFusion Exchange = "thena-fusion"
	ExchangePearl       Exchange = "pearl"
	ExchangePearlV2     Exchange = "pearl-v2"
	ExchangeBaso        Exchange = "baso"
	ExchangeLyve        Exchange = "lyve"
	ExchangeScale       Exchange = "scale"
	ExchangeUSDFi       Exchange = "usdfi"
	ExchangeSkydrome    Exchange = "skydrome"

	ExchangePlatypus   Exchange = "platypus"
	ExchangeWombat     Exchange = "wombat"
	ExchangeMantisSwap Exchange = "mantisswap"

	ExchangeSyncSwap Exchange = "syncswap"

	ExchangeMaverickV1 Exchange = "maverick-v1"

	ExchangeKyberSwapLimitOrder   Exchange = "kyberswap-limit-order"
	ExchangeKyberSwapLimitOrderDS Exchange = "kyberswap-limit-order-v2"

	ExchangeKyberPMM Exchange = "kyber-pmm"

	ExchangeTraderJoeV20 Exchange = "traderjoe-v20"
	ExchangeTraderJoeV21 Exchange = "traderjoe-v21"

	ExchangeIZiSwap Exchange = "iziswap"

	ExchangeWooFiV2  Exchange = "woofi-v2"
	ExchangeVesync   Exchange = "vesync"
	ExchangeDackieV2 Exchange = "dackie-v2"

	ExchangeMMFV3 Exchange = "mmf-v3"

	ExchangeVooi Exchange = "vooi"

	ExchangePolMatic Exchange = "pol-matic"
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
	ExchangeCrowdswapV2:              {},
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
	ExchangeBMXGLP:                   {},
	ExchangeBMX:                      {},
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
	ExchangeThenaFusion:              {},
	ExchangePearl:                    {},
	ExchangePearlV2:                  {},
	ExchangeDystopia:                 {},
	ExchangeChronos:                  {},
	ExchangeRamses:                   {},
	ExchangeVelocore:                 {},
	ExchangeMuteSwitch:               {},
	ExchangePlatypus:                 {},
	ExchangeWombat:                   {},
	ExchangeMantisSwap:               {},
	ExchangeSyncSwap:                 {},
	ExchangeKyberSwapLimitOrder:      {},
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
	ExchangeTraderJoeV20:             {},
	ExchangeTraderJoeV21:             {},
	ExchangeSpartaDex:                {},
	ExchangeArbiDex:                  {},
	ExchangeAlienBaseStableSwap:      {},
	ExchangeIZiSwap:                  {},
	ExchangeZyberSwapV3:              {},
	ExchangeSpacefi:                  {},
	ExchangeHoriza:                   {},
	ExchangeLyve:                     {},
	ExchangeBaseSwapV3:               {},
	ExchangeWooFiV2:                  {},
	ExchangeEzkalibur:                {},
	ExchangeVesync:                   {},
	ExchangeArbiDexV3:                {},
	ExchangeWagmi:                    {},
	ExchangeDackieV2:                 {},
	ExchangeMoonBase:                 {},
	ExchangeScale:                    {},
	ExchangeBalDex:                   {},
	ExchangeSynthSwapPerp:            {},
	ExchangeSwapBasedPerp:            {},
	ExchangeMMFV3:                    {},
	ExchangeUSDFi:                    {},
	ExchangeZkSwapFinance:            {},
	ExchangeSkydrome:                 {},
	ExchangeScrollSwap:               {},
	ExchangePunkSwap:                 {},
	ExchangeVooi:                     {},
	ExchangeMetavaultV2:              {},
	ExchangeMetavaultV3:              {},
	ExchangeNomiswap:                 {},
	ExchangeArbswapAMM:               {},
	ExchangePolMatic:                 {},
}

func IsAnExchange(exchange Exchange) bool {
	var contained bool
	_, contained = AMMSourceSet[exchange]
	if contained {
		return true
	}

	_, contained = RFQSourceSet[exchange]
	return contained
}

func GetSourcesAsSlice(sources map[Exchange]struct{}) []string {
	result := make([]string, len(sources))
	count := 0
	for src := range sources {
		result[count] = string(src)
		count = count + 1
	}
	return result
}

var RFQSourceSet = map[Exchange]struct{}{
	ExchangeKyberPMM: {},

	ExchangeKyberSwapLimitOrderDS: {},
}

func IsRFQSource(exchange Exchange) bool {
	_, contained := RFQSourceSet[exchange]

	return contained
}

// HashSources unique, then sort and has the slice string
func HashSources(sources []string) uint64 {
	// Step 1: Make the elements unique
	uniqueMap := make(map[string]bool)
	for _, str := range sources {
		uniqueMap[str] = true
	}

	// Extract the unique elements
	uniqueSlice := make([]string, 0, len(uniqueMap))
	for str := range uniqueMap {
		uniqueSlice = append(uniqueSlice, str)
	}

	// Step 2: Sort the unique elements stably
	sort.Strings(uniqueSlice)

	// Step 3: Hash
	h := fnv.New64()
	for _, str := range uniqueSlice {
		_, _ = h.Write([]byte(str))
	}
	return h.Sum64()
}

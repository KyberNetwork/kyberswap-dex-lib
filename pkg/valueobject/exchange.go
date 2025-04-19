package valueobject

type Exchange string

var (
	ExchangeSushiSwap       Exchange = "sushiswap"
	ExchangeTrisolaris      Exchange = "trisolaris"
	ExchangeWannaSwap       Exchange = "wannaswap"
	ExchangeNearPad         Exchange = "nearpad"
	ExchangePangolin        Exchange = "pangolin"
	ExchangeTraderJoe       Exchange = "traderjoe"
	ExchangeLydia           Exchange = "lydia"
	ExchangeYetiSwap        Exchange = "yetiswap"
	ExchangeApeSwap         Exchange = "apeswap"
	ExchangeJetSwap         Exchange = "jetswap"
	ExchangeMDex            Exchange = "mdex"
	ExchangePancake         Exchange = "pancake"
	ExchangeWault           Exchange = "wault"
	ExchangePancakeLegacy   Exchange = "pancake-legacy"
	ExchangeBiSwap          Exchange = "biswap"
	ExchangePantherSwap     Exchange = "pantherswap"
	ExchangeVVS             Exchange = "vvs"
	ExchangeCronaSwap       Exchange = "cronaswap"
	ExchangeCrodex          Exchange = "crodex"
	ExchangeMMF             Exchange = "mmf"
	ExchangeEmpireDex       Exchange = "empiredex"
	ExchangePhotonSwap      Exchange = "photonswap"
	ExchangeUniSwap         Exchange = "uniswap"
	ExchangeUniSwapV1       Exchange = "uniswap-v1"
	ExchangeUniSwapV2       Exchange = "uniswap-v2"
	ExchangeShibaSwap       Exchange = "shibaswap"
	ExchangeDefiSwap        Exchange = "defiswap"
	ExchangeSpookySwap      Exchange = "spookyswap"
	ExchangeSpiritSwap      Exchange = "spiritswap"
	ExchangePaintSwap       Exchange = "paintswap"
	ExchangeMorpheus        Exchange = "morpheus"
	ExchangeValleySwap      Exchange = "valleyswap"
	ExchangeYuzuSwap        Exchange = "yuzuswap"
	ExchangeGemKeeper       Exchange = "gemkeeper"
	ExchangeLizard          Exchange = "lizard"
	ExchangeValleySwapV2    Exchange = "valleyswap-v2"
	ExchangeZipSwap         Exchange = "zipswap"
	ExchangeQuickSwap       Exchange = "quickswap"
	ExchangePolycat         Exchange = "polycat"
	ExchangeDFYN            Exchange = "dfyn"
	ExchangePolyDex         Exchange = "polydex"
	ExchangeGravity         Exchange = "gravity"
	ExchangeCometh          Exchange = "cometh"
	ExchangeDinoSwap        Exchange = "dinoswap"
	ExchangeKrptoDex        Exchange = "kryptodex"
	ExchangeSafeSwap        Exchange = "safeswap"
	ExchangeSwapr           Exchange = "swapr"
	ExchangeWagyuSwap       Exchange = "wagyuswap"
	ExchangeAstroSwap       Exchange = "astroswap"
	ExchangeCamelot         Exchange = "camelot"
	ExchangeFraxSwap        Exchange = "fraxswap"
	ExchangeBlasterSwap     Exchange = "blasterswap"
	ExchangeBlastDex        Exchange = "blastdex"
	ExchangeHyperBlast      Exchange = "hyper-blast"
	ExchangeSquadSwap       Exchange = "squadswap"
	ExchangeSquadSwapV2     Exchange = "squadswap-v2"
	ExchangeSquadSwapV3     Exchange = "squadswap-v3"
	ExchangeLiquidusFinance Exchange = "liquidus-finance"

	ExchangeOneSwap    Exchange = "oneswap"
	ExchangeNerve      Exchange = "nerve"
	ExchangeIronStable Exchange = "iron-stable"
	ExchangeSynapse    Exchange = "synapse"
	ExchangeSaddle     Exchange = "saddle"
	ExchangeAxial      Exchange = "axial"

	ExchangeCurve         Exchange = "curve"
	ExchangeEllipsis      Exchange = "ellipsis"
	ExchangePancakeStable Exchange = "pancake-stable"
	ExchangeZkSwapStable  Exchange = "zkswap-stable"

	ExchangeCurveStablePlain  Exchange = "curve-stable-plain"
	ExchangeCurveStableNg     Exchange = "curve-stable-ng"
	ExchangeCurveStableMetaNg Exchange = "curve-stable-meta-ng"
	ExchangeCurveTriCryptoNg  Exchange = "curve-tricrypto-ng"
	ExchangeCurveTwoCryptoNg  Exchange = "curve-twocrypto-ng"
	ExchangeCurveLlamma       Exchange = "curve-llamma"
	ExchangeCurveLending      Exchange = "curve-lending"

	ExchangeUniSwapV3        Exchange = "uniswapv3"
	ExchangeKyberswapElastic Exchange = "kyberswap-elastic"
	ExchangeRoguex           Exchange = "roguex"
	ExchangeEqualizerCL      Exchange = "equalizer-cl"

	ExchangeBalancer   Exchange = "balancer"
	ExchangeBeethovenX Exchange = "beethovenx"

	ExchangeDodo               Exchange = "dodo"
	ExchangeDodoClassical      Exchange = "dodo-classical"
	ExchangeDodoPrivatePool    Exchange = "dodo-dpp"
	ExchangeDodoStablePool     Exchange = "dodo-dsp"
	ExchangeDodoVendingMachine Exchange = "dodo-dvm"

	ExchangeGMX       Exchange = "gmx"
	ExchangeMadMex    Exchange = "madmex"
	ExchangeMetavault Exchange = "metavault"

	ExchangeSynthetix Exchange = "synthetix"

	ExchangeMakerPSM Exchange = "maker-psm"

	ExchangeMakerLido Exchange = "lido"

	ExchangeDMM             Exchange = "dmm"
	ExchangeKyberSwap       Exchange = "kyberswap"
	ExchangeKyberSwapStatic Exchange = "kyberswap-static"

	ExchangeVelodrome    Exchange = "velodrome"
	ExchangeDystopia     Exchange = "dystopia"
	ExchangeChronos      Exchange = "chronos"
	ExchangeRamses       Exchange = "ramses"
	ExchangeVelocore     Exchange = "velocore"
	ExchangePearl        Exchange = "pearl"
	ExchangeDegenExpress Exchange = "degen-express"

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
	ExchangePharaohV2   Exchange = "pharaoh-v2"
	ExchangeDackieV3    Exchange = "dackie-v3"
	ExchangeZkSwapV3    Exchange = "zkswap-v3"
	ExchangeHoriza      Exchange = "horiza"
	ExchangeBaseSwapV3  Exchange = "baseswap-v3"
	ExchangeArbiDexV3   Exchange = "arbidex-v3"
	ExchangeSuperSwapV3 Exchange = "superswap-v3"
	ExchangeWagmi       Exchange = "wagmi"
	ExchangeMetavaultV3 Exchange = "metavault-v3"
	ExchangeSolidlyV2   Exchange = "solidly-v2"
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

	ExchangeBalancerV3Weighted   Exchange = "balancer-v3-weighted"
	ExchangeBalancerV3Stable     Exchange = "balancer-v3-stable"
	ExchangeBalancerV3ECLP       Exchange = "balancer-v3-eclp"
	ExchangeBeethovenXV3Weighted Exchange = "beethovenx-v3-weighted"
	ExchangeBeethovenXV3Stable   Exchange = "beethovenx-v3-stable"

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
	ExchangeZkEraFinance  Exchange = "zkera-finance"
	ExchangeNavigator     Exchange = "navigator"
	ExchangeAmped         Exchange = "amped"

	ExchangeMakerLidoStETH Exchange = "lido-steth"

	ExchangeVelodromeV2   Exchange = "velodrome-v2"
	ExchangeAerodrome     Exchange = "aerodrome"
	ExchangeFvm           Exchange = "fvm"
	ExchangeBvm           Exchange = "bvm"
	ExchangeMuteSwitch    Exchange = "muteswitch"
	ExchangeRetro         Exchange = "retro"
	ExchangeThena         Exchange = "thena"
	ExchangeThenaFusion   Exchange = "thena-fusion"
	ExchangeThenaFusionV3 Exchange = "thena-fusion-v3"
	ExchangePearlV2       Exchange = "pearl-v2"
	ExchangeBaso          Exchange = "baso"
	ExchangeLyve          Exchange = "lyve"
	ExchangeScale         Exchange = "scale"
	ExchangeUSDFi         Exchange = "usdfi"
	ExchangeSkydrome      Exchange = "skydrome"
	ExchangeEqual         Exchange = "equal"
	ExchangeLynexV1       Exchange = "lynex-v1"
	ExchangeSkydromeV2    Exchange = "skydrome-v2"
	ExchangeKellerFinance Exchange = "keller-finance"

	ExchangeWombat     Exchange = "wombat"
	ExchangeMantisSwap Exchange = "mantisswap"

	ExchangeSyncSwap          Exchange = "syncswap"
	ExchangeSyncSwapCL        Exchange = "syncswap-cl"
	ExchangeSyncSwapV2Classic Exchange = "syncswapv2-classic"
	ExchangeSyncSwapV2Stable  Exchange = "syncswapv2-stable"
	ExchangeSyncSwapV2Aqua    Exchange = "syncswapv2-aqua"

	ExchangeMaverickV1 Exchange = "maverick-v1"

	ExchangeKyberSwapLimitOrderDS Exchange = "kyberswap-limit-order-v2"

	ExchangeKyberPMM   Exchange = "kyber-pmm"
	ExchangeSwaapV2    Exchange = "swaap-v2"
	ExchangeHashflowV3 Exchange = "hashflow-v3"
	ExchangeNativeV1   Exchange = "native-v1"
	ExchangeNativeV2   Exchange = "native-v2"
	ExchangeBebop      Exchange = "bebop"
	ExchangeDexalot    Exchange = "dexalot"
	ExchangeOneBit     Exchange = "onebit"

	ExchangeTraderJoeV20 Exchange = "traderjoe-v20"
	ExchangeTraderJoeV21 Exchange = "traderjoe-v21"
	ExchangeTraderJoeV22 Exchange = "traderjoe-v22"

	ExchangeIZiSwap Exchange = "iziswap"

	ExchangeWooFiV2  Exchange = "woofi-v2"
	ExchangeWooFiV3  Exchange = "woofi-v3"
	ExchangeVesync   Exchange = "vesync"
	ExchangeDackieV2 Exchange = "dackie-v2"

	ExchangeMMFV3 Exchange = "mmf-v3"

	ExchangeVooi Exchange = "vooi"

	ExchangePolMatic Exchange = "pol-matic"

	ExchangeSmardex Exchange = "smardex"

	ExchangeIntegral Exchange = "integral"

	ExchangeZebra  Exchange = "zebra"
	ExchangeZKSwap Exchange = "zkswap"

	ExchangeBalancerV1 Exchange = "balancer-v1"

	ExchangeVelocoreV2CPMM         Exchange = "velocore-v2-cpmm"
	ExchangeVelocoreV2WombatStable Exchange = "velocore-v2-wombat-stable"
	ExchangeAlienBaseStableSwap    Exchange = "alien-base-stableswap"

	ExchangeBladeSwap Exchange = "blade-swap"

	ExchangeNile   Exchange = "nile"
	ExchangeNileV2 Exchange = "nile-v2"

	ExchangePharaoh Exchange = "pharaoh"

	ExchangeBlueprint Exchange = "blueprint"

	ExchangeNuri   Exchange = "nuri"
	ExchangeNuriV2 Exchange = "nuri-v2"

	ExchangeBancorV21 Exchange = "bancor-v21"
	ExchangeBancorV3  Exchange = "bancor-v3"

	ExchangeEtherfiEETH  Exchange = "etherfi-eeth"
	ExchangeEtherfiWEETH Exchange = "etherfi-weeth"

	ExchangeKelpRSETH Exchange = "kelp-rseth"

	ExchangeRocketPoolRETH Exchange = "rocketpool-reth"

	ExchangeEthenaSusde Exchange = "ethena-susde"

	ExchangeMakerSavingsDai Exchange = "maker-savingsdai"

	ExchangeRenzoEZETH Exchange = "renzo-ezeth"

	ExchangeSwellSWETH  Exchange = "swell-sweth"
	ExchangeSwellRSWETH Exchange = "swell-rsweth"

	ExchangeBedrockUniETH Exchange = "bedrock-unieth"

	ExchangePufferPufETH Exchange = "puffer-pufeth"

	ExchangeThrusterV2      Exchange = "thruster-v2"
	ExchangeThrusterV2Degen Exchange = "thruster-v2-degen"
	ExchangeDyorSwap        Exchange = "dyor-swap"
	ExchangeSwapBlast       Exchange = "swap-blast"
	ExchangeMonoswap        Exchange = "monoswap"
	ExchangeThrusterV3      Exchange = "thruster-v3"
	ExchangeCyberblastV3    Exchange = "cyberblast-v3"
	ExchangeMonoswapV3      Exchange = "monoswap-v3"

	ExchangeAgniFinance    Exchange = "agni-finance"
	ExchangeMerchantMoe    Exchange = "merchant-moe"
	ExchangeFusionX        Exchange = "fusion-x"
	ExchangeFusionXV3      Exchange = "fusion-x-v3"
	ExchangeKTX            Exchange = "ktx"
	ExchangeTsunamiX       Exchange = "tsunami-x"
	ExchangeCleopatra      Exchange = "cleopatra"
	ExchangeCleopatraV2    Exchange = "cleopatra-v2"
	ExchangeStratumFinance Exchange = "stratum-finance"
	ExchangeMVM            Exchange = "mvm"
	ExchangeButterFi       Exchange = "butter-fi"
	ExchangeMerchantMoeV22 Exchange = "merchant-moe-v22"
	ExchangeNomiswapStable Exchange = "nomiswap-stable"

	ExchangeStationDexV2 Exchange = "station-dex-v2"
	ExchangeAbstra       Exchange = "abstra"
	ExchangeRevoSwap     Exchange = "revo-swap"
	ExchangePotatoSwap   Exchange = "potato-swap"
	ExchangeXLayerSwap   Exchange = "xlayer-swap"
	ExchangeStationDexV3 Exchange = "station-dex-v3"

	ExchangeVelodromeCL  Exchange = "velodrome-cl"
	ExchangeAerodromeCL  Exchange = "aerodrome-cl"
	ExchangeVelodromeCL2 Exchange = "velodrome-cl-2"

	ExchangeLineHubV2 Exchange = "linehub-v2"
	ExchangeLineHubV3 Exchange = "linehub-v3"
	ExchangeWigoSwap  Exchange = "wigo-swap"

	ExchangeInfusion Exchange = "infusion"
	ExchangeSoSwap   Exchange = "soswap"

	ExchangeSpookySwapV3 Exchange = "spookyswap-v3"
	ExchangeThick        Exchange = "thick"
	ExchangeE3           Exchange = "e3"
	ExchangeAlienBaseCL  Exchange = "alien-base-cl"
	ExchangeKinetixV2    Exchange = "kinetix-v2"
	ExchangeKinetixV3    Exchange = "kinetix-v3"
	Exchange9mmProV2     Exchange = "9mm-pro-v2"
	Exchange9mmProV3     Exchange = "9mm-pro-v3"

	ExchangeAlienBaseDegen Exchange = "alien-base-degen"
	ExchangeKoiCL          Exchange = "koi-cl"
	ExchangeTokan          Exchange = "tokan-exchange"
	ExchangeSectaV2        Exchange = "secta-v2"
	ExchangeSectaV3        Exchange = "secta-v3"
	ExchangeQuickSwapUniV3 Exchange = "quickswap-uni-v3"
	ExchangeHoldFun        Exchange = "hold-fun"
	ExchangeOwlSwapV3      Exchange = "owlswap-v3"

	ExchangeAmbient    Exchange = "ambient"
	ExchangeMaverickV2 Exchange = "maverick-v2"

	ExchangeEtherVista       Exchange = "ether-vista"
	ExchangeLitePSM          Exchange = "lite-psm"
	ExchangeMkrSky           Exchange = "mkr-sky"
	ExchangeDaiUsds          Exchange = "dai-usds"
	ExchangeUsd0PP           Exchange = "usd0pp"
	ExchangeWBETH            Exchange = "wbeth"
	ExchangeOETH             Exchange = "oeth"
	ExchangeRingSwap         Exchange = "ringswap"
	ExchangePrimeETH         Exchange = "primeeth"
	ExchangeStaderETHx       Exchange = "staderethx"
	ExchangeFrxETH           Exchange = "frxeth"
	ExchangeFluidVaultT1     Exchange = "fluid-vault-t1"
	ExchangeFluidDexT1       Exchange = "fluid-dex-t1"
	ExchangeMantleETH        Exchange = "meth"
	ExchangeOndoUSDY         Exchange = "ondo-usdy"
	ExchangeClipper          Exchange = "clipper"
	ExchangeDeltaSwapV1      Exchange = "deltaswap-v1"
	ExchangeSfrxETH          Exchange = "sfrxeth"
	ExchangeSfrxETHConvertor Exchange = "sfrxeth-convertor"
	ExchangeEtherfiVampire   Exchange = "etherfi-vampire"
	ExchangeSwapsicle        Exchange = "swapsicle"
	ExchangeFenix            Exchange = "fenix"
	ExchangeBlade            Exchange = "blade"
	ExchangeSilverSwap       Exchange = "silverswap"
	ExchangeScribe           Exchange = "scribe"
	ExchangeHorizonIntegral  Exchange = "horizon-integral"
	ExchangeMxTrading        Exchange = "mx-trading"
	ExchangeLO1inch          Exchange = "lo1inch"
	ExchangeVirtualFun       Exchange = "virtual-fun"
	ExchangeMetropolis       Exchange = "metropolis"
	ExchangeMetropolisLB     Exchange = "metropolis-lb"
	ExchangeSonicMarket      Exchange = "sonic-market"
	ExchangeSboom            Exchange = "sboom"
	ExchangeBeetsSS          Exchange = "beets-ss"
	ExchangeShadowDex        Exchange = "shadow-dex"
	ExchangeSwapXCL          Exchange = "swap-x-cl"
	ExchangeSwapXV2          Exchange = "swap-x-v2"
	ExchangeMemeBox          Exchange = "memebox"
	ExchangeShadowLegacy     Exchange = "shadow-legacy"
	ExchangeDeFive           Exchange = "defive"

	ExchangeKodiakV2                 Exchange = "kodiak-v2"
	ExchangeKodiakV3                 Exchange = "kodiak-v3"
	ExchangeBeraSwapWeighted         Exchange = "beraswap-weighted"
	ExchangeBeraSwapStable           Exchange = "beraswap-stable"
	ExchangeBeraSwapComposableStable Exchange = "beraswap-composable-stable"
	ExchangeBurrBearWeighted         Exchange = "burrbear-weighted"
	ExchangeBurrBearStable           Exchange = "burrbear-stable"
	ExchangeBurrBearComposableStable Exchange = "burrbear-composable-stable"
	ExchangeBeracaine                Exchange = "beracaine"
	ExchangeBulla                    Exchange = "bulla"
	ExchangeMemeswap                 Exchange = "memeswap"
	ExchangePandaFun                 Exchange = "panda-fun"
	ExchangeHoney                    Exchange = "honey"

	ExchangeKatanaV2 Exchange = "katana-v2"
	ExchangeKatanaV3 Exchange = "katana-v3"

	ExchangeUcsFinance Exchange = "ucs-finance"

	ExchangeUniswapV4        Exchange = "uniswap-v4"
	ExchangeUniswapV4BunniV2 Exchange = "uniswap-v4-bunni-v2"
	ExchangeUniswapV4Kem     Exchange = "uniswap-v4-kem"

	ExchangeFakePool      Exchange = "fake-pool"
	ExchangeEtherFieBTC   Exchange = "etherfi-ebtc"
	ExchangeOvernightUsdp Exchange = "overnight-usdp"
	ExchangeSavingsUSDS   Exchange = "savings-usds"
	ExchangeSkyPSM        Exchange = "sky-psm"
	ExchangeEulerSwap     Exchange = "euler-swap"
	ExchangeEkubo         Exchange = "ekubo"
	ExchangeBeefySonic    Exchange = "beefy-sonic"
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
	ExchangeUniSwapV1:                  {},
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
	ExchangeCurveStablePlain:           {},
	ExchangeCurveStableNg:              {},
	ExchangeCurveStableMetaNg:          {},
	ExchangeCurveTriCryptoNg:           {},
	ExchangeCurveTwoCryptoNg:           {},
	ExchangeCurveLlamma:                {},
	ExchangeCurveLending:               {},
	ExchangeEllipsis:                   {},
	ExchangePancakeStable:              {},
	ExchangeZkSwapStable:               {},
	ExchangeUniSwapV3:                  {},
	ExchangeKyberswapElastic:           {},
	ExchangeBalancer:                   {},
	ExchangeBeethovenX:                 {},
	ExchangeDodo:                       {},
	ExchangeDodoClassical:              {},
	ExchangeDodoPrivatePool:            {},
	ExchangeDodoStablePool:             {},
	ExchangeDodoVendingMachine:         {},
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
	ExchangeZkSwapV3:                   {},
	ExchangeHoriza:                     {},
	ExchangeBaseSwapV3:                 {},
	ExchangeArbiDexV3:                  {},
	ExchangeSuperSwapV3:                {},
	ExchangeWagmi:                      {},
	ExchangeMetavaultV3:                {},
	ExchangeSolidlyV2:                  {},
	ExchangeSolidlyV3:                  {},
	ExchangeZero:                       {},
	ExchangeBalancerV2Weighted:         {},
	ExchangeBalancerV2Stable:           {},
	ExchangeBalancerV2ComposableStable: {},
	ExchangeBalancerV3Weighted:         {},
	ExchangeBalancerV3Stable:           {},
	ExchangeBalancerV3ECLP:             {},
	ExchangeBeethovenXV3Weighted:       {},
	ExchangeBeethovenXV3Stable:         {},
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
	ExchangeThenaFusionV3:              {},
	ExchangePearlV2:                    {},
	ExchangeBaso:                       {},
	ExchangeLyve:                       {},
	ExchangeScale:                      {},
	ExchangeUSDFi:                      {},
	ExchangeSkydrome:                   {},
	ExchangeWombat:                     {},
	ExchangeMantisSwap:                 {},
	ExchangeSyncSwap:                   {},
	ExchangeSyncSwapCL:                 {},
	ExchangeSyncSwapV2Classic:          {},
	ExchangeSyncSwapV2Stable:           {},
	ExchangeSyncSwapV2Aqua:             {},
	ExchangeMaverickV1:                 {},
	ExchangeKyberSwapLimitOrderDS:      {},
	ExchangeTraderJoeV20:               {},
	ExchangeTraderJoeV21:               {},
	ExchangeTraderJoeV22:               {},
	ExchangeIZiSwap:                    {},
	ExchangeWooFiV2:                    {},
	ExchangeWooFiV3:                    {},
	ExchangeVesync:                     {},
	ExchangeDackieV2:                   {},
	ExchangeMMFV3:                      {},
	ExchangeVooi:                       {},
	ExchangePolMatic:                   {},
	ExchangeSmardex:                    {},
	ExchangeIntegral:                   {},
	ExchangeZebra:                      {},
	ExchangeZKSwap:                     {},
	ExchangeBalancerV1:                 {},
	ExchangeVelocoreV2CPMM:             {},
	ExchangeVelocoreV2WombatStable:     {},
	ExchangeAlienBaseStableSwap:        {},
	ExchangeZebraV2:                    {},
	ExchangePharaohV2:                  {},
	ExchangeNile:                       {},
	ExchangeNileV2:                     {},
	ExchangePharaoh:                    {},
	ExchangeBlueprint:                  {},
	ExchangeNuri:                       {},
	ExchangeNuriV2:                     {},
	ExchangeBancorV21:                  {},
	ExchangeBancorV3:                   {},
	ExchangeEtherfiEETH:                {},
	ExchangeEtherfiWEETH:               {},
	ExchangeZkEraFinance:               {},
	ExchangeNavigator:                  {},
	ExchangeAmped:                      {},
	ExchangeKelpRSETH:                  {},
	ExchangeRocketPoolRETH:             {},
	ExchangeEthenaSusde:                {},
	ExchangeMakerSavingsDai:            {},
	ExchangeSwellSWETH:                 {},
	ExchangeSwellRSWETH:                {},
	ExchangeBedrockUniETH:              {},
	ExchangePufferPufETH:               {},
	ExchangeRingSwap:                   {},
	ExchangeThrusterV2:                 {},
	ExchangeThrusterV2Degen:            {},
	ExchangeDyorSwap:                   {},
	ExchangeSwapBlast:                  {},
	ExchangeMonoswap:                   {},
	ExchangeThrusterV3:                 {},
	ExchangeCyberblastV3:               {},
	ExchangeMonoswapV3:                 {},
	ExchangeBlasterSwap:                {},
	ExchangeBladeSwap:                  {},
	ExchangeBlastDex:                   {},
	ExchangeHyperBlast:                 {},
	ExchangeRoguex:                     {},
	ExchangeEqual:                      {},
	ExchangeEqualizerCL:                {},
	ExchangeAgniFinance:                {},
	ExchangeMerchantMoe:                {},
	ExchangeFusionX:                    {},
	ExchangeFusionXV3:                  {},
	ExchangeKTX:                        {},
	ExchangeTsunamiX:                   {},
	ExchangeCleopatra:                  {},
	ExchangeCleopatraV2:                {},
	ExchangeStratumFinance:             {},
	ExchangeMVM:                        {},
	ExchangeButterFi:                   {},
	ExchangeLynexV1:                    {},
	ExchangeSkydromeV2:                 {},
	ExchangeSquadSwap:                  {},
	ExchangeSquadSwapV2:                {},
	ExchangeSquadSwapV3:                {},
	ExchangeMerchantMoeV22:             {},
	ExchangeLiquidusFinance:            {},
	ExchangeNomiswapStable:             {},
	ExchangeKellerFinance:              {},
	ExchangeRenzoEZETH:                 {},
	ExchangeDegenExpress:               {},
	ExchangeStationDexV2:               {},
	ExchangeAbstra:                     {},
	ExchangeRevoSwap:                   {},
	ExchangePotatoSwap:                 {},
	ExchangeXLayerSwap:                 {},
	ExchangeStationDexV3:               {},
	ExchangeVelodromeCL:                {},
	ExchangeAerodromeCL:                {},
	ExchangeVelodromeCL2:               {},
	ExchangeLineHubV2:                  {},
	ExchangeLineHubV3:                  {},
	ExchangeWigoSwap:                   {},
	ExchangeInfusion:                   {},
	ExchangeSoSwap:                     {},
	ExchangeSpookySwapV3:               {},
	ExchangeThick:                      {},
	ExchangeE3:                         {},
	ExchangeAlienBaseCL:                {},
	ExchangeKinetixV2:                  {},
	ExchangeKinetixV3:                  {},
	Exchange9mmProV2:                   {},
	Exchange9mmProV3:                   {},
	ExchangeAlienBaseDegen:             {},
	ExchangeKoiCL:                      {},
	ExchangeTokan:                      {},
	ExchangeSectaV2:                    {},
	ExchangeSectaV3:                    {},
	ExchangeQuickSwapUniV3:             {},
	ExchangeHoldFun:                    {},
	ExchangeOwlSwapV3:                  {},
	ExchangeAmbient:                    {},
	ExchangeMaverickV2:                 {},
	ExchangeEtherVista:                 {},
	ExchangeLitePSM:                    {},
	ExchangeMkrSky:                     {},
	ExchangeDaiUsds:                    {},
	ExchangeFluidVaultT1:               {},
	ExchangeFluidDexT1:                 {},
	ExchangeUsd0PP:                     {},
	ExchangeWBETH:                      {},
	ExchangeOETH:                       {},
	ExchangeRingSwap:                   {},
	ExchangePrimeETH:                   {},
	ExchangeStaderETHx:                 {},
	ExchangeFrxETH:                     {},
	ExchangeMantleETH:                  {},
	ExchangeOndoUSDY:                   {},
	ExchangeDeltaSwapV1:                {},
	ExchangeSfrxETH:                    {},
	ExchangeSfrxETHConvertor:           {},
	ExchangeEtherfiVampire:             {},
	ExchangeSwapsicle:                  {},
	ExchangeFenix:                      {},
	ExchangeBlade:                      {},
	ExchangeSilverSwap:                 {},
	ExchangeScribe:                     {},
	ExchangeHorizonIntegral:            {},
	ExchangeVirtualFun:                 {},
	ExchangeMetropolis:                 {},
	ExchangeMetropolisLB:               {},
	ExchangeSonicMarket:                {},
	ExchangeSboom:                      {},
	ExchangeBeetsSS:                    {},
	ExchangeShadowDex:                  {},
	ExchangeSwapXCL:                    {},
	ExchangeSwapXV2:                    {},
	ExchangeMemeBox:                    {},
	ExchangeShadowLegacy:               {},
	ExchangeDeFive:                     {},
	ExchangeKodiakV2:                   {},
	ExchangeKodiakV3:                   {},
	ExchangeBeraSwapWeighted:           {},
	ExchangeBeraSwapStable:             {},
	ExchangeBeraSwapComposableStable:   {},
	ExchangeBurrBearWeighted:           {},
	ExchangeBurrBearStable:             {},
	ExchangeBurrBearComposableStable:   {},
	ExchangeBeracaine:                  {},
	ExchangeBulla:                      {},
	ExchangeMemeswap:                   {},
	ExchangePandaFun:                   {},
	ExchangeHoney:                      {},
	ExchangeKatanaV2:                   {},
	ExchangeKatanaV3:                   {},
	ExchangeUniswapV4:                  {},
	ExchangeUniswapV4BunniV2:           {},
	ExchangeUniswapV4Kem:               {},
	ExchangeEtherFieBTC:                {},
	ExchangeOvernightUsdp:              {},
	ExchangeSavingsUSDS:                {},
	ExchangeSkyPSM:                     {},
	ExchangeEulerSwap:                  {},
	ExchangeEkubo:                      {},
	ExchangeBeefySonic:                 {},
}

func IsAMMSource(exchange Exchange) bool {
	_, contained := AMMSourceSet[exchange]

	return contained
}

var RFQSourceSet = map[Exchange]struct{}{
	ExchangeKyberPMM: {},

	ExchangeKyberSwapLimitOrderDS: {},

	ExchangeSwaapV2:    {},
	ExchangeHashflowV3: {},
	ExchangeNativeV1:   {},
	ExchangeNativeV2:   {},
	ExchangeBebop:      {},
	ExchangeClipper:    {},
	ExchangeDexalot:    {},
	ExchangeMxTrading:  {},
	ExchangeLO1inch:    {},
	ExchangeOneBit:     {},
}

func IsRFQSource(exchange Exchange) bool {
	_, contained := RFQSourceSet[exchange]

	return contained
}

// SingleSwapSourceSet is a set of exchanges that
// only allow a single swap in a route.
var SingleSwapSourceSet = map[Exchange]struct{}{
	ExchangeClipper:       {},
	ExchangeBebop:         {},
	ExchangeOvernightUsdp: {},
}

func IsSingleSwapSource(exchange Exchange) bool {
	_, contained := SingleSwapSourceSet[exchange]

	return contained
}

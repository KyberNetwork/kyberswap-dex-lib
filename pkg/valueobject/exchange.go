package valueobject

type Exchange string

const (
	Exchange9mmProV2                   = "9mm-pro-v2"
	Exchange9mmProV3                   = "9mm-pro-v3"
	ExchangeAbstra                     = "abstra"
	ExchangeAerodrome                  = "aerodrome"
	ExchangeAerodromeCL                = "aerodrome-cl"
	ExchangeAgniFinance                = "agni-finance"
	ExchangeAlienBase                  = "alien-base"
	ExchangeAlienBaseCL                = "alien-base-cl"
	ExchangeAlienBaseDegen             = "alien-base-degen"
	ExchangeAlienBaseStableSwap        = "alien-base-stableswap"
	ExchangeAmbient                    = "ambient"
	ExchangeAmped                      = "amped"
	ExchangeApeSwap                    = "apeswap"
	ExchangeArbiDex                    = "arbi-dex"
	ExchangeArbiDexV3                  = "arbidex-v3"
	ExchangeArbswapAMM                 = "arbswap-amm"
	ExchangeArenaDex                   = "arenadex"
	ExchangeArenaDexV2                 = "arenadex-v2"
	ExchangeAstroSwap                  = "astroswap"
	ExchangeAxial                      = "axial"
	ExchangeBabySwap                   = "babyswap"
	ExchangeBakerySwap                 = "bakeryswap"
	ExchangeBMX                        = "bmx"
	ExchangeBMXGLP                     = "bmx-glp"
	ExchangeBabyDogeSwap               = "babydogeswap"
	ExchangeBalDex                     = "baldex"
	ExchangeBalancer                   = "balancer"
	ExchangeBalancerV1                 = "balancer-v1"
	ExchangeBalancerV2ComposableStable = "balancer-v2-composable-stable"
	ExchangeBalancerV2Stable           = "balancer-v2-stable"
	ExchangeBalancerV2Weighted         = "balancer-v2-weighted"
	ExchangeBalancerV3ECLP             = "balancer-v3-eclp"
	ExchangeBalancerV3Stable           = "balancer-v3-stable"
	ExchangeBalancerV3Weighted         = "balancer-v3-weighted"
	ExchangeBancorV21                  = "bancor-v21"
	ExchangeBancorV3                   = "bancor-v3"
	ExchangeBaseSwap                   = "baseswap"
	ExchangeBaseSwapV3                 = "baseswap-v3"
	ExchangeBaso                       = "baso"
	ExchangeBebop                      = "bebop"
	ExchangeBedrockUniETH              = "bedrock-unieth"
	ExchangeBeefySonic                 = "beefy-sonic"
	ExchangeBeethovenX                 = "beethovenx"
	ExchangeBeethovenXComposableStable = "beethovenx-composable-stable"
	ExchangeBeethovenXStable           = "beethovenx-stable"
	ExchangeBeethovenXV3Stable         = "beethovenx-v3-stable"
	ExchangeBeethovenXV3Weighted       = "beethovenx-v3-weighted"
	ExchangeBeethovenXWeighted         = "beethovenx-weighted"
	ExchangeBeetsSS                    = "beets-ss"
	ExchangeBeraSwapComposableStable   = "beraswap-composable-stable"
	ExchangeBeraSwapStable             = "beraswap-stable"
	ExchangeBeraSwapWeighted           = "beraswap-weighted"
	ExchangeBeracaine                  = "beracaine"
	ExchangeBiSwap                     = "biswap"
	ExchangeBlade                      = "blade"
	ExchangeBladeSwap                  = "blade-swap"
	ExchangeBlastDex                   = "blastdex"
	ExchangeBlasterSwap                = "blasterswap"
	ExchangeBlueprint                  = "blueprint"
	ExchangeBulla                      = "bulla"
	ExchangeBurrBearComposableStable   = "burrbear-composable-stable"
	ExchangeBurrBearStable             = "burrbear-stable"
	ExchangeBurrBearWeighted           = "burrbear-weighted"
	ExchangeButterFi                   = "butter-fi"
	ExchangeBvm                        = "bvm"
	ExchangeCamelot                    = "camelot"
	ExchangeCamelotV3                  = "camelot-v3"
	ExchangeChronos                    = "chronos"
	ExchangeChronosV3                  = "chronos-v3"
	ExchangeCleopatra                  = "cleopatra"
	ExchangeCleopatraV2                = "cleopatra-v2"
	ExchangeClipper                    = "clipper"
	ExchangeCometh                     = "cometh"
	ExchangeCrodex                     = "crodex"
	ExchangeCronaSwap                  = "cronaswap"
	ExchangeCrowdswapV2                = "crowdswap-v2"
	ExchangeCurve                      = "curve"
	ExchangeCurveLending               = "curve-lending"
	ExchangeCurveLlamma                = "curve-llamma"
	ExchangeCurveStableMetaNg          = "curve-stable-meta-ng"
	ExchangeCurveStableNg              = "curve-stable-ng"
	ExchangeCurveStablePlain           = "curve-stable-plain"
	ExchangeCurveTriCryptoNg           = "curve-tricrypto-ng"
	ExchangeCurveTwoCryptoNg           = "curve-twocrypto-ng"
	ExchangeCyberblastV3               = "cyberblast-v3"
	ExchangeDFYN                       = "dfyn"
	ExchangeDMM                        = "dmm"
	ExchangeDackieV2                   = "dackie-v2"
	ExchangeDackieV3                   = "dackie-v3"
	ExchangeDaiUsds                    = "dai-usds"
	ExchangeDeFive                     = "defive"
	ExchangeDefiSwap                   = "defiswap"
	ExchangeDegenExpress               = "degen-express"
	ExchangeDeltaSwapV1                = "deltaswap-v1"
	ExchangeDexalot                    = "dexalot"
	ExchangeDinoSwap                   = "dinoswap"
	ExchangeDinosaurEggs               = "dinosaureggs"
	ExchangeDodo                       = "dodo"
	ExchangeDodoClassical              = "dodo-classical"
	ExchangeDodoPrivatePool            = "dodo-dpp"
	ExchangeDodoStablePool             = "dodo-dsp"
	ExchangeDodoVendingMachine         = "dodo-dvm"
	ExchangeDoveSwapV3                 = "doveswap-v3"
	ExchangeDyorSwap                   = "dyor-swap"
	ExchangeDystopia                   = "dystopia"
	ExchangeE3                         = "e3"
	ExchangeEchoDex                    = "echo-dex"
	ExchangeEchoDexV3                  = "echo-dex-v3"
	ExchangeEkubo                      = "ekubo"
	ExchangeEllipsis                   = "ellipsis"
	ExchangeEmpireDex                  = "empiredex"
	ExchangeEqual                      = "equal"
	ExchangeEqualizerCL                = "equalizer-cl"
	ExchangeEthenaSusde                = "ethena-susde"
	ExchangeEtherFieBTC                = "etherfi-ebtc"
	ExchangeEtherVista                 = "ether-vista"
	ExchangeEtherfiEETH                = "etherfi-eeth"
	ExchangeEtherfiVampire             = "etherfi-vampire"
	ExchangeEtherfiWEETH               = "etherfi-weeth"
	ExchangeEulerSwap                  = "euler-swap"
	ExchangeEzkalibur                  = "ezkalibur"
	ExchangeFakePool                   = "fake-pool"
	ExchangeFenix                      = "fenix"
	ExchangeFluidDexT1                 = "fluid-dex-t1"
	ExchangeFluidVaultT1               = "fluid-vault-t1"
	ExchangeFraxSwap                   = "fraxswap"
	ExchangeFrxETH                     = "frxeth"
	ExchangeFstsSwap                   = "fstsswap"
	ExchangeFulcrom                    = "fulcrom"
	ExchangeFusionX                    = "fusion-x"
	ExchangeFusionXV3                  = "fusion-x-v3"
	ExchangeFvm                        = "fvm"
	ExchangeFxdx                       = "fxdx"
	ExchangeGMX                        = "gmx"
	ExchangeGemKeeper                  = "gemkeeper"
	ExchangeGravity                    = "gravity"
	ExchangeGyroscope2CLP              = "gyroscope-2clp"
	ExchangeGyroscope3CLP              = "gyroscope-3clp"
	ExchangeGyroscopeECLP              = "gyroscope-eclp"
	ExchangeHashflowV3                 = "hashflow-v3"
	ExchangeHoldFun                    = "hold-fun"
	ExchangeHoney                      = "honey"
	ExchangeHoriza                     = "horiza"
	ExchangeHorizonDex                 = "horizon-dex"
	ExchangeHorizonIntegral            = "horizon-integral"
	ExchangeHyeth                      = "hyeth"
	ExchangeHyperBlast                 = "hyper-blast"
	ExchangeIZiSwap                    = "iziswap"
	ExchangeInfusion                   = "infusion"
	ExchangeIntegral                   = "integral"
	ExchangeIronStable                 = "iron-stable"
	ExchangeJetSwap                    = "jetswap"
	ExchangeKTX                        = "ktx"
	ExchangeKatanaV2                   = "katana-v2"
	ExchangeKatanaV3                   = "katana-v3"
	ExchangeKellerFinance              = "keller-finance"
	ExchangeKelpRSETH                  = "kelp-rseth"
	ExchangeKinetixV2                  = "kinetix-v2"
	ExchangeKinetixV3                  = "kinetix-v3"
	ExchangeKodiakV2                   = "kodiak-v2"
	ExchangeKodiakV3                   = "kodiak-v3"
	ExchangeKoiCL                      = "koi-cl"
	ExchangeKokonutCpmm                = "kokonut-cpmm"
	ExchangeKokonutCrypto              = "kokonut-crypto"
	ExchangeKrptoDex                   = "kryptodex"
	ExchangeKyberPMM                   = "kyber-pmm"
	ExchangeKyberSwap                  = "kyberswap"
	ExchangeKyberSwapLimitOrder        = "kyberswap-limit-order"
	ExchangeKyberSwapLimitOrderDS      = "kyberswap-limit-order-v2"
	ExchangeKyberSwapStatic            = "kyberswap-static"
	ExchangeKyberswapElastic           = "kyberswap-elastic"
	ExchangeLO1inch                    = "lo1inch"
	ExchangeLineHubV2                  = "linehub-v2"
	ExchangeLineHubV3                  = "linehub-v3"
	ExchangeLiquidusFinance            = "liquidus-finance"
	ExchangeLitePSM                    = "lite-psm"
	ExchangeLizard                     = "lizard"
	ExchangeLydia                      = "lydia"
	ExchangeLynex                      = "lynex"
	ExchangeLynexV1                    = "lynex-v1"
	ExchangeLyve                       = "lyve"
	ExchangeMDex                       = "mdex"
	ExchangeMMF                        = "mmf"
	ExchangeMMFV3                      = "mmf-v3"
	ExchangeMVM                        = "mvm"
	ExchangeMadMex                     = "madmex"
	ExchangeMakerLido                  = "lido"
	ExchangeMakerLidoStETH             = "lido-steth"
	ExchangeMakerPSM                   = "maker-psm"
	ExchangeMakerSavingsDai            = "maker-savingsdai"
	ExchangeMantisSwap                 = "mantisswap"
	ExchangeMantleETH                  = "meth"
	ExchangeMaverickV1                 = "maverick-v1"
	ExchangeMaverickV2                 = "maverick-v2"
	ExchangeMemeBox                    = "memebox"
	ExchangeMemeswap                   = "memeswap"
	ExchangeMerchantMoe                = "merchant-moe"
	ExchangeMerchantMoeV22             = "merchant-moe-v22"
	ExchangeMetavault                  = "metavault"
	ExchangeMetavaultV2                = "metavault-v2"
	ExchangeMetavaultV3                = "metavault-v3"
	ExchangeMetropolis                 = "metropolis"
	ExchangeMetropolisLB               = "metropolis-lb"
	ExchangeMkrSky                     = "mkr-sky"
	ExchangeMonoswap                   = "monoswap"
	ExchangeMonoswapV3                 = "monoswap-v3"
	ExchangeMoonBase                   = "moonbase"
	ExchangeMorpheus                   = "morpheus"
	ExchangeMummyFinance               = "mummy-finance"
	ExchangeMuteSwitch                 = "muteswitch"
	ExchangeNativeV1                   = "native-v1"
	ExchangeNativeV2                   = "native-v2"
	ExchangeNavigator                  = "navigator"
	ExchangeNearPad                    = "nearpad"
	ExchangeNerve                      = "nerve"
	ExchangeNile                       = "nile"
	ExchangeNileV2                     = "nile-v2"
	ExchangeNomiswap                   = "nomiswap"
	ExchangeNomiswapStable             = "nomiswap-stable"
	ExchangeNuri                       = "nuri"
	ExchangeNuriV2                     = "nuri-v2"
	ExchangeOETH                       = "oeth"
	ExchangeOndoUSDY                   = "ondo-usdy"
	ExchangeOneSwap                    = "oneswap"
	ExchangeOpx                        = "opx"
	ExchangeOvernightUsdp              = "overnight-usdp"
	ExchangeOwlSwapV3                  = "owlswap-v3"
	ExchangePaintSwap                  = "paintswap"
	ExchangePancake                    = "pancake"
	ExchangePancakeInfinityBin         = "pancake-infinity-bin"
	ExchangePancakeInfinityBinFairflow = "pancake-infinity-bin-fairflow"
	ExchangePancakeInfinityCL          = "pancake-infinity-cl"
	ExchangePancakeInfinityCLFairflow  = "pancake-infinity-cl-fairflow"
	ExchangePancakeLegacy              = "pancake-legacy"
	ExchangePancakeStable              = "pancake-stable"
	ExchangePancakeV3                  = "pancake-v3"
	ExchangePandaFun                   = "panda-fun"
	ExchangePangolin                   = "pangolin"
	ExchangePantherSwap                = "pantherswap"
	ExchangePearl                      = "pearl"
	ExchangePearlV2                    = "pearl-v2"
	ExchangePharaoh                    = "pharaoh"
	ExchangePharaohV2                  = "pharaoh-v2"
	ExchangePhotonSwap                 = "photonswap"
	ExchangePlatypus                   = "platypus"
	ExchangePmm1                       = "pmm-1"
	ExchangePmm2                       = "pmm-2"
	ExchangePolMatic                   = "pol-matic"
	ExchangePolyDex                    = "polydex"
	ExchangePolycat                    = "polycat"
	ExchangePotatoSwap                 = "potato-swap"
	ExchangePrimeETH                   = "primeeth"
	ExchangePufferPufETH               = "puffer-pufeth"
	ExchangePunkSwap                   = "punkswap"
	ExchangeQuickPerps                 = "quickperps"
	ExchangeQuickSwap                  = "quickswap"
	ExchangeQuickSwapUniV3             = "quickswap-uni-v3"
	ExchangeQuickSwapV3                = "quickswap-v3"
	ExchangeRamses                     = "ramses"
	ExchangeRamsesV2                   = "ramses-v2"
	ExchangeRenzoEZETH                 = "renzo-ezeth"
	ExchangeRetro                      = "retro"
	ExchangeRetroV3                    = "retro-v3"
	ExchangeRevoSwap                   = "revo-swap"
	ExchangeRingSwap                   = "ringswap"
	ExchangeRocketPoolRETH             = "rocketpool-reth"
	ExchangeRocketSwapV2               = "rocketswap-v2"
	ExchangeRoguex                     = "roguex"
	ExchangeSaddle                     = "saddle"
	ExchangeSafeSwap                   = "safeswap"
	ExchangeSavingsUSDS                = "savings-usds"
	ExchangeSboom                      = "sboom"
	ExchangeScale                      = "scale"
	ExchangeScribe                     = "scribe"
	ExchangeScrollSwap                 = "scrollswap"
	ExchangeSectaV2                    = "secta-v2"
	ExchangeSectaV3                    = "secta-v3"
	ExchangeSfrxETH                    = "sfrxeth"
	ExchangeSfrxETHConvertor           = "sfrxeth-convertor"
	ExchangeShadowDex                  = "shadow-dex"
	ExchangeShadowLegacy               = "shadow-legacy"
	ExchangeShibaSwap                  = "shibaswap"
	ExchangeSilverSwap                 = "silverswap"
	ExchangeSkyPSM                     = "sky-psm"
	ExchangeSkydrome                   = "skydrome"
	ExchangeSkydromeV2                 = "skydrome-v2"
	ExchangeSmardex                    = "smardex"
	ExchangeSoSwap                     = "soswap"
	ExchangeSolidlyV2                  = "solidly-v2"
	ExchangeSolidlyV3                  = "solidly-v3"
	ExchangeSonicMarket                = "sonic-market"
	ExchangeSpacefi                    = "spacefi"
	ExchangeSpartaDex                  = "sparta-dex"
	ExchangeSpiritSwap                 = "spiritswap"
	ExchangeSpookySwap                 = "spookyswap"
	ExchangeSpookySwapV3               = "spookyswap-v3"
	ExchangeSquadSwap                  = "squadswap"
	ExchangeSquadSwapV2                = "squadswap-v2"
	ExchangeSquadSwapV3                = "squadswap-v3"
	ExchangeStaderETHx                 = "staderethx"
	ExchangeStationDexV2               = "station-dex-v2"
	ExchangeStationDexV3               = "station-dex-v3"
	ExchangeStratumFinance             = "stratum-finance"
	ExchangeSuperSwapV3                = "superswap-v3"
	ExchangeSushiSwap                  = "sushiswap"
	ExchangeSushiSwapV3                = "sushiswap-v3"
	ExchangeSwaapV2                    = "swaap-v2"
	ExchangeSwapBased                  = "swapbased"
	ExchangeSwapBasedPerp              = "swapbased-perp"
	ExchangeSwapBasedV3                = "swapbased-v3"
	ExchangeSwapBlast                  = "swap-blast"
	ExchangeSwapXCL                    = "swap-x-cl"
	ExchangeSwapXV2                    = "swap-x-v2"
	ExchangeSwapr                      = "swapr"
	ExchangeSwapsicle                  = "swapsicle"
	ExchangeSwellRSWETH                = "swell-rsweth"
	ExchangeSwellSWETH                 = "swell-sweth"
	ExchangeSynapse                    = "synapse"
	ExchangeSyncSwap                   = "syncswap"
	ExchangeSyncSwapCL                 = "syncswap-cl"
	ExchangeSyncSwapV2Aqua             = "syncswapv2-aqua"
	ExchangeSyncSwapV2Classic          = "syncswapv2-classic"
	ExchangeSyncSwapV2Stable           = "syncswapv2-stable"
	ExchangeSynthSwap                  = "synthswap"
	ExchangeSynthSwapPerp              = "synthswap-perp"
	ExchangeSynthSwapV3                = "synthswap-v3"
	ExchangeSynthetix                  = "synthetix"
	ExchangeThena                      = "thena"
	ExchangeThenaFusion                = "thena-fusion"
	ExchangeThenaFusionV3              = "thena-fusion-v3"
	ExchangeThick                      = "thick"
	ExchangeThrusterV2                 = "thruster-v2"
	ExchangeThrusterV2Degen            = "thruster-v2-degen"
	ExchangeThrusterV3                 = "thruster-v3"
	ExchangeTokan                      = "tokan-exchange"
	ExchangeTraderJoe                  = "traderjoe"
	ExchangeTraderJoeV20               = "traderjoe-v20"
	ExchangeTraderJoeV21               = "traderjoe-v21"
	ExchangeTraderJoeV22               = "traderjoe-v22"
	ExchangeTrisolaris                 = "trisolaris"
	ExchangeTsunamiX                   = "tsunami-x"
	ExchangeUSDFi                      = "usdfi"
	ExchangeUcsFinance                 = "ucs-finance"
	ExchangeUnchainX                   = "unchainx"
	ExchangeUniSwap                    = "uniswap"
	ExchangeUniSwapV1                  = "uniswap-v1"
	ExchangeUniSwapV2                  = "uniswap-v2"
	ExchangeUniSwapV3                  = "uniswapv3"
	ExchangeUniswapV4                  = "uniswap-v4"
	ExchangeUniswapV4BunniV2           = "uniswap-v4-bunni-v2"
	ExchangeUniswapV4FairFlow          = "uniswap-v4-fairflow"
	ExchangeUniswapV4Kem               = "uniswap-v4-kem"
	ExchangeUsd0PP                     = "usd0pp"
	ExchangeVVS                        = "vvs"
	ExchangeValleySwap                 = "valleyswap"
	ExchangeValleySwapV2               = "valleyswap-v2"
	ExchangeVelocore                   = "velocore"
	ExchangeVelocoreV2CPMM             = "velocore-v2-cpmm"
	ExchangeVelocoreV2WombatStable     = "velocore-v2-wombat-stable"
	ExchangeVelodrome                  = "velodrome"
	ExchangeVelodromeCL                = "velodrome-cl"
	ExchangeVelodromeCL2               = "velodrome-cl-2"
	ExchangeVelodromeV2                = "velodrome-v2"
	ExchangeVerse                      = "verse"
	ExchangeVesync                     = "vesync"
	ExchangeVirtualFun                 = "virtual-fun"
	ExchangeVodoo                      = "vodoo"
	ExchangeVooi                       = "vooi"
	ExchangeWBETH                      = "wbeth"
	ExchangeWagmi                      = "wagmi"
	ExchangeWagyuSwap                  = "wagyuswap"
	ExchangeWannaSwap                  = "wannaswap"
	ExchangeWasabi                     = "wasabi"
	ExchangeWault                      = "wault"
	ExchangeWigoSwap                   = "wigo-swap"
	ExchangeWombat                     = "wombat"
	ExchangeWooFiV2                    = "woofi-v2"
	ExchangeWooFiV3                    = "woofi-v3"
	ExchangeXLayerSwap                 = "xlayer-swap"
	ExchangeYetiSwap                   = "yetiswap"
	ExchangeYuzuSwap                   = "yuzuswap"
	ExchangeZKSwap                     = "zkswap"
	ExchangeZebra                      = "zebra"
	ExchangeZebraV2                    = "zebra-v2"
	ExchangeZero                       = "zero"
	ExchangeZipSwap                    = "zipswap"
	ExchangeZkEraFinance               = "zkera-finance"
	ExchangeZkSwapFinance              = "zkswap-finance"
	ExchangeZkSwapStable               = "zkswap-stable"
	ExchangeZkSwapV3                   = "zkswap-v3"
	ExchangeZyberSwapV3                = "zyberswap-v3"
)

var AMMSourceSet = map[Exchange]struct{}{
	Exchange9mmProV2:                   {},
	Exchange9mmProV3:                   {},
	ExchangeAbstra:                     {},
	ExchangeAerodrome:                  {},
	ExchangeAerodromeCL:                {},
	ExchangeAgniFinance:                {},
	ExchangeAlienBase:                  {},
	ExchangeAlienBaseCL:                {},
	ExchangeAlienBaseDegen:             {},
	ExchangeAlienBaseStableSwap:        {},
	ExchangeAmbient:                    {},
	ExchangeAmped:                      {},
	ExchangeApeSwap:                    {},
	ExchangeArbiDex:                    {},
	ExchangeArbiDexV3:                  {},
	ExchangeArbswapAMM:                 {},
	ExchangeArenaDex:                   {},
	ExchangeArenaDexV2:                 {},
	ExchangeAstroSwap:                  {},
	ExchangeAxial:                      {},
	ExchangeBMX:                        {},
	ExchangeBMXGLP:                     {},
	ExchangeBabyDogeSwap:               {},
	ExchangeBabySwap:                   {},
	ExchangeBakerySwap:                 {},
	ExchangeBalDex:                     {},
	ExchangeBalancer:                   {},
	ExchangeBalancerV1:                 {},
	ExchangeBalancerV2ComposableStable: {},
	ExchangeBalancerV2Stable:           {},
	ExchangeBalancerV2Weighted:         {},
	ExchangeBalancerV3ECLP:             {},
	ExchangeBalancerV3Stable:           {},
	ExchangeBalancerV3Weighted:         {},
	ExchangeBancorV21:                  {},
	ExchangeBancorV3:                   {},
	ExchangeBaseSwap:                   {},
	ExchangeBaseSwapV3:                 {},
	ExchangeBaso:                       {},
	ExchangeBedrockUniETH:              {},
	ExchangeBeefySonic:                 {},
	ExchangeBeethovenX:                 {},
	ExchangeBeethovenXComposableStable: {},
	ExchangeBeethovenXStable:           {},
	ExchangeBeethovenXV3Stable:         {},
	ExchangeBeethovenXV3Weighted:       {},
	ExchangeBeethovenXWeighted:         {},
	ExchangeBeetsSS:                    {},
	ExchangeBeraSwapComposableStable:   {},
	ExchangeBeraSwapStable:             {},
	ExchangeBeraSwapWeighted:           {},
	ExchangeBeracaine:                  {},
	ExchangeBiSwap:                     {},
	ExchangeBlade:                      {},
	ExchangeBladeSwap:                  {},
	ExchangeBlastDex:                   {},
	ExchangeBlasterSwap:                {},
	ExchangeBlueprint:                  {},
	ExchangeBulla:                      {},
	ExchangeBurrBearComposableStable:   {},
	ExchangeBurrBearStable:             {},
	ExchangeBurrBearWeighted:           {},
	ExchangeButterFi:                   {},
	ExchangeBvm:                        {},
	ExchangeCamelot:                    {},
	ExchangeCamelotV3:                  {},
	ExchangeChronos:                    {},
	ExchangeChronosV3:                  {},
	ExchangeCleopatra:                  {},
	ExchangeCleopatraV2:                {},
	ExchangeCometh:                     {},
	ExchangeCrodex:                     {},
	ExchangeCronaSwap:                  {},
	ExchangeCrowdswapV2:                {},
	ExchangeCurve:                      {},
	ExchangeCurveLending:               {},
	ExchangeCurveLlamma:                {},
	ExchangeCurveStableMetaNg:          {},
	ExchangeCurveStableNg:              {},
	ExchangeCurveStablePlain:           {},
	ExchangeCurveTriCryptoNg:           {},
	ExchangeCurveTwoCryptoNg:           {},
	ExchangeCyberblastV3:               {},
	ExchangeDFYN:                       {},
	ExchangeDMM:                        {},
	ExchangeDackieV2:                   {},
	ExchangeDackieV3:                   {},
	ExchangeDaiUsds:                    {},
	ExchangeDeFive:                     {},
	ExchangeDefiSwap:                   {},
	ExchangeDegenExpress:               {},
	ExchangeDeltaSwapV1:                {},
	ExchangeDinoSwap:                   {},
	ExchangeDinosaurEggs:               {},
	ExchangeDodo:                       {},
	ExchangeDodoClassical:              {},
	ExchangeDodoPrivatePool:            {},
	ExchangeDodoStablePool:             {},
	ExchangeDodoVendingMachine:         {},
	ExchangeDoveSwapV3:                 {},
	ExchangeDyorSwap:                   {},
	ExchangeDystopia:                   {},
	ExchangeE3:                         {},
	ExchangeEchoDex:                    {},
	ExchangeEchoDexV3:                  {},
	ExchangeEkubo:                      {},
	ExchangeEllipsis:                   {},
	ExchangeEmpireDex:                  {},
	ExchangeEqual:                      {},
	ExchangeEqualizerCL:                {},
	ExchangeEthenaSusde:                {},
	ExchangeEtherFieBTC:                {},
	ExchangeEtherVista:                 {},
	ExchangeEtherfiEETH:                {},
	ExchangeEtherfiVampire:             {},
	ExchangeEtherfiWEETH:               {},
	ExchangeEulerSwap:                  {},
	ExchangeEzkalibur:                  {},
	ExchangeFenix:                      {},
	ExchangeFluidDexT1:                 {},
	ExchangeFluidVaultT1:               {},
	ExchangeFraxSwap:                   {},
	ExchangeFrxETH:                     {},
	ExchangeFstsSwap:                   {},
	ExchangeFulcrom:                    {},
	ExchangeFusionX:                    {},
	ExchangeFusionXV3:                  {},
	ExchangeFvm:                        {},
	ExchangeFxdx:                       {},
	ExchangeGMX:                        {},
	ExchangeGemKeeper:                  {},
	ExchangeGravity:                    {},
	ExchangeGyroscope2CLP:              {},
	ExchangeGyroscope3CLP:              {},
	ExchangeGyroscopeECLP:              {},
	ExchangeHoldFun:                    {},
	ExchangeHoney:                      {},
	ExchangeHoriza:                     {},
	ExchangeHorizonDex:                 {},
	ExchangeHorizonIntegral:            {},
	ExchangeHyeth:                      {},
	ExchangeHyperBlast:                 {},
	ExchangeIZiSwap:                    {},
	ExchangeInfusion:                   {},
	ExchangeIntegral:                   {},
	ExchangeIronStable:                 {},
	ExchangeJetSwap:                    {},
	ExchangeKTX:                        {},
	ExchangeKatanaV2:                   {},
	ExchangeKatanaV3:                   {},
	ExchangeKellerFinance:              {},
	ExchangeKelpRSETH:                  {},
	ExchangeKinetixV2:                  {},
	ExchangeKinetixV3:                  {},
	ExchangeKodiakV2:                   {},
	ExchangeKodiakV3:                   {},
	ExchangeKoiCL:                      {},
	ExchangeKokonutCpmm:                {},
	ExchangeKokonutCrypto:              {},
	ExchangeKrptoDex:                   {},
	ExchangeKyberSwap:                  {},
	ExchangeKyberSwapLimitOrder:        {},
	ExchangeKyberSwapLimitOrderDS:      {},
	ExchangeKyberSwapStatic:            {},
	ExchangeKyberswapElastic:           {},
	ExchangeLineHubV2:                  {},
	ExchangeLineHubV3:                  {},
	ExchangeLiquidusFinance:            {},
	ExchangeLitePSM:                    {},
	ExchangeLizard:                     {},
	ExchangeLydia:                      {},
	ExchangeLynex:                      {},
	ExchangeLynexV1:                    {},
	ExchangeLyve:                       {},
	ExchangeMDex:                       {},
	ExchangeMMF:                        {},
	ExchangeMMFV3:                      {},
	ExchangeMVM:                        {},
	ExchangeMadMex:                     {},
	ExchangeMakerLido:                  {},
	ExchangeMakerLidoStETH:             {},
	ExchangeMakerPSM:                   {},
	ExchangeMakerSavingsDai:            {},
	ExchangeMantisSwap:                 {},
	ExchangeMantleETH:                  {},
	ExchangeMaverickV1:                 {},
	ExchangeMaverickV2:                 {},
	ExchangeMemeBox:                    {},
	ExchangeMemeswap:                   {},
	ExchangeMerchantMoe:                {},
	ExchangeMerchantMoeV22:             {},
	ExchangeMetavault:                  {},
	ExchangeMetavaultV2:                {},
	ExchangeMetavaultV3:                {},
	ExchangeMetropolis:                 {},
	ExchangeMetropolisLB:               {},
	ExchangeMkrSky:                     {},
	ExchangeMonoswap:                   {},
	ExchangeMonoswapV3:                 {},
	ExchangeMoonBase:                   {},
	ExchangeMorpheus:                   {},
	ExchangeMummyFinance:               {},
	ExchangeMuteSwitch:                 {},
	ExchangeNavigator:                  {},
	ExchangeNearPad:                    {},
	ExchangeNerve:                      {},
	ExchangeNile:                       {},
	ExchangeNileV2:                     {},
	ExchangeNomiswap:                   {},
	ExchangeNomiswapStable:             {},
	ExchangeNuri:                       {},
	ExchangeNuriV2:                     {},
	ExchangeOETH:                       {},
	ExchangeOndoUSDY:                   {},
	ExchangeOneSwap:                    {},
	ExchangeOpx:                        {},
	ExchangeOvernightUsdp:              {},
	ExchangeOwlSwapV3:                  {},
	ExchangePaintSwap:                  {},
	ExchangePancake:                    {},
	ExchangePancakeInfinityBin:         {},
	ExchangePancakeInfinityBinFairflow: {},
	ExchangePancakeInfinityCL:          {},
	ExchangePancakeInfinityCLFairflow:  {},
	ExchangePancakeLegacy:              {},
	ExchangePancakeStable:              {},
	ExchangePancakeV3:                  {},
	ExchangePandaFun:                   {},
	ExchangePangolin:                   {},
	ExchangePantherSwap:                {},
	ExchangePearl:                      {},
	ExchangePearlV2:                    {},
	ExchangePharaoh:                    {},
	ExchangePharaohV2:                  {},
	ExchangePhotonSwap:                 {},
	ExchangePlatypus:                   {},
	ExchangePolMatic:                   {},
	ExchangePolyDex:                    {},
	ExchangePolycat:                    {},
	ExchangePotatoSwap:                 {},
	ExchangePrimeETH:                   {},
	ExchangePufferPufETH:               {},
	ExchangePunkSwap:                   {},
	ExchangeQuickPerps:                 {},
	ExchangeQuickSwap:                  {},
	ExchangeQuickSwapUniV3:             {},
	ExchangeQuickSwapV3:                {},
	ExchangeRamses:                     {},
	ExchangeRamsesV2:                   {},
	ExchangeRenzoEZETH:                 {},
	ExchangeRetro:                      {},
	ExchangeRetroV3:                    {},
	ExchangeRevoSwap:                   {},
	ExchangeRingSwap:                   {},
	ExchangeRocketPoolRETH:             {},
	ExchangeRocketSwapV2:               {},
	ExchangeRoguex:                     {},
	ExchangeSaddle:                     {},
	ExchangeSafeSwap:                   {},
	ExchangeSavingsUSDS:                {},
	ExchangeSboom:                      {},
	ExchangeScale:                      {},
	ExchangeScribe:                     {},
	ExchangeScrollSwap:                 {},
	ExchangeSectaV2:                    {},
	ExchangeSectaV3:                    {},
	ExchangeSfrxETH:                    {},
	ExchangeSfrxETHConvertor:           {},
	ExchangeShadowDex:                  {},
	ExchangeShadowLegacy:               {},
	ExchangeShibaSwap:                  {},
	ExchangeSilverSwap:                 {},
	ExchangeSkyPSM:                     {},
	ExchangeSkydrome:                   {},
	ExchangeSkydromeV2:                 {},
	ExchangeSmardex:                    {},
	ExchangeSoSwap:                     {},
	ExchangeSolidlyV2:                  {},
	ExchangeSolidlyV3:                  {},
	ExchangeSonicMarket:                {},
	ExchangeSpacefi:                    {},
	ExchangeSpartaDex:                  {},
	ExchangeSpiritSwap:                 {},
	ExchangeSpookySwap:                 {},
	ExchangeSpookySwapV3:               {},
	ExchangeSquadSwap:                  {},
	ExchangeSquadSwapV2:                {},
	ExchangeSquadSwapV3:                {},
	ExchangeStaderETHx:                 {},
	ExchangeStationDexV2:               {},
	ExchangeStationDexV3:               {},
	ExchangeStratumFinance:             {},
	ExchangeSuperSwapV3:                {},
	ExchangeSushiSwap:                  {},
	ExchangeSushiSwapV3:                {},
	ExchangeSwapBased:                  {},
	ExchangeSwapBasedPerp:              {},
	ExchangeSwapBasedV3:                {},
	ExchangeSwapBlast:                  {},
	ExchangeSwapXCL:                    {},
	ExchangeSwapXV2:                    {},
	ExchangeSwapr:                      {},
	ExchangeSwapsicle:                  {},
	ExchangeSwellRSWETH:                {},
	ExchangeSwellSWETH:                 {},
	ExchangeSynapse:                    {},
	ExchangeSyncSwap:                   {},
	ExchangeSyncSwapCL:                 {},
	ExchangeSyncSwapV2Aqua:             {},
	ExchangeSyncSwapV2Classic:          {},
	ExchangeSyncSwapV2Stable:           {},
	ExchangeSynthSwap:                  {},
	ExchangeSynthSwapPerp:              {},
	ExchangeSynthSwapV3:                {},
	ExchangeSynthetix:                  {},
	ExchangeThena:                      {},
	ExchangeThenaFusion:                {},
	ExchangeThenaFusionV3:              {},
	ExchangeThick:                      {},
	ExchangeThrusterV2:                 {},
	ExchangeThrusterV2Degen:            {},
	ExchangeThrusterV3:                 {},
	ExchangeTokan:                      {},
	ExchangeTraderJoe:                  {},
	ExchangeTraderJoeV20:               {},
	ExchangeTraderJoeV21:               {},
	ExchangeTraderJoeV22:               {},
	ExchangeTrisolaris:                 {},
	ExchangeTsunamiX:                   {},
	ExchangeUSDFi:                      {},
	ExchangeUcsFinance:                 {},
	ExchangeUnchainX:                   {},
	ExchangeUniSwap:                    {},
	ExchangeUniSwapV1:                  {},
	ExchangeUniSwapV2:                  {},
	ExchangeUniSwapV3:                  {},
	ExchangeUniswapV4:                  {},
	ExchangeUniswapV4BunniV2:           {},
	ExchangeUniswapV4FairFlow:          {},
	ExchangeUniswapV4Kem:               {},
	ExchangeUsd0PP:                     {},
	ExchangeVVS:                        {},
	ExchangeValleySwap:                 {},
	ExchangeValleySwapV2:               {},
	ExchangeVelocore:                   {},
	ExchangeVelocoreV2CPMM:             {},
	ExchangeVelocoreV2WombatStable:     {},
	ExchangeVelodrome:                  {},
	ExchangeVelodromeCL2:               {},
	ExchangeVelodromeCL:                {},
	ExchangeVelodromeV2:                {},
	ExchangeVerse:                      {},
	ExchangeVesync:                     {},
	ExchangeVirtualFun:                 {},
	ExchangeVodoo:                      {},
	ExchangeVooi:                       {},
	ExchangeWBETH:                      {},
	ExchangeWagmi:                      {},
	ExchangeWagyuSwap:                  {},
	ExchangeWannaSwap:                  {},
	ExchangeWasabi:                     {},
	ExchangeWault:                      {},
	ExchangeWigoSwap:                   {},
	ExchangeWombat:                     {},
	ExchangeWooFiV2:                    {},
	ExchangeWooFiV3:                    {},
	ExchangeXLayerSwap:                 {},
	ExchangeYetiSwap:                   {},
	ExchangeYuzuSwap:                   {},
	ExchangeZKSwap:                     {},
	ExchangeZebra:                      {},
	ExchangeZebraV2:                    {},
	ExchangeZero:                       {},
	ExchangeZipSwap:                    {},
	ExchangeZkEraFinance:               {},
	ExchangeZkSwapFinance:              {},
	ExchangeZkSwapStable:               {},
	ExchangeZkSwapV3:                   {},
	ExchangeZyberSwapV3:                {},
}

func IsAMMSource(exchange Exchange) bool {
	_, ok := AMMSourceSet[exchange]
	return ok
}

var RFQSourceSet = map[Exchange]struct{}{
	ExchangeKyberSwapLimitOrderDS: {},
	ExchangeLO1inch:               {},

	ExchangeBebop:      {},
	ExchangeClipper:    {},
	ExchangeDexalot:    {},
	ExchangeHashflowV3: {},
	ExchangeKyberPMM:   {},
	ExchangeNativeV1:   {},
	ExchangeNativeV2:   {},
	ExchangePmm1:       {},
	ExchangePmm2:       {},
	ExchangeSwaapV2:    {},
}

func IsRFQSource(exchange Exchange) bool {
	_, ok := RFQSourceSet[exchange]
	return ok
}

// SingleSwapSourceSet is a set of exchanges that
// only allow a single swap in a route.
var SingleSwapSourceSet = map[Exchange]struct{}{
	ExchangeBebop:         {},
	ExchangeClipper:       {},
	ExchangeOvernightUsdp: {},
}

func IsSingleSwapSource(exchange Exchange) bool {
	_, ok := SingleSwapSourceSet[exchange]
	return ok
}

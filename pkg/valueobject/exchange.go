package valueobject

type Exchange string

const (
	Exchange9mmProV2                   Exchange = "9mm-pro-v2"
	Exchange9mmProV3                   Exchange = "9mm-pro-v3"
	ExchangeAbstra                     Exchange = "abstra"
	ExchangeAerodrome                  Exchange = "aerodrome"
	ExchangeAerodromeCL                Exchange = "aerodrome-cl"
	ExchangeAgniFinance                Exchange = "agni-finance"
	ExchangeAlienBase                  Exchange = "alien-base"
	ExchangeAlienBaseCL                Exchange = "alien-base-cl"
	ExchangeAlienBaseDegen             Exchange = "alien-base-degen"
	ExchangeAlienBaseStableSwap        Exchange = "alien-base-stableswap"
	ExchangeAmbient                    Exchange = "ambient"
	ExchangeAmped                      Exchange = "amped"
	ExchangeApeSwap                    Exchange = "apeswap"
	ExchangeArbiDex                    Exchange = "arbi-dex"
	ExchangeArbiDexV3                  Exchange = "arbidex-v3"
	ExchangeArbswapAMM                 Exchange = "arbswap-amm"
	ExchangeArenaDex                   Exchange = "arenadex"
	ExchangeArenaDexV2                 Exchange = "arenadex-v2"
	ExchangeAstroSwap                  Exchange = "astroswap"
	ExchangeAxial                      Exchange = "axial"
	ExchangeBabySwap                   Exchange = "babyswap"
	ExchangeBakerySwap                 Exchange = "bakeryswap"
	ExchangeBMX                        Exchange = "bmx"
	ExchangeBMXGLP                     Exchange = "bmx-glp"
	ExchangeBabyDogeSwap               Exchange = "babydogeswap"
	ExchangeBalDex                     Exchange = "baldex"
	ExchangeBalancer                   Exchange = "balancer"
	ExchangeBalancerV1                 Exchange = "balancer-v1"
	ExchangeBalancerV2ComposableStable Exchange = "balancer-v2-composable-stable"
	ExchangeBalancerV2Stable           Exchange = "balancer-v2-stable"
	ExchangeBalancerV2Weighted         Exchange = "balancer-v2-weighted"
	ExchangeBalancerV3ECLP             Exchange = "balancer-v3-eclp"
	ExchangeBalancerV3Stable           Exchange = "balancer-v3-stable"
	ExchangeBalancerV3Weighted         Exchange = "balancer-v3-weighted"
	ExchangeBancorV21                  Exchange = "bancor-v21"
	ExchangeBancorV3                   Exchange = "bancor-v3"
	ExchangeBaseSwap                   Exchange = "baseswap"
	ExchangeBaseSwapV3                 Exchange = "baseswap-v3"
	ExchangeBaso                       Exchange = "baso"
	ExchangeBebop                      Exchange = "bebop"
	ExchangeBedrockUniETH              Exchange = "bedrock-unieth"
	ExchangeBeefySonic                 Exchange = "beefy-sonic"
	ExchangeBeethovenX                 Exchange = "beethovenx"
	ExchangeBeethovenXComposableStable Exchange = "beethovenx-composable-stable"
	ExchangeBeethovenXStable           Exchange = "beethovenx-stable"
	ExchangeBeethovenXV3Stable         Exchange = "beethovenx-v3-stable"
	ExchangeBeethovenXV3Weighted       Exchange = "beethovenx-v3-weighted"
	ExchangeBeethovenXWeighted         Exchange = "beethovenx-weighted"
	ExchangeBeetsSS                    Exchange = "beets-ss"
	ExchangeBeraSwapComposableStable   Exchange = "beraswap-composable-stable"
	ExchangeBeraSwapStable             Exchange = "beraswap-stable"
	ExchangeBeraSwapWeighted           Exchange = "beraswap-weighted"
	ExchangeBeracaine                  Exchange = "beracaine"
	ExchangeBiSwap                     Exchange = "biswap"
	ExchangeBlade                      Exchange = "blade"
	ExchangeBladeSwap                  Exchange = "blade-swap"
	ExchangeBlastDex                   Exchange = "blastdex"
	ExchangeBlasterSwap                Exchange = "blasterswap"
	ExchangeBlueprint                  Exchange = "blueprint"
	ExchangeBulla                      Exchange = "bulla"
	ExchangeBurrBearComposableStable   Exchange = "burrbear-composable-stable"
	ExchangeBurrBearStable             Exchange = "burrbear-stable"
	ExchangeBurrBearWeighted           Exchange = "burrbear-weighted"
	ExchangeButterFi                   Exchange = "butter-fi"
	ExchangeBvm                        Exchange = "bvm"
	ExchangeCamelot                    Exchange = "camelot"
	ExchangeCamelotV3                  Exchange = "camelot-v3"
	ExchangeChronos                    Exchange = "chronos"
	ExchangeChronosV3                  Exchange = "chronos-v3"
	ExchangeCleopatra                  Exchange = "cleopatra"
	ExchangeCleopatraV2                Exchange = "cleopatra-v2"
	ExchangeClipper                    Exchange = "clipper"
	ExchangeCometh                     Exchange = "cometh"
	ExchangeCrodex                     Exchange = "crodex"
	ExchangeCronaSwap                  Exchange = "cronaswap"
	ExchangeCrowdswapV2                Exchange = "crowdswap-v2"
	ExchangeCurve                      Exchange = "curve"
	ExchangeCurveLending               Exchange = "curve-lending"
	ExchangeCurveLlamma                Exchange = "curve-llamma"
	ExchangeCurveStableMetaNg          Exchange = "curve-stable-meta-ng"
	ExchangeCurveStableNg              Exchange = "curve-stable-ng"
	ExchangeCurveStablePlain           Exchange = "curve-stable-plain"
	ExchangeCurveTriCryptoNg           Exchange = "curve-tricrypto-ng"
	ExchangeCurveTwoCryptoNg           Exchange = "curve-twocrypto-ng"
	ExchangeCyberblastV3               Exchange = "cyberblast-v3"
	ExchangeDFYN                       Exchange = "dfyn"
	ExchangeDMM                        Exchange = "dmm"
	ExchangeDackieV2                   Exchange = "dackie-v2"
	ExchangeDackieV3                   Exchange = "dackie-v3"
	ExchangeDaiUsds                    Exchange = "dai-usds"
	ExchangeDeFive                     Exchange = "defive"
	ExchangeDefiSwap                   Exchange = "defiswap"
	ExchangeDegenExpress               Exchange = "degen-express"
	ExchangeDeltaSwapV1                Exchange = "deltaswap-v1"
	ExchangeDexalot                    Exchange = "dexalot"
	ExchangeDinoSwap                   Exchange = "dinoswap"
	ExchangeDinosaurEggs               Exchange = "dinosaureggs"
	ExchangeDodo                       Exchange = "dodo"
	ExchangeDodoClassical              Exchange = "dodo-classical"
	ExchangeDodoPrivatePool            Exchange = "dodo-dpp"
	ExchangeDodoStablePool             Exchange = "dodo-dsp"
	ExchangeDodoVendingMachine         Exchange = "dodo-dvm"
	ExchangeDoveSwapV3                 Exchange = "doveswap-v3"
	ExchangeDyorSwap                   Exchange = "dyor-swap"
	ExchangeDystopia                   Exchange = "dystopia"
	ExchangeE3                         Exchange = "e3"
	ExchangeEchoDex                    Exchange = "echo-dex"
	ExchangeEchoDexV3                  Exchange = "echo-dex-v3"
	ExchangeEkubo                      Exchange = "ekubo"
	ExchangeEllipsis                   Exchange = "ellipsis"
	ExchangeEmpireDex                  Exchange = "empiredex"
	ExchangeEqual                      Exchange = "equal"
	ExchangeEqualizerCL                Exchange = "equalizer-cl"
	ExchangeEthenaSusde                Exchange = "ethena-susde"
	ExchangeEtherFieBTC                Exchange = "etherfi-ebtc"
	ExchangeEtherVista                 Exchange = "ether-vista"
	ExchangeEtherfiEETH                Exchange = "etherfi-eeth"
	ExchangeEtherfiVampire             Exchange = "etherfi-vampire"
	ExchangeEtherfiWEETH               Exchange = "etherfi-weeth"
	ExchangeEulerSwap                  Exchange = "euler-swap"
	ExchangeEzkalibur                  Exchange = "ezkalibur"
	ExchangeFakePool                   Exchange = "fake-pool"
	ExchangeFenix                      Exchange = "fenix"
	ExchangeFluidDexT1                 Exchange = "fluid-dex-t1"
	ExchangeFluidVaultT1               Exchange = "fluid-vault-t1"
	ExchangeFraxSwap                   Exchange = "fraxswap"
	ExchangeFrxETH                     Exchange = "frxeth"
	ExchangeFstsSwap                   Exchange = "fstsswap"
	ExchangeFulcrom                    Exchange = "fulcrom"
	ExchangeFusionX                    Exchange = "fusion-x"
	ExchangeFusionXV3                  Exchange = "fusion-x-v3"
	ExchangeFvm                        Exchange = "fvm"
	ExchangeFxdx                       Exchange = "fxdx"
	ExchangeGMX                        Exchange = "gmx"
	ExchangeGemKeeper                  Exchange = "gemkeeper"
	ExchangeGravity                    Exchange = "gravity"
	ExchangeGyroscope2CLP              Exchange = "gyroscope-2clp"
	ExchangeGyroscope3CLP              Exchange = "gyroscope-3clp"
	ExchangeGyroscopeECLP              Exchange = "gyroscope-eclp"
	ExchangeHashflowV3                 Exchange = "hashflow-v3"
	ExchangeHoldFun                    Exchange = "hold-fun"
	ExchangeHoney                      Exchange = "honey"
	ExchangeHoriza                     Exchange = "horiza"
	ExchangeHorizonDex                 Exchange = "horizon-dex"
	ExchangeHorizonIntegral            Exchange = "horizon-integral"
	ExchangeHyeth                      Exchange = "hyeth"
	ExchangeHyperBlast                 Exchange = "hyper-blast"
	ExchangeIZiSwap                    Exchange = "iziswap"
	ExchangeInfusion                   Exchange = "infusion"
	ExchangeIntegral                   Exchange = "integral"
	ExchangeIronStable                 Exchange = "iron-stable"
	ExchangeJetSwap                    Exchange = "jetswap"
	ExchangeKTX                        Exchange = "ktx"
	ExchangeKatanaV2                   Exchange = "katana-v2"
	ExchangeKatanaV3                   Exchange = "katana-v3"
	ExchangeKellerFinance              Exchange = "keller-finance"
	ExchangeKelpRSETH                  Exchange = "kelp-rseth"
	ExchangeKinetixV2                  Exchange = "kinetix-v2"
	ExchangeKinetixV3                  Exchange = "kinetix-v3"
	ExchangeKodiakV2                   Exchange = "kodiak-v2"
	ExchangeKodiakV3                   Exchange = "kodiak-v3"
	ExchangeKoiCL                      Exchange = "koi-cl"
	ExchangeKokonutCpmm                Exchange = "kokonut-cpmm"
	ExchangeKokonutCrypto              Exchange = "kokonut-crypto"
	ExchangeKrptoDex                   Exchange = "kryptodex"
	ExchangeKyberPMM                   Exchange = "kyber-pmm"
	ExchangeKyberSwap                  Exchange = "kyberswap"
	ExchangeKyberSwapLimitOrder        Exchange = "kyberswap-limit-order"
	ExchangeKyberSwapLimitOrderDS      Exchange = "kyberswap-limit-order-v2"
	ExchangeKyberSwapStatic            Exchange = "kyberswap-static"
	ExchangeKyberswapElastic           Exchange = "kyberswap-elastic"
	ExchangeLO1inch                    Exchange = "lo1inch"
	ExchangeLineHubV2                  Exchange = "linehub-v2"
	ExchangeLineHubV3                  Exchange = "linehub-v3"
	ExchangeLiquidusFinance            Exchange = "liquidus-finance"
	ExchangeLitePSM                    Exchange = "lite-psm"
	ExchangeLizard                     Exchange = "lizard"
	ExchangeLydia                      Exchange = "lydia"
	ExchangeLynex                      Exchange = "lynex"
	ExchangeLynexV1                    Exchange = "lynex-v1"
	ExchangeLyve                       Exchange = "lyve"
	ExchangeMDex                       Exchange = "mdex"
	ExchangeMMF                        Exchange = "mmf"
	ExchangeMMFV3                      Exchange = "mmf-v3"
	ExchangeMVM                        Exchange = "mvm"
	ExchangeMadMex                     Exchange = "madmex"
	ExchangeMakerLido                  Exchange = "lido"
	ExchangeMakerLidoStETH             Exchange = "lido-steth"
	ExchangeMakerPSM                   Exchange = "maker-psm"
	ExchangeMakerSavingsDai            Exchange = "maker-savingsdai"
	ExchangeMantisSwap                 Exchange = "mantisswap"
	ExchangeMantleETH                  Exchange = "meth"
	ExchangeMaverickV1                 Exchange = "maverick-v1"
	ExchangeMaverickV2                 Exchange = "maverick-v2"
	ExchangeMemeBox                    Exchange = "memebox"
	ExchangeMemeswap                   Exchange = "memeswap"
	ExchangeMerchantMoe                Exchange = "merchant-moe"
	ExchangeMerchantMoeV22             Exchange = "merchant-moe-v22"
	ExchangeMetavault                  Exchange = "metavault"
	ExchangeMetavaultV2                Exchange = "metavault-v2"
	ExchangeMetavaultV3                Exchange = "metavault-v3"
	ExchangeMetropolis                 Exchange = "metropolis"
	ExchangeMetropolisLB               Exchange = "metropolis-lb"
	ExchangeMkrSky                     Exchange = "mkr-sky"
	ExchangeMonoswap                   Exchange = "monoswap"
	ExchangeMonoswapV3                 Exchange = "monoswap-v3"
	ExchangeMoonBase                   Exchange = "moonbase"
	ExchangeMorpheus                   Exchange = "morpheus"
	ExchangeMummyFinance               Exchange = "mummy-finance"
	ExchangeMuteSwitch                 Exchange = "muteswitch"
	ExchangeNativeV1                   Exchange = "native-v1"
	ExchangeNativeV2                   Exchange = "native-v2"
	ExchangeNavigator                  Exchange = "navigator"
	ExchangeNearPad                    Exchange = "nearpad"
	ExchangeNerve                      Exchange = "nerve"
	ExchangeNile                       Exchange = "nile"
	ExchangeNileV2                     Exchange = "nile-v2"
	ExchangeNomiswap                   Exchange = "nomiswap"
	ExchangeNomiswapStable             Exchange = "nomiswap-stable"
	ExchangeNuri                       Exchange = "nuri"
	ExchangeNuriV2                     Exchange = "nuri-v2"
	ExchangeOETH                       Exchange = "oeth"
	ExchangeOndoUSDY                   Exchange = "ondo-usdy"
	ExchangeOneSwap                    Exchange = "oneswap"
	ExchangeOpx                        Exchange = "opx"
	ExchangeOvernightUsdp              Exchange = "overnight-usdp"
	ExchangeOwlSwapV3                  Exchange = "owlswap-v3"
	ExchangePaintSwap                  Exchange = "paintswap"
	ExchangePancake                    Exchange = "pancake"
	ExchangePancakeInfinityBin         Exchange = "pancake-infinity-bin"
	ExchangePancakeInfinityCL          Exchange = "pancake-infinity-cl"
	ExchangePancakeLegacy              Exchange = "pancake-legacy"
	ExchangePancakeStable              Exchange = "pancake-stable"
	ExchangePancakeV3                  Exchange = "pancake-v3"
	ExchangePandaFun                   Exchange = "panda-fun"
	ExchangePangolin                   Exchange = "pangolin"
	ExchangePantherSwap                Exchange = "pantherswap"
	ExchangePearl                      Exchange = "pearl"
	ExchangePearlV2                    Exchange = "pearl-v2"
	ExchangePharaoh                    Exchange = "pharaoh"
	ExchangePharaohV2                  Exchange = "pharaoh-v2"
	ExchangePhotonSwap                 Exchange = "photonswap"
	ExchangePlatypus                   Exchange = "platypus"
	ExchangePmm1                       Exchange = "pmm-1"
	ExchangePmm2                       Exchange = "pmm-2"
	ExchangePolMatic                   Exchange = "pol-matic"
	ExchangePolyDex                    Exchange = "polydex"
	ExchangePolycat                    Exchange = "polycat"
	ExchangePotatoSwap                 Exchange = "potato-swap"
	ExchangePrimeETH                   Exchange = "primeeth"
	ExchangePufferPufETH               Exchange = "puffer-pufeth"
	ExchangePunkSwap                   Exchange = "punkswap"
	ExchangeQuickPerps                 Exchange = "quickperps"
	ExchangeQuickSwap                  Exchange = "quickswap"
	ExchangeQuickSwapUniV3             Exchange = "quickswap-uni-v3"
	ExchangeQuickSwapV3                Exchange = "quickswap-v3"
	ExchangeRamses                     Exchange = "ramses"
	ExchangeRamsesV2                   Exchange = "ramses-v2"
	ExchangeRenzoEZETH                 Exchange = "renzo-ezeth"
	ExchangeRetro                      Exchange = "retro"
	ExchangeRetroV3                    Exchange = "retro-v3"
	ExchangeRevoSwap                   Exchange = "revo-swap"
	ExchangeRingSwap                   Exchange = "ringswap"
	ExchangeRocketPoolRETH             Exchange = "rocketpool-reth"
	ExchangeRocketSwapV2               Exchange = "rocketswap-v2"
	ExchangeRoguex                     Exchange = "roguex"
	ExchangeSaddle                     Exchange = "saddle"
	ExchangeSafeSwap                   Exchange = "safeswap"
	ExchangeSavingsUSDS                Exchange = "savings-usds"
	ExchangeSboom                      Exchange = "sboom"
	ExchangeScale                      Exchange = "scale"
	ExchangeScribe                     Exchange = "scribe"
	ExchangeScrollSwap                 Exchange = "scrollswap"
	ExchangeSectaV2                    Exchange = "secta-v2"
	ExchangeSectaV3                    Exchange = "secta-v3"
	ExchangeSfrxETH                    Exchange = "sfrxeth"
	ExchangeSfrxETHConvertor           Exchange = "sfrxeth-convertor"
	ExchangeShadowDex                  Exchange = "shadow-dex"
	ExchangeShadowLegacy               Exchange = "shadow-legacy"
	ExchangeShibaSwap                  Exchange = "shibaswap"
	ExchangeSilverSwap                 Exchange = "silverswap"
	ExchangeSkyPSM                     Exchange = "sky-psm"
	ExchangeSkydrome                   Exchange = "skydrome"
	ExchangeSkydromeV2                 Exchange = "skydrome-v2"
	ExchangeSmardex                    Exchange = "smardex"
	ExchangeSoSwap                     Exchange = "soswap"
	ExchangeSolidlyV2                  Exchange = "solidly-v2"
	ExchangeSolidlyV3                  Exchange = "solidly-v3"
	ExchangeSonicMarket                Exchange = "sonic-market"
	ExchangeSpacefi                    Exchange = "spacefi"
	ExchangeSpartaDex                  Exchange = "sparta-dex"
	ExchangeSpiritSwap                 Exchange = "spiritswap"
	ExchangeSpookySwap                 Exchange = "spookyswap"
	ExchangeSpookySwapV3               Exchange = "spookyswap-v3"
	ExchangeSquadSwap                  Exchange = "squadswap"
	ExchangeSquadSwapV2                Exchange = "squadswap-v2"
	ExchangeSquadSwapV3                Exchange = "squadswap-v3"
	ExchangeStaderETHx                 Exchange = "staderethx"
	ExchangeStationDexV2               Exchange = "station-dex-v2"
	ExchangeStationDexV3               Exchange = "station-dex-v3"
	ExchangeStratumFinance             Exchange = "stratum-finance"
	ExchangeSuperSwapV3                Exchange = "superswap-v3"
	ExchangeSushiSwap                  Exchange = "sushiswap"
	ExchangeSushiSwapV3                Exchange = "sushiswap-v3"
	ExchangeSwaapV2                    Exchange = "swaap-v2"
	ExchangeSwapBased                  Exchange = "swapbased"
	ExchangeSwapBasedPerp              Exchange = "swapbased-perp"
	ExchangeSwapBasedV3                Exchange = "swapbased-v3"
	ExchangeSwapBlast                  Exchange = "swap-blast"
	ExchangeSwapXCL                    Exchange = "swap-x-cl"
	ExchangeSwapXV2                    Exchange = "swap-x-v2"
	ExchangeSwapr                      Exchange = "swapr"
	ExchangeSwapsicle                  Exchange = "swapsicle"
	ExchangeSwellRSWETH                Exchange = "swell-rsweth"
	ExchangeSwellSWETH                 Exchange = "swell-sweth"
	ExchangeSynapse                    Exchange = "synapse"
	ExchangeSyncSwap                   Exchange = "syncswap"
	ExchangeSyncSwapCL                 Exchange = "syncswap-cl"
	ExchangeSyncSwapV2Aqua             Exchange = "syncswapv2-aqua"
	ExchangeSyncSwapV2Classic          Exchange = "syncswapv2-classic"
	ExchangeSyncSwapV2Stable           Exchange = "syncswapv2-stable"
	ExchangeSynthSwap                  Exchange = "synthswap"
	ExchangeSynthSwapPerp              Exchange = "synthswap-perp"
	ExchangeSynthSwapV3                Exchange = "synthswap-v3"
	ExchangeSynthetix                  Exchange = "synthetix"
	ExchangeThena                      Exchange = "thena"
	ExchangeThenaFusion                Exchange = "thena-fusion"
	ExchangeThenaFusionV3              Exchange = "thena-fusion-v3"
	ExchangeThick                      Exchange = "thick"
	ExchangeThrusterV2                 Exchange = "thruster-v2"
	ExchangeThrusterV2Degen            Exchange = "thruster-v2-degen"
	ExchangeThrusterV3                 Exchange = "thruster-v3"
	ExchangeTokan                      Exchange = "tokan-exchange"
	ExchangeTraderJoe                  Exchange = "traderjoe"
	ExchangeTraderJoeV20               Exchange = "traderjoe-v20"
	ExchangeTraderJoeV21               Exchange = "traderjoe-v21"
	ExchangeTraderJoeV22               Exchange = "traderjoe-v22"
	ExchangeTrisolaris                 Exchange = "trisolaris"
	ExchangeTsunamiX                   Exchange = "tsunami-x"
	ExchangeUSDFi                      Exchange = "usdfi"
	ExchangeUcsFinance                 Exchange = "ucs-finance"
	ExchangeUnchainX                   Exchange = "unchainx"
	ExchangeUniSwap                    Exchange = "uniswap"
	ExchangeUniSwapV1                  Exchange = "uniswap-v1"
	ExchangeUniSwapV2                  Exchange = "uniswap-v2"
	ExchangeUniSwapV3                  Exchange = "uniswapv3"
	ExchangeUniswapV4                  Exchange = "uniswap-v4"
	ExchangeUniswapV4BunniV2           Exchange = "uniswap-v4-bunni-v2"
	ExchangeUniswapV4FairFlow          Exchange = "uniswap-v4-fairflow"
	ExchangeUniswapV4Kem               Exchange = "uniswap-v4-kem"
	ExchangeUsd0PP                     Exchange = "usd0pp"
	ExchangeVVS                        Exchange = "vvs"
	ExchangeValleySwap                 Exchange = "valleyswap"
	ExchangeValleySwapV2               Exchange = "valleyswap-v2"
	ExchangeVelocore                   Exchange = "velocore"
	ExchangeVelocoreV2CPMM             Exchange = "velocore-v2-cpmm"
	ExchangeVelocoreV2WombatStable     Exchange = "velocore-v2-wombat-stable"
	ExchangeVelodrome                  Exchange = "velodrome"
	ExchangeVelodromeCL                Exchange = "velodrome-cl"
	ExchangeVelodromeCL2               Exchange = "velodrome-cl-2"
	ExchangeVelodromeV2                Exchange = "velodrome-v2"
	ExchangeVerse                      Exchange = "verse"
	ExchangeVesync                     Exchange = "vesync"
	ExchangeVirtualFun                 Exchange = "virtual-fun"
	ExchangeVodoo                      Exchange = "vodoo"
	ExchangeVooi                       Exchange = "vooi"
	ExchangeWBETH                      Exchange = "wbeth"
	ExchangeWagmi                      Exchange = "wagmi"
	ExchangeWagyuSwap                  Exchange = "wagyuswap"
	ExchangeWannaSwap                  Exchange = "wannaswap"
	ExchangeWasabi                     Exchange = "wasabi"
	ExchangeWault                      Exchange = "wault"
	ExchangeWigoSwap                   Exchange = "wigo-swap"
	ExchangeWombat                     Exchange = "wombat"
	ExchangeWooFiV2                    Exchange = "woofi-v2"
	ExchangeWooFiV3                    Exchange = "woofi-v3"
	ExchangeXLayerSwap                 Exchange = "xlayer-swap"
	ExchangeYetiSwap                   Exchange = "yetiswap"
	ExchangeYuzuSwap                   Exchange = "yuzuswap"
	ExchangeZKSwap                     Exchange = "zkswap"
	ExchangeZebra                      Exchange = "zebra"
	ExchangeZebraV2                    Exchange = "zebra-v2"
	ExchangeZero                       Exchange = "zero"
	ExchangeZipSwap                    Exchange = "zipswap"
	ExchangeZkEraFinance               Exchange = "zkera-finance"
	ExchangeZkSwapFinance              Exchange = "zkswap-finance"
	ExchangeZkSwapStable               Exchange = "zkswap-stable"
	ExchangeZkSwapV3                   Exchange = "zkswap-v3"
	ExchangeZyberSwapV3                Exchange = "zyberswap-v3"
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
	ExchangePancakeInfinityCL:          {},
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

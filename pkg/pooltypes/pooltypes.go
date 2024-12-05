package pooltypes

import (
	algebraintegral "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
	balancerv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v1"
	balancerv2composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
	balancerv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/stable"
	balancerv2weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/weighted"
	bancorv21 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v21"
	bancorv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bedrock/unieth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	curveStableMetaNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-meta-ng"
	curveStableNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	curveTricryptoNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng"
	curveTwocryptoNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	daiusds "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dai-usds"
	deltaswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/deltaswap-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dodoclassical "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/classical"
	dododpp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dpp"
	dododsp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dsp"
	dododvm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dvm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ethena/susde"
	ethervista "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ether-vista"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/eeth"
	etherfivampire "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/vampire"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/weeth"
	fluidDexT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	fluidVaultT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/vault-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth"
	sfrxeth_convertor "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth-convertor"
	generic_simple_rate "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/generic-simple-rate"
	gyro2clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/2clp"
	gyro3clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/3clp"
	gyroeclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/eclp"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/integral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/rseth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/litepsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/meth"
	maverickv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick-v2"
	mkrsky "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mkr-sky"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nomiswap"
	ondo_usdy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ondo-usdy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/primeeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/puffer/pufeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/renzo/ezeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ringswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/rocketpool/reth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/staderethx"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/rsweth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/sweth"
	syncswapv2aqua "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/aqua"
	syncswapv2classic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/classic"
	syncswapv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/stable"
	uniswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v1"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/usd0pp"
	velocorev2cpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/cpmm"
	velocorev2wombatstable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/wombat-stable"
	velodrome "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	woofiv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v2"
	woofiv21 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/algebrav1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer"
	balancercomposablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer-composable-stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/biswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/camelot"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dystopia"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/elastic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/equalizer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fraxswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fulcrom"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fxdx"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	gmxglp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx-glp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ironstable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap"
	kokonutcrypto "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kokonut-crypto"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido"
	lidosteth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido-steth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/madmex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/makerpsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/mantisswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/muteswitch"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nerve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nuriv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/oneswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pancakev3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pearl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/platypus"
	polmatic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pol-matic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/polydex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/quickperps"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramses"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/slipstream"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/smardex"
	solidlyv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3"
	swapbasedperp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/swapbased-perp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoev20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/usdfi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocimeter"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/vooi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	zkerafinance "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/zkera-finance"
)

type Types struct {
	CurveBase                      string
	CurvePlainOracle               string
	CurveMeta                      string
	CurveAave                      string
	CurveCompound                  string
	CurveLending                   string
	CurveTricrypto                 string
	CurveTwo                       string
	Uni                            string
	UniswapV3                      string
	Biswap                         string
	Polydex                        string
	Firebird                       string
	Dmm                            string
	Elastic                        string
	Saddle                         string
	Nerve                          string
	OneSwap                        string
	IronStable                     string
	DodoClassical                  string
	DodoVendingMachine             string
	DodoStablePool                 string
	DodoPrivatePool                string
	Velodrome                      string
	VelodromeV2                    string
	Velocimeter                    string
	Pearl                          string
	Ramses                         string
	RamsesV2                       string
	Dystopia                       string
	PlatypusBase                   string
	PlatypusPure                   string
	PlatypusAvax                   string
	WombatMain                     string
	WombatLsd                      string
	GMX                            string
	GMXGLP                         string
	MakerPSM                       string
	Synthetix                      string
	MadMex                         string
	Metavault                      string
	Lido                           string
	LidoStEth                      string
	LimitOrder                     string
	Fraxswap                       string
	Camelot                        string
	MuteSwitch                     string
	SyncSwapClassic                string
	SyncSwapStable                 string
	SyncSwapV2Classic              string
	SyncSwapV2Stable               string
	SyncSwapV2Aqua                 string
	PancakeV3                      string
	MaverickV1                     string
	AlgebraV1                      string
	TraderJoeV20                   string
	KyberPMM                       string
	IZiSwap                        string
	WooFiV2                        string
	WooFiV21                       string
	Equalizer                      string
	SwapBasedPerp                  string
	USDFi                          string
	ZkSwapFinance                  string
	MantisSwap                     string
	Vooi                           string
	PolMatic                       string
	KokonutCrypto                  string
	LiquidityBookV21               string
	LiquidityBookV20               string
	Smardex                        string
	Integral                       string
	Fxdx                           string
	UniswapV1                      string
	UniswapV2                      string
	QuickPerps                     string
	BalancerV1                     string
	BalancerV2Weighted             string
	BalancerV2Stable               string
	BalancerV2ComposableStable     string
	VelocoreV2CPMM                 string
	VelocoreV2WombatStable         string
	Fulcrom                        string
	SolidlyV3                      string
	LegacyBalancerWeighted         string
	LegacyBalancerStable           string
	LegacyBalancerMetaStable       string
	LegacyBalancerComposableStable string
	Gyroscope2CLP                  string
	Gyroscope3CLP                  string
	GyroscopeECLP                  string
	ZkEraFinance                   string
	SwaapV2                        string
	EtherfiEETH                    string
	EtherfiWEETH                   string
	SwellSWETH                     string
	SwellRSWETH                    string
	BedrockUniETH                  string
	PufferPufETH                   string
	BancorV21                      string
	BancorV3                       string
	CurveStablePlain               string
	CurveStableNg                  string
	CurveStableMetaNg              string
	CurveTriCryptoNg               string
	CurveTwoCryptoNg               string
	KelpRSETH                      string
	RocketPoolRETH                 string
	EthenaSusde                    string
	MakerSavingsDai                string
	HashflowV3                     string
	NomiSwapStable                 string
	NativeV1                       string
	RenzoEZETH                     string
	Slipstream                     string
	NuriV2                         string
	EtherVista                     string
	MkrSky                         string
	DaiUsds                        string
	MaverickV2                     string
	LitePSM                        string
	Usd0PP                         string
	Bebop                          string
	Dexalot                        string
	GenericSimpleRate              string
	RingSwap                       string
	PrimeETH                       string
	StaderETHx                     string
	FluidVaultT1                   string
	FluidDexT1                     string
	MantleETH                      string
	OndoUSDY                       string
	Clipper                        string
	DeltaSwapV1                    string
	SfrxETH                        string
	SfrxETHConvertor               string
	EtherfiVampire                 string
	AlgebraIntegral                string
}

var (
	PoolTypes = Types{
		CurveBase:                      curve.PoolTypeBase,
		CurvePlainOracle:               curve.PoolTypePlainOracle,
		CurveMeta:                      curve.PoolTypeMeta,
		CurveAave:                      curve.PoolTypeAave,
		CurveCompound:                  curve.PoolTypeCompound,
		CurveLending:                   curve.PoolTypeLending,
		CurveTricrypto:                 curve.PoolTypeTricrypto,
		CurveTwo:                       curve.PoolTypeTwo,
		Uni:                            uniswap.DexTypeUniswap,
		UniswapV3:                      uniswapv3.DexTypeUniswapV3,
		Biswap:                         biswap.DexTypeBiswap,
		Polydex:                        polydex.DexTypePolydex,
		Dmm:                            dmm.DexTypeDMM,
		Elastic:                        elastic.DexTypeElastic,
		Saddle:                         saddle.DexTypeSaddle,
		Nerve:                          nerve.DexTypeNerve,
		OneSwap:                        oneswap.DexTypeOneSwap,
		IronStable:                     ironstable.DexTypeIronStable,
		DodoClassical:                  dodoclassical.PoolType,
		DodoVendingMachine:             dododvm.PoolType,
		DodoStablePool:                 dododsp.PoolType,
		DodoPrivatePool:                dododpp.PoolType,
		Velodrome:                      velodrome.DexType,
		VelodromeV2:                    velodromev2.DexType,
		Velocimeter:                    velocimeter.DexTypeVelocimeter,
		Pearl:                          pearl.DexTypePearl,
		Ramses:                         ramses.DexTypeRamses,
		RamsesV2:                       ramsesv2.DexTypeRamsesV2,
		Dystopia:                       dystopia.DexTypeDystopia,
		PlatypusBase:                   platypus.PoolTypePlatypusBase,
		PlatypusPure:                   platypus.PoolTypePlatypusPure,
		PlatypusAvax:                   platypus.PoolTypePlatypusAvax,
		WombatMain:                     wombat.PoolTypeWombatMain,
		WombatLsd:                      wombat.PoolTypeWombatLSD,
		GMX:                            gmx.DexTypeGmx,
		GMXGLP:                         gmxglp.DexTypeGmxGlp,
		MakerPSM:                       makerpsm.DexTypeMakerPSM,
		Synthetix:                      synthetix.DexTypeSynthetix,
		MadMex:                         madmex.DexTypeMadmex,
		Metavault:                      metavault.DexTypeMetavault,
		Lido:                           lido.DexTypeLido,
		LidoStEth:                      lidosteth.DexTypeLidoStETH,
		LimitOrder:                     limitorder.DexTypeLimitOrder,
		Fraxswap:                       fraxswap.DexTypeFraxswap,
		Camelot:                        camelot.DexTypeCamelot,
		MuteSwitch:                     muteswitch.DexTypeMuteSwitch,
		SyncSwapClassic:                syncswap.PoolTypeSyncSwapClassic,
		SyncSwapStable:                 syncswap.PoolTypeSyncSwapStable,
		SyncSwapV2Classic:              syncswapv2classic.PoolTypeSyncSwapV2Classic,
		SyncSwapV2Stable:               syncswapv2stable.PoolTypeSyncSwapV2Stable,
		SyncSwapV2Aqua:                 syncswapv2aqua.PoolTypeSyncSwapV2Aqua,
		PancakeV3:                      pancakev3.DexTypePancakeV3,
		MaverickV1:                     maverickv1.DexTypeMaverickV1,
		AlgebraV1:                      algebrav1.DexTypeAlgebraV1,
		TraderJoeV20:                   traderjoev20.DexTypeTraderJoeV20,
		KyberPMM:                       kyberpmm.DexTypeKyberPMM,
		IZiSwap:                        iziswap.DexTypeiZiSwap,
		WooFiV2:                        woofiv2.DexTypeWooFiV2,
		WooFiV21:                       woofiv21.DexTypeWooFiV21,
		Equalizer:                      equalizer.DexTypeEqualizer,
		SwapBasedPerp:                  swapbasedperp.DexTypeSwapBasedPerp,
		USDFi:                          usdfi.DexTypeUSDFi,
		MantisSwap:                     mantisswap.DexTypeMantisSwap,
		Vooi:                           vooi.DexTypeVooi,
		PolMatic:                       polmatic.DexTypePolMatic,
		KokonutCrypto:                  kokonutcrypto.DexTypeKokonutCrypto,
		LiquidityBookV21:               liquiditybookv21.DexTypeLiquidityBookV21,
		LiquidityBookV20:               liquiditybookv20.DexTypeLiquidityBookV20,
		Smardex:                        smardex.DexTypeSmardex,
		Integral:                       integral.DexTypeIntegral,
		Fxdx:                           fxdx.DexTypeFxdx,
		UniswapV1:                      uniswapv1.DexType,
		UniswapV2:                      uniswapv2.DexType,
		QuickPerps:                     quickperps.DexTypeQuickperps,
		BalancerV1:                     balancerv1.DexType,
		BalancerV2Weighted:             balancerv2weighted.DexType,
		BalancerV2Stable:               balancerv2stable.DexType,
		BalancerV2ComposableStable:     balancerv2composablestable.DexType,
		VelocoreV2CPMM:                 velocorev2cpmm.DexType,
		VelocoreV2WombatStable:         velocorev2wombatstable.DexType,
		Fulcrom:                        fulcrom.DexTypeFulcrom,
		SolidlyV3:                      solidlyv3.DexTypeSolidlyV3,
		LegacyBalancerWeighted:         string(balancer.DexTypeBalancerWeighted),
		LegacyBalancerStable:           string(balancer.DexTypeBalancerStable),
		LegacyBalancerMetaStable:       string(balancer.DexTypeBalancerMetaStable),
		LegacyBalancerComposableStable: string(balancercomposablestable.DexTypeBalancerComposableStable),
		Gyroscope2CLP:                  gyro2clp.DexType,
		Gyroscope3CLP:                  gyro3clp.DexType,
		GyroscopeECLP:                  gyroeclp.DexType,
		ZkEraFinance:                   zkerafinance.DexType,
		SwaapV2:                        swaapv2.DexType,
		EtherfiEETH:                    eeth.DexType,
		EtherfiWEETH:                   weeth.DexType,
		BancorV21:                      bancorv21.DexType,
		BancorV3:                       bancorv3.DexType,
		CurveStablePlain:               plain.DexType,
		CurveStableNg:                  curveStableNg.DexType,
		CurveStableMetaNg:              curveStableMetaNg.DexType,
		CurveTriCryptoNg:               curveTricryptoNg.DexType,
		CurveTwoCryptoNg:               curveTwocryptoNg.DexType,
		KelpRSETH:                      rseth.DexType,
		RocketPoolRETH:                 reth.DexType,
		SwellSWETH:                     sweth.DexType,
		SwellRSWETH:                    rsweth.DexType,
		BedrockUniETH:                  unieth.DexType,
		PufferPufETH:                   pufeth.DexType,
		EthenaSusde:                    susde.DexType,
		MakerSavingsDai:                savingsdai.DexType,
		HashflowV3:                     hashflowv3.DexType,
		NomiSwapStable:                 nomiswap.DexType,
		NativeV1:                       nativev1.DexType,
		RenzoEZETH:                     ezeth.DexType,
		Slipstream:                     slipstream.DexType,
		NuriV2:                         nuriv2.DexType,
		EtherVista:                     ethervista.DexType,
		MkrSky:                         mkrsky.DexType,
		DaiUsds:                        daiusds.DexType,
		MaverickV2:                     maverickv2.DexType,
		LitePSM:                        litepsm.DexTypeLitePSM,
		Usd0PP:                         usd0pp.DexType,
		Bebop:                          bebop.DexType,
		Dexalot:                        dexalot.DexType,
		GenericSimpleRate:              generic_simple_rate.DexType,
		RingSwap:                       ringswap.DexType,
		PrimeETH:                       primeeth.DexType,
		StaderETHx:                     staderethx.DexType,
		FluidVaultT1:                   fluidVaultT1.DexType,
		FluidDexT1:                     fluidDexT1.DexType,
		MantleETH:                      meth.DexType,
		OndoUSDY:                       ondo_usdy.DexType,
		Clipper:                        clipper.DexType,
		DeltaSwapV1:                    deltaswapv1.DexType,
		SfrxETH:                        sfrxeth.DexType,
		SfrxETHConvertor:               sfrxeth_convertor.DexType,
		EtherfiVampire:                 etherfivampire.DexType,
		AlgebraIntegral:                algebraintegral.DexType,
	}
)

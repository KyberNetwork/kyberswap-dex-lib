package pooltypes

import (
	aavev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/aave-v3"
	algebraintegral "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
	algebrav1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
	angletransmuter "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/angle-transmuter"
	arberaden "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/arbera/den"
	arberazap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/arbera/zap"
	arenabc "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/arena-bc"
	balancerv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v1"
	balancerv2composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/composable-stable"
	balancerv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/stable"
	balancerv2weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/weighted"
	balancerv3eclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/eclp"
	balancerv3quantamm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/quant-amm"
	balancerv3stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/stable"
	balancerv3weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/weighted"
	bancorv21 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v21"
	bancorv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bedrock/unibtc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bedrock/unieth"
	beetsss "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/beets-ss"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/brownfi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/cap/cusd"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	cloberob "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob"
	compoundv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/compound/v2"
	compoundv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/compound/v3"
	curvelending "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/lending"
	curvellamma "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/llamma"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	curvestablemetang "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-meta-ng"
	curvestableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	curvetricryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng"
	curvetwocryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	daiusds "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dai-usds"
	deltaswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/deltaswap-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dodoclassical "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/classical"
	dododpp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dpp"
	dododsp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dsp"
	dododvm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dvm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ethena/susde"
	ethervista "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ether-vista"
	etherfiebtc "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/ebtc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/eeth"
	etherfivampire "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/vampire"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/weeth"
	eulerswap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap"
	fluidDexLite "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-lite"
	fluidDexT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	dexv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2"
	fluidVaultT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/vault-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth"
	sfrxethconvertor "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth-convertor"
	genericarm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/generic-arm"
	genericsimplerate "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/generic-simple-rate"
	gsm4626 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gsm-4626"
	gyro2clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/2clp"
	gyro3clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/3clp"
	gyroeclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/eclp"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/honey"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hyeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/infinitypools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/integral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/rseth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/litepsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	skypsm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/sky-psm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/meth"
	maplesyrup "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maple-syrup"
	maverickv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick/v1"
	maverickv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/midas"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mimswap"
	miromigrator "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/miro-migrator"
	mkrsky "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mkr-sky"
	nativev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nomiswap"
	ondousdy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ondo-usdy"
	overnightusdp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/overnight-usdp"
	pancakeinfinitybin "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin"
	pancakeinfinitycl "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	pancakestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/stable"
	pancakev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pandafun"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/primeeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/puffer/pufeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/renzo/ezeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ringswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/rocketpool/reth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/smardex"
	solidlyv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/solidly-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/staderethx"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	swapxv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swap-x-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/rsweth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/sweth"
	syncswapv2aqua "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/aqua"
	syncswapv2classic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/classic"
	syncswapv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/stable"
	uniswaplo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/lo"
	uniswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v1"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/usd0pp"
	velocorev2cpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/cpmm"
	velocorev2wombatstable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/wombat-stable"
	velodrome "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	virtualfun "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/virtual-fun"
	woofiv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v2"
	woofiv21 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/xsolvbtc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/biswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/camelot"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dmm"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido"
	lidosteth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido-steth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/madmex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/makerpsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/mantisswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/muteswitch"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nerve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nuriv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/oneswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/platypus"
	polmatic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pol-matic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/polydex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/quickperps"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/slipstream"
	solidlyv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3"
	swapbasedperp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/swapbased-perp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/usdfi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocimeter"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/vooi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Types struct {
	AaveV3                     string
	AlgebraIntegral            string
	AlgebraV1                  string
	Ambient                    string
	AngleTransmuter            string
	ArberaDen                  string
	ArberaZap                  string
	ArenaBC                    string
	BalancerV1                 string
	BalancerV2ComposableStable string
	BalancerV2Stable           string
	BalancerV2Weighted         string
	BalancerV3ECLP             string
	BalancerV3QuantAMM         string
	BalancerV3Stable           string
	BalancerV3Weighted         string
	BancorV21                  string
	BancorV3                   string
	Bebop                      string
	BedrockUniBTC              string
	BedrockUniETH              string
	BeetsSS                    string
	Biswap                     string
	Brownfi                    string
	Camelot                    string
	Clipper                    string
	CloberOB                   string
	CompoundV2                 string
	CompoundV3                 string
	CurveAave                  string
	CurveBase                  string
	CurveCompound              string
	CurveLending               string
	CurveLlamma                string
	CurveMeta                  string
	CurvePlainOracle           string
	CurveStableMetaNg          string
	CurveStableNg              string
	CurveStablePlain           string
	CurveTricrypto             string
	CurveTriCryptoNg           string
	CurveTwo                   string
	CurveTwoCryptoNg           string
	CUSD                       string
	DaiUsds                    string
	DeltaSwapV1                string
	Dexalot                    string
	Dmm                        string
	DodoClassical              string
	DodoPrivatePool            string
	DodoStablePool             string
	DodoVendingMachine         string
	Ekubo                      string
	Elastic                    string
	Equalizer                  string
	ERC4626                    string
	EthenaSusde                string
	EtherFieBTC                string
	EtherfiEETH                string
	EtherfiVampire             string
	EtherfiWEETH               string
	EtherVista                 string
	EulerSwap                  string
	FluidDexLite               string
	FluidDexT1                 string
	FluidDexV2                 string
	FluidVaultT1               string
	Fraxswap                   string
	Fulcrom                    string
	Fxdx                       string
	GenericArm                 string
	GenericSimpleRate          string
	GMX                        string
	GMXGLP                     string
	Gsm4626                    string
	Gyroscope2CLP              string
	Gyroscope3CLP              string
	GyroscopeECLP              string
	HashflowV3                 string
	Honey                      string
	HyETH                      string
	InfinityPools              string
	Integral                   string
	IronStable                 string
	IZiSwap                    string
	KelpRSETH                  string
	KokonutCrypto              string
	KyberPMM                   string
	Lido                       string
	LidoStEth                  string
	LimitOrder                 string
	LiquidityBookV20           string
	LiquidityBookV21           string
	LitePSM                    string
	LO1inch                    string
	MadMex                     string
	MakerPSM                   string
	MakerSavingsDai            string
	MantisSwap                 string
	MantleETH                  string
	MapleSyrup                 string
	MaverickV1                 string
	MaverickV2                 string
	Metavault                  string
	Midas                      string
	MimSwap                    string
	MiroMigrator               string
	MkrSky                     string
	MuteSwitch                 string
	NativeV3                   string
	Nerve                      string
	NomiSwapStable             string
	NuriV2                     string
	OndoUSDY                   string
	OneSwap                    string
	OvernightUsdp              string
	PancakeInfinityBin         string
	PancakeInfinityCL          string
	PancakeStable              string
	PancakeV3                  string
	PandaFun                   string
	PlatypusAvax               string
	PlatypusBase               string
	PlatypusPure               string
	Pmm1                       string
	Pmm2                       string
	Pmm3                       string
	PolMatic                   string
	Polydex                    string
	PrimeETH                   string
	PufferPufETH               string
	QuickPerps                 string
	RamsesV2                   string
	RenzoEZETH                 string
	RingSwap                   string
	RocketPoolRETH             string
	Saddle                     string
	SfrxETH                    string
	SfrxETHConvertor           string
	SkyPSM                     string
	Slipstream                 string
	Smardex                    string
	SolidlyV2                  string
	SolidlyV3                  string
	StaderETHx                 string
	SwaapV2                    string
	SwapBasedPerp              string
	SwapXV2                    string
	SwellRSWETH                string
	SwellSWETH                 string
	SyncSwapClassic            string
	SyncSwapStable             string
	SyncSwapV2Aqua             string
	SyncSwapV2Classic          string
	SyncSwapV2Stable           string
	Synthetix                  string
	Uni                        string
	UniswapLO                  string
	UniswapV1                  string
	UniswapV2                  string
	UniswapV3                  string
	UniswapV4                  string
	Usd0PP                     string
	USDFi                      string
	Velocimeter                string
	VelocoreV2CPMM             string
	VelocoreV2WombatStable     string
	Velodrome                  string
	VelodromeV2                string
	VirtualFun                 string
	Vooi                       string
	WombatLsd                  string
	WombatMain                 string
	WooFiV2                    string
	WooFiV21                   string
	XsolvBTC                   string
}

var (
	// PoolTypes is a list of supported pool types.
	PoolTypes = Types{
		AaveV3:                     aavev3.DexType,
		AlgebraIntegral:            algebraintegral.DexType,
		AlgebraV1:                  algebrav1.DexTypeAlgebraV1,
		Ambient:                    ambient.DexTypeAmbient,
		AngleTransmuter:            angletransmuter.DexType,
		ArberaDen:                  arberaden.DexType,
		ArberaZap:                  arberazap.DexType,
		ArenaBC:                    arenabc.DexType,
		BalancerV1:                 balancerv1.DexType,
		BalancerV2ComposableStable: balancerv2composablestable.DexType,
		BalancerV2Stable:           balancerv2stable.DexType,
		BalancerV2Weighted:         balancerv2weighted.DexType,
		BalancerV3ECLP:             balancerv3eclp.DexType,
		BalancerV3QuantAMM:         balancerv3quantamm.DexType,
		BalancerV3Stable:           balancerv3stable.DexType,
		BalancerV3Weighted:         balancerv3weighted.DexType,
		BancorV21:                  bancorv21.DexType,
		BancorV3:                   bancorv3.DexType,
		Bebop:                      bebop.DexType,
		BedrockUniBTC:              unibtc.DexType,
		BedrockUniETH:              unieth.DexType,
		BeetsSS:                    beetsss.DexType,
		Biswap:                     biswap.DexTypeBiswap,
		Brownfi:                    brownfi.DexType,
		Camelot:                    camelot.DexTypeCamelot,
		Clipper:                    clipper.DexType,
		CloberOB:                   cloberob.DexType,
		CompoundV2:                 compoundv2.DexType,
		CompoundV3:                 compoundv3.DexType,
		CurveAave:                  curve.PoolTypeAave,
		CurveBase:                  curve.PoolTypeBase,
		CurveCompound:              curve.PoolTypeCompound,
		CurveLending:               curvelending.DexType,
		CurveLlamma:                curvellamma.DexType,
		CurveMeta:                  curve.PoolTypeMeta,
		CurvePlainOracle:           curve.PoolTypePlainOracle,
		CurveStableMetaNg:          curvestablemetang.DexType,
		CurveStableNg:              curvestableng.DexType,
		CurveStablePlain:           plain.DexType,
		CurveTricrypto:             curve.PoolTypeTricrypto,
		CurveTriCryptoNg:           curvetricryptong.DexType,
		CurveTwo:                   curve.PoolTypeTwo,
		CurveTwoCryptoNg:           curvetwocryptong.DexType,
		CUSD:                       cusd.DexType,
		DaiUsds:                    daiusds.DexType,
		DeltaSwapV1:                deltaswapv1.DexType,
		Dexalot:                    dexalot.DexType,
		Dmm:                        dmm.DexTypeDMM,
		DodoClassical:              dodoclassical.PoolType,
		DodoPrivatePool:            dododpp.PoolType,
		DodoStablePool:             dododsp.PoolType,
		DodoVendingMachine:         dododvm.PoolType,
		Ekubo:                      ekubo.DexType,
		Elastic:                    elastic.DexTypeElastic,
		Equalizer:                  equalizer.DexTypeEqualizer,
		ERC4626:                    erc4626.DexType,
		EthenaSusde:                susde.DexType,
		EtherFieBTC:                etherfiebtc.DexType,
		EtherfiEETH:                eeth.DexType,
		EtherfiVampire:             etherfivampire.DexType,
		EtherfiWEETH:               weeth.DexType,
		EtherVista:                 ethervista.DexType,
		EulerSwap:                  eulerswap.DexType,
		FluidDexLite:               fluidDexLite.DexType,
		FluidDexT1:                 fluidDexT1.DexType,
		FluidDexV2:                 dexv2.DexType,
		FluidVaultT1:               fluidVaultT1.DexType,
		Fraxswap:                   fraxswap.DexTypeFraxswap,
		Fulcrom:                    fulcrom.DexTypeFulcrom,
		Fxdx:                       fxdx.DexTypeFxdx,
		GenericArm:                 genericarm.DexType,
		GenericSimpleRate:          genericsimplerate.DexType,
		GMX:                        gmx.DexTypeGmx,
		GMXGLP:                     gmxglp.DexTypeGmxGlp,
		Gsm4626:                    gsm4626.DexType,
		Gyroscope2CLP:              gyro2clp.DexType,
		Gyroscope3CLP:              gyro3clp.DexType,
		GyroscopeECLP:              gyroeclp.DexType,
		HashflowV3:                 hashflowv3.DexType,
		Honey:                      honey.DexType,
		HyETH:                      hyeth.DexType,
		InfinityPools:              infinitypools.DexType,
		Integral:                   integral.DexTypeIntegral,
		IronStable:                 ironstable.DexTypeIronStable,
		IZiSwap:                    iziswap.DexTypeiZiSwap,
		KelpRSETH:                  rseth.DexType,
		KokonutCrypto:              kokonutcrypto.DexTypeKokonutCrypto,
		KyberPMM:                   valueobject.ExchangeKyberPMM,
		Lido:                       lido.DexTypeLido,
		LidoStEth:                  lidosteth.DexTypeLidoStETH,
		LimitOrder:                 limitorder.DexTypeLimitOrder,
		LiquidityBookV20:           liquiditybookv20.DexTypeLiquidityBookV20,
		LiquidityBookV21:           liquiditybookv21.DexTypeLiquidityBookV21,
		LitePSM:                    litepsm.DexTypeLitePSM,
		LO1inch:                    lo1inch.DexType,
		MadMex:                     madmex.DexTypeMadmex,
		MakerPSM:                   makerpsm.DexTypeMakerPSM,
		MakerSavingsDai:            savingsdai.DexType,
		MantisSwap:                 mantisswap.DexTypeMantisSwap,
		MantleETH:                  meth.DexType,
		MapleSyrup:                 maplesyrup.DexType,
		MaverickV1:                 maverickv1.DexTypeMaverickV1,
		MaverickV2:                 maverickv2.DexType,
		Metavault:                  metavault.DexTypeMetavault,
		Midas:                      midas.DexType,
		MimSwap:                    mimswap.DexType,
		MiroMigrator:               miromigrator.DexType,
		MkrSky:                     mkrsky.DexType,
		MuteSwitch:                 muteswitch.DexTypeMuteSwitch,
		NativeV3:                   nativev3.DexType,
		Nerve:                      nerve.DexTypeNerve,
		NomiSwapStable:             nomiswap.DexType,
		NuriV2:                     nuriv2.DexType,
		OndoUSDY:                   ondousdy.DexType,
		OneSwap:                    oneswap.DexTypeOneSwap,
		OvernightUsdp:              overnightusdp.DexType,
		PancakeInfinityBin:         pancakeinfinitybin.DexType,
		PancakeInfinityCL:          pancakeinfinitycl.DexType,
		PancakeStable:              pancakestable.DexType,
		PancakeV3:                  pancakev3.DexTypePancakeV3,
		PandaFun:                   pandafun.DexType,
		PlatypusAvax:               platypus.PoolTypePlatypusAvax,
		PlatypusBase:               platypus.PoolTypePlatypusBase,
		PlatypusPure:               platypus.PoolTypePlatypusPure,
		Pmm1:                       valueobject.ExchangePmm1,
		Pmm2:                       valueobject.ExchangePmm2,
		Pmm3:                       valueobject.ExchangePmm3,
		PolMatic:                   polmatic.DexTypePolMatic,
		Polydex:                    polydex.DexTypePolydex,
		PrimeETH:                   primeeth.DexType,
		PufferPufETH:               pufeth.DexType,
		QuickPerps:                 quickperps.DexTypeQuickperps,
		RamsesV2:                   ramsesv2.DexTypeRamsesV2,
		RenzoEZETH:                 ezeth.DexType,
		RingSwap:                   ringswap.DexType,
		RocketPoolRETH:             reth.DexType,
		Saddle:                     saddle.DexTypeSaddle,
		SfrxETH:                    sfrxeth.DexType,
		SfrxETHConvertor:           sfrxethconvertor.DexType,
		SkyPSM:                     skypsm.DexType,
		Slipstream:                 slipstream.DexType,
		Smardex:                    smardex.DexTypeSmardex,
		SolidlyV2:                  solidlyv2.DexType,
		SolidlyV3:                  solidlyv3.DexTypeSolidlyV3,
		StaderETHx:                 staderethx.DexType,
		SwaapV2:                    swaapv2.DexType,
		SwapBasedPerp:              swapbasedperp.DexTypeSwapBasedPerp,
		SwapXV2:                    swapxv2.DexType,
		SwellRSWETH:                rsweth.DexType,
		SwellSWETH:                 sweth.DexType,
		SyncSwapClassic:            syncswap.PoolTypeSyncSwapClassic,
		SyncSwapStable:             syncswap.PoolTypeSyncSwapStable,
		SyncSwapV2Aqua:             syncswapv2aqua.PoolTypeSyncSwapV2Aqua,
		SyncSwapV2Classic:          syncswapv2classic.PoolTypeSyncSwapV2Classic,
		SyncSwapV2Stable:           syncswapv2stable.PoolTypeSyncSwapV2Stable,
		Synthetix:                  synthetix.DexTypeSynthetix,
		Uni:                        uniswap.DexTypeUniswap,
		UniswapLO:                  uniswaplo.DexType,
		UniswapV1:                  uniswapv1.DexType,
		UniswapV2:                  uniswapv2.DexType,
		UniswapV3:                  uniswapv3.DexTypeUniswapV3,
		UniswapV4:                  uniswapv4.DexType,
		Usd0PP:                     usd0pp.DexType,
		USDFi:                      usdfi.DexTypeUSDFi,
		Velocimeter:                velocimeter.DexTypeVelocimeter,
		VelocoreV2CPMM:             velocorev2cpmm.DexType,
		VelocoreV2WombatStable:     velocorev2wombatstable.DexType,
		Velodrome:                  velodrome.DexType,
		VelodromeV2:                velodromev2.DexType,
		VirtualFun:                 virtualfun.DexType,
		Vooi:                       vooi.DexTypeVooi,
		WombatLsd:                  wombat.PoolTypeWombatLSD,
		WombatMain:                 wombat.PoolTypeWombatMain,
		WooFiV2:                    woofiv2.DexTypeWooFiV2,
		WooFiV21:                   woofiv21.DexTypeWooFiV21,
		XsolvBTC:                   xsolvbtc.DexType,
	}
)

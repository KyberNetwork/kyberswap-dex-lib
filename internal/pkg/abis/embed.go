package abis

import _ "embed"

//go:embed MetaAggregationRouterV2.json
var metaAggregationRouterV2 []byte

//go:embed ERC20.json
var erc20 []byte

//go:embed ERC20DS.json
var erc20ds []byte

//go:embed Multicall.json
var multicall []byte

//go:embed curve/Aave.json
var curveAave []byte

//go:embed curve/AddressProvider.json
var curveAddressProvider []byte

//go:embed curve/AaveV1.json
var curveAaveV1 []byte

//go:embed curve/Compound.json
var curveCompound []byte

//go:embed curve/BasePool.json
var curveBasepool []byte

//go:embed curve/BasePoolV1.json
var curveBasepoolV1 []byte

//go:embed curve/PlainOraclePool.json
var curvePlainOraclePool []byte

//go:embed curve/Oracle.json
var curveOracle []byte

//go:embed curve/Tricrypto.json
var curveTricrypto []byte

//go:embed curve/MetaPool.json
var curveMetapool []byte

//go:embed curve/MetaPoolFactory.json
var curveMetaPoolFactory []byte

//go:embed curve/MainRegistry.json
var curveMainRegistry []byte

//go:embed curve/CryptoRegistry.json
var curveCryptoRegistry []byte

//go:embed curve/CryptoFactory.json
var curveCryptoFactory []byte

//go:embed curve/Two.json
var curveTwo []byte

//go:embed curve/Getter.json
var curveGetter []byte

//go:embed sushiswap/Pair.json
var sushiswapPair []byte

//go:embed quickswap/Pair.json
var quickswapPair []byte

//go:embed dfyn/Pair.json
var dfynPair []byte

//go:embed wault/WaultSwapPair.json
var waultSwapPair []byte

//go:embed jetswap/Pair.json
var jetswapPair []byte

//go:embed polydex/Pair.json
var polydexPair []byte

//go:embed dmm/DmmPool.json
var dmmPool []byte

//go:embed dmm/DmmFactory.json
var dmmFactory []byte

//go:embed polycat/Pair.json
var polycatPair []byte

//go:embed firebird/SwapPair.json
var firebirdSwapPair []byte

//go:embed firebird/OneSwap.json
var firebirdOneSwap []byte

//go:embed firebird/OneSwapFactory.json
var firebirdOneSwapFactory []byte

//go:embed iron/IronSwap.json
var ironSwap []byte

//go:embed nerve/Swap.json
var nerveSwap []byte

//go:embed biswap/BiswapPair.json
var biswapPair []byte

//go:embed biswap/BiswapFactory.json
var biswapFactory []byte

//go:embed balancer/BalancerVault.json
var balancerVault []byte

//go:embed balancer/BalancerPool.json
var balancerPool []byte

//go:embed balancer/MetaStablePool.json
var balancerMetaStablePool []byte

//go:embed aave/AToken.json
var aaveAtoken []byte

//go:embed synapse/SwapFlashLoan.json
var synapseSwapFlashLoan []byte

//go:embed uniswapv3/UniswapV3Pool.json
var uniswapv3Pool []byte

//go:embed uniswapv3/TickLensProxy.json
var uniV3TickLens []byte

//go:embed saddle/SwapFlashLoan.json
var saddleSwapFlashLoan []byte

//go:embed promm/ProMMPool.json
var prommPool []byte

//go:embed dodo/DodoV1Pool.json
var dodoV1 []byte

//go:embed dodo/DodoV2Pool.json
var dodoV2 []byte

//go:embed velodrome/Pair.json
var velodromePair []byte

//go:embed velodrome/Factory.json
var velodromeFactory []byte

//go:embed platypus/Pool.json
var platypusPool []byte

//go:embed platypus/Asset.json
var platypusAsset []byte

//go:embed StakedAVAX.json
var stakedAvax []byte

//go:embed gmx/Vault.json
var gmxVault []byte

//go:embed gmx/VaultPriceFeed.json
var gmxVaultPriceFeed []byte

//go:embed gmx/PancakePair.json
var gmxPancakePair []byte

//go:embed gmx/ChainlinkFlags.json
var gmxChainlinkFlags []byte

//go:embed gmx/FastPriceFeedV1.json
var gmxFastPriceFeedV1 []byte

//go:embed gmx/FastPriceFeedV2.json
var gmxFastPriceFeedV2 []byte

//go:embed gmx/PriceFeed.json
var gmxPriceFeed []byte

//go:embed makerpsm/PSM.json
var makerPSMPSM []byte

//go:embed makerpsm/Vat.json
var makerPSMVat []byte

//go:embed synthetix/Synthetix.json
var synthetix []byte

//go:embed synthetix/SystemSettings.json
var synthetixSystemSettings []byte

//go:embed synthetix/Exchanger.json
var synthetixExchanger []byte

//go:embed synthetix/ExchangerWithFeeRecAlternatives.json
var synthetixExchangerWithFeeRecAlternatives []byte

//go:embed synthetix/ExchangeRates.json
var synthetixExchangeRates []byte

//go:embed synthetix/ExchangeRatesWithDexPricing.json
var synthetixExchangeRatesWithDexPricing []byte

//go:embed synthetix/ChainlinkDataFeed.json
var synthetixChainlinkDataFeed []byte

//go:embed synthetix/DexPriceAggregatorUniswapV3.json
var synthetixDexPriceAggregatorUniswapV3 []byte

//go:embed synthetix/MultiCollateralSynth.json
var synthetixMultiCollateralSynth []byte

//go:embed metavault/Vault.json
var metavaultVault []byte

//go:embed metavault/VaultPriceFeed.json
var metavaultVaultPriceFeed []byte

//go:embed metavault/PancakePair.json
var metavaultPancakePair []byte

//go:embed metavault/ChainlinkFlags.json
var metavaultChainlinkFlags []byte

//go:embed metavault/FastPriceFeedV1.json
var metavaultFastPriceFeedV1 []byte

//go:embed metavault/FastPriceFeedV2.json
var metavaultFastPriceFeedV2 []byte

//go:embed metavault/PriceFeed.json
var metavaultPriceFeed []byte

//go:embed lido/WstETH.json
var lidoWstETH []byte

//go:embed fraxswap/FraxswapFactory.json
var fraxswapFactory []byte

//go:embed fraxswap/FraswapPair.json
var fraxswapPair []byte

//go:embed camelot/CamelotFactory.json
var camelotFactory []byte

//go:embed camelot/CamelotPair.json
var camelotPair []byte

//go:embed optimism/OVMGasPriceOracle.json
var ovmGasPriceOracle []byte

//go:embed arbitrum/ArbGasInfo.json
var arbGasInfo []byte

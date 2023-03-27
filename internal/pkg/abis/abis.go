package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ERC20                                    abi.ABI
	ERC20DS                                  abi.ABI
	Multicall                                abi.ABI
	CurveAddressProvider                     abi.ABI
	CurveAave                                abi.ABI
	CurveAaveV1                              abi.ABI
	CurveCompound                            abi.ABI
	CurveBase                                abi.ABI
	CurveBaseV1                              abi.ABI
	CurvePlainOraclePool                     abi.ABI
	CurveOracle                              abi.ABI
	CurveTricrypto                           abi.ABI
	CurveMeta                                abi.ABI
	CurveMainRegistry                        abi.ABI
	CurveGetter                              abi.ABI
	CurveMetaFactory                         abi.ABI
	CurveCryptoRegistry                      abi.ABI
	CurveCryptoFactory                       abi.ABI
	CurveTwo                                 abi.ABI
	SushiswapPair                            abi.ABI
	QuickswapPair                            abi.ABI
	DfynPair                                 abi.ABI
	WaultPair                                abi.ABI
	JetswapPair                              abi.ABI
	PolydexPair                              abi.ABI
	DmmPool                                  abi.ABI
	DmmFactory                               abi.ABI
	PolycatPair                              abi.ABI
	FirebirdPair                             abi.ABI
	FirebirdOneSwap                          abi.ABI
	FirebirdOneSwapFactory                   abi.ABI
	IronSwap                                 abi.ABI
	NerveSwap                                abi.ABI
	BiswapPair                               abi.ABI
	BiswapFactory                            abi.ABI
	BalancerVault                            abi.ABI
	BalancerPool                             abi.ABI
	BalancerMetaStablePool                   abi.ABI
	AToken                                   abi.ABI
	SynapseSwap                              abi.ABI
	UniswapV3Pool                            abi.ABI
	UniV3TickLens                            abi.ABI
	Saddle                                   abi.ABI
	ProMMPool                                abi.ABI
	MetaAggregationRouterV2                  abi.ABI
	DodoV1                                   abi.ABI
	DodoV2                                   abi.ABI
	VelodromePair                            abi.ABI
	VelodromeFactory                         abi.ABI
	PlatypusPool                             abi.ABI
	PlatypusAsset                            abi.ABI
	StakedAvax                               abi.ABI
	GMXVault                                 abi.ABI
	GMXVaultPriceFeed                        abi.ABI
	GMXPancakePair                           abi.ABI
	GMXChainlinkFlags                        abi.ABI
	GMXFastPriceFeedV1                       abi.ABI
	GMXFastPriceFeedV2                       abi.ABI
	GMXPriceFeed                             abi.ABI
	MakerPSMPSM                              abi.ABI
	MakerPSMVat                              abi.ABI
	Synthetix                                abi.ABI
	SynthetixSystemSettings                  abi.ABI
	SynthetixExchanger                       abi.ABI
	SynthetixExchangerWithFeeRecAlternatives abi.ABI
	SynthetixExchangeRates                   abi.ABI
	SynthetixExchangeRatesWithDexPricing     abi.ABI
	SynthetixChainlinkDataFeed               abi.ABI
	SynthetixDexPriceAggregatorUniswapV3     abi.ABI
	SynthetixMultiCollateralSynth            abi.ABI
	MetavaultVault                           abi.ABI
	MetavaultVaultPriceFeed                  abi.ABI
	MetavaultPancakePair                     abi.ABI
	MetavaultChainlinkFlags                  abi.ABI
	MetavaultFastPriceFeedV1                 abi.ABI
	MetavaultFastPriceFeedV2                 abi.ABI
	MetavaultPriceFeed                       abi.ABI
	LidoWstETH                               abi.ABI
	FraxswapFactory                          abi.ABI
	FraxswapPair                             abi.ABI
	CamelotFactory                           abi.ABI
	CamelotPair                              abi.ABI
	OVMGasPriceOracle                        abi.ABI
	ArbGasInfo                               abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ERC20, erc20},
		{&ERC20DS, erc20ds},
		{&Multicall, multicall},
		{&CurveAddressProvider, curveAddressProvider},
		{&CurveAaveV1, curveAaveV1},
		{&CurveAave, curveAave},
		{&CurveCompound, curveCompound},
		{&CurveBase, curveBasepool},
		{&CurveBaseV1, curveBasepoolV1},
		{&CurvePlainOraclePool, curvePlainOraclePool},
		{&CurveOracle, curveOracle},
		{&CurveTricrypto, curveTricrypto},
		{&CurveMeta, curveMetapool},
		{&CurveMainRegistry, curveMainRegistry},
		{&CurveGetter, curveGetter},
		{&CurveMetaFactory, curveMetaPoolFactory},
		{&CurveCryptoRegistry, curveCryptoRegistry},
		{&CurveCryptoFactory, curveCryptoFactory},
		{&CurveTwo, curveTwo},
		{&SushiswapPair, sushiswapPair},
		{&QuickswapPair, quickswapPair},
		{&DfynPair, dfynPair},
		{&WaultPair, waultSwapPair},
		{&JetswapPair, jetswapPair},
		{&PolydexPair, polydexPair},
		{&DmmPool, dmmPool},
		{&DmmFactory, dmmFactory},
		{&PolycatPair, polycatPair},
		{&FirebirdPair, firebirdSwapPair},
		{&FirebirdOneSwap, firebirdOneSwap},
		{&FirebirdOneSwapFactory, firebirdOneSwapFactory},
		{&IronSwap, ironSwap},
		{&NerveSwap, nerveSwap},
		{&BiswapPair, biswapPair},
		{&BiswapFactory, biswapFactory},
		{&BalancerVault, balancerVault},
		{&BalancerPool, balancerPool},
		{&BalancerMetaStablePool, balancerMetaStablePool},
		{&AToken, aaveAtoken},
		{&SynapseSwap, synapseSwapFlashLoan},
		{&UniswapV3Pool, uniswapv3Pool},
		{&UniV3TickLens, uniV3TickLens},
		{&Saddle, saddleSwapFlashLoan},
		{&ProMMPool, prommPool},
		{&MetaAggregationRouterV2, metaAggregationRouterV2},
		{&DodoV1, dodoV1},
		{&DodoV2, dodoV2},
		{&VelodromePair, velodromePair},
		{&VelodromeFactory, velodromeFactory},
		{&PlatypusPool, platypusPool},
		{&PlatypusAsset, platypusAsset},
		{&StakedAvax, stakedAvax},
		{&GMXVault, gmxVault},
		{&GMXVaultPriceFeed, gmxVaultPriceFeed},
		{&GMXPancakePair, gmxPancakePair},
		{&GMXChainlinkFlags, gmxChainlinkFlags},
		{&GMXFastPriceFeedV1, gmxFastPriceFeedV1},
		{&GMXFastPriceFeedV2, gmxFastPriceFeedV2},
		{&GMXPriceFeed, gmxPriceFeed},
		{&MakerPSMPSM, makerPSMPSM},
		{&MakerPSMVat, makerPSMVat},
		{&Synthetix, synthetix},
		{&SynthetixSystemSettings, synthetixSystemSettings},
		{&SynthetixExchanger, synthetixExchanger},
		{&SynthetixExchangerWithFeeRecAlternatives, synthetixExchangerWithFeeRecAlternatives},
		{&SynthetixExchangeRates, synthetixExchangeRates},
		{&SynthetixExchangeRatesWithDexPricing, synthetixExchangeRatesWithDexPricing},
		{&SynthetixChainlinkDataFeed, synthetixChainlinkDataFeed},
		{&SynthetixDexPriceAggregatorUniswapV3, synthetixDexPriceAggregatorUniswapV3},
		{&SynthetixMultiCollateralSynth, synthetixMultiCollateralSynth},
		{&MetavaultVault, metavaultVault},
		{&MetavaultVaultPriceFeed, metavaultVaultPriceFeed},
		{&MetavaultPancakePair, metavaultPancakePair},
		{&MetavaultChainlinkFlags, metavaultChainlinkFlags},
		{&MetavaultFastPriceFeedV1, metavaultFastPriceFeedV1},
		{&MetavaultFastPriceFeedV2, metavaultFastPriceFeedV2},
		{&MetavaultPriceFeed, metavaultPriceFeed},
		{&LidoWstETH, lidoWstETH},
		{&FraxswapFactory, fraxswapFactory},
		{&FraxswapPair, fraxswapPair},
		{&CamelotFactory, camelotFactory},
		{&CamelotPair, camelotPair},
		{&OVMGasPriceOracle, ovmGasPriceOracle},
		{&ArbGasInfo, arbGasInfo},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}

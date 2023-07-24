package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexTypeSynthetix = "synthetix"

const (
	// Synthetix methods

	PoolStateMethodAvailableCurrencyKeys        = "availableCurrencyKeys"
	PoolStateMethodAvailableSynthCount          = "availableSynthCount"
	PoolStateMethodAvailableSynths              = "availableSynths"
	PoolStateMethodGetSUSDCurrencyKey           = "sUSD"
	PoolStateMethodGetSynthAddressByCurrencyKey = "synths"
	PoolStateMethodTotalIssuedSynths            = "totalIssuedSynths" // to get the "reserves" of tokens

	// MultiCollateralSynth methods

	MultiCollateralSynthMethodGetProxy = "proxy"

	// ProxyERC20 methods

	ProxyERC20MethodTotalSupply = "totalSupply"

	// SystemSettings methods

	SystemSettingsMethodPureChainlinkPriceForAtomicSwapsEnabled = "pureChainlinkPriceForAtomicSwapsEnabled"
	SystemSettingsMethodAtomicEquivalentForDexPricing           = "atomicEquivalentForDexPricing"
	SystemSettingsMethodAtomicTwapWindow                        = "atomicTwapWindow"
	SystemSettingsMethodAtomicVolatilityConsiderationWindow     = "atomicVolatilityConsiderationWindow"
	SystemSettingsMethodAtomicVolatilityUpdateThreshold         = "atomicVolatilityUpdateThreshold"
	SystemSettingsMethodAtomicExchangeFeeRate                   = "atomicExchangeFeeRate"
	SystemSettingsMethodExchangeFeeRate                         = "exchangeFeeRate"
	SystemSettingsMethodRateStalePeriod                         = "rateStalePeriod"
	SystemSettingsMethodExchangeDynamicFeeRounds                = "exchangeDynamicFeeRounds"
	SystemSettingsMethodExchangeDynamicFeeThreshold             = "exchangeDynamicFeeThreshold"
	SystemSettingsMethodExchangeDynamicFeeWeightDecay           = "exchangeDynamicFeeWeightDecay"
	SystemSettingsMethodExchangeMaxDynamicFee                   = "exchangeMaxDynamicFee"

	// Token methods

	TokenMethodDecimals = "decimals"
	TokenMethodSymbol   = "symbol"

	// ExchangerWithFeeRecAlternatives methods

	ExchangerWithFeeRecAlternativesMethodAtomicMaxVolumePerBlock = "atomicMaxVolumePerBlock"
	ExchangerWithFeeRecAlternativesMethodLastAtomicVolume        = "lastAtomicVolume"

	// ExchangeRates methods

	ExchangeRatesMethodAggregators         = "aggregators"
	ExchangeRatesMethodCurrencyKeyDecimals = "currencyKeyDecimals"
	ExchangeRatesMethodGetCurrentRoundId   = "getCurrentRoundId"

	// ExchangeRatesWithDexPricing methods

	ExchangeRatesWithDexPricingMethodAggregators                       = "aggregators"
	ExchangeRatesWithDexPricingMethodCurrencyKeyDecimals               = "currencyKeyDecimals"
	ExchangeRatesWithDexPricingMethodDexPriceAggregator                = "dexPriceAggregator"
	ExchangeRatesWithDexPricingMethodGetCurrentRoundId                 = "getCurrentRoundId"
	ExchangeRatesWithDexPricingMethodSynthTooVolatileForAtomicExchange = "synthTooVolatileForAtomicExchange"

	// ChainlinkDataFeed methods

	ChainlinkDataFeedMethodLatestRoundData = "latestRoundData"
	ChainlinkDataFeedMethodGetRoundData    = "getRoundData"

	// DexPriceAggregatorUniswapV3 methods

	DexPriceAggregatorUniswapV3MethodDefaultPoolFee         = "defaultPoolFee"
	DexPriceAggregatorUniswapV3MethodUniswapV3Factory       = "uniswapV3Factory"
	DexPriceAggregatorUniswapV3MethodWeth                   = "weth"
	DexPriceAggregatorUniswapV3MethodOverriddenPoolForRoute = "overriddenPoolForRoute"

	// UniswapV3 Pool methods

	UniswapV3PoolMethodSlot0        = "slot0"
	UniswapV3PoolMethodObserve      = "observe"
	UniswapV3PoolMethodObservations = "observations"
)

type PoolStateVersion uint

const (
	PoolStateVersionNormal PoolStateVersion = 1
	PoolStateVersionAtomic PoolStateVersion = 2
	PoolInitCodeHash                        = "0xe34f199b19b2b4f47f68442619d555527d244f78a3297ea89325f843f87b8b54"
)

var (
	DefaultPoolStateVersion   = PoolStateVersionAtomic
	DefaultChainlinkNumRounds = big.NewInt(5)
	DefaultGas                = Gas{ExchangeAtomically: 600000, Exchange: 130000}
	PoolStateVersionByChainID = map[valueobject.ChainID]PoolStateVersion{
		valueobject.ChainIDEthereum: PoolStateVersionAtomic,
		valueobject.ChainIDOptimism: PoolStateVersionNormal,
	}
)

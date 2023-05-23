package synthetix

import _ "embed"

//go:embed abis/Synthetix.json
var synthetixBytes []byte

//go:embed abis/SystemSettings.json
var systemSettingsBytes []byte

//go:embed abis/Exchanger.json
var exchangerBytes []byte

//go:embed abis/ExchangerWithFeeRecAlternatives.json
var exchangerWithFeeRecAlternativesBytes []byte

//go:embed abis/ExchangeRates.json
var exchangeRatesBytes []byte

//go:embed abis/ExchangeRatesWithDexPricing.json
var exchangeRatesWithDexPricingBytes []byte

//go:embed abis/ChainlinkDataFeed.json
var chainlinkDataFeedBytes []byte

//go:embed abis/DexPriceAggregatorUniswapV3.json
var dexPriceAggregatorUniswapV3Bytes []byte

//go:embed abis/MultiCollateralSynth.json
var multiCollateralSynthBytes []byte

//go:embed abis/ERC20.json
var erc20Bytes []byte

//go:embed abis/UniswapV3Pool.json
var uniswapv3PoolBytes []byte

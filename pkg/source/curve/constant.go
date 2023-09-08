package curve

import "math/big"

const (
	DexTypeCurve = "curve"

	addressProviderMethodGetAddress           = "get_address"
	registryOrFactoryMethodPoolList           = "pool_list"
	registryOrFactoryMethodPoolCount          = "pool_count"
	registryOrFactoryMethodGetCoins           = "get_coins"
	registryOrFactoryMethodGetUnderlyingCoins = "get_underlying_coins"
	registryOrFactoryMethodIsMeta             = "is_meta"
	registryOrFactoryMethodGetBasePool        = "get_base_pool"
	registryOrFactoryMethodGetDecimals        = "get_decimals"
	registryOrFactoryMethodGetUnderDecimals   = "get_underlying_decimals"
	registryOrFactoryMethodGetRates           = "get_rates"
	registryOrFactoryMethodGetLpToken         = "get_lp_token"

	poolMethodA                   = "A"
	poolMethodAPrecise            = "A_precise"
	poolMethodToken               = "token"
	poolMethodBalances            = "balances"
	poolMethodInitialA            = "initial_A"
	poolMethodInitialATime        = "initial_A_time"
	poolMethodFutureA             = "future_A"
	poolMethodFutureATime         = "future_A_time"
	poolMethodFee                 = "fee"
	poolMethodAdminFee            = "admin_fee"
	poolMethodD                   = "D"
	poolMethodGamma               = "gamma"
	poolMethodFeeGamma            = "fee_gamma"
	poolMethodMidFee              = "mid_fee"
	poolMethodOutFee              = "out_fee"
	poolMethodFutureAGammaTime    = "future_A_gamma_time"
	poolMethodFutureAGamma        = "future_A_gamma"
	poolMethodInitialAGammaTime   = "initial_A_gamma_time"
	poolMethodInitialAGamma       = "initial_A_gamma"
	poolMethodLastPricesTimestamp = "last_prices_timestamp"
	poolMethodXcpProfit           = "xcp_profit"
	poolMethodVirtualPrice        = "virtual_price"
	poolMethodAllowedExtraProfit  = "allowed_extra_profit"
	poolMethodAdjustmentStep      = "adjustment_step"
	poolMethodMaHalfTime          = "ma_half_time"
	poolMethodPriceScale          = "price_scale"
	poolMethodPriceOracle         = "price_oracle"
	poolMethodLastPrices          = "last_prices"
	poolMethodBasePool            = "base_pool"

	aaveMethodOffpegFeeMultiplier = "offpeg_fee_multiplier"
	oracleMethodLatestAnswer      = "latestAnswer"
	plainOracleMethodOracle       = "oracle"

	erc20MethodName        = "name"
	erc20MethodTotalSupply = "totalSupply"

	addressEther = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	addressZero  = "0x0000000000000000000000000000000000000000"

	zeroString    = "0"
	defaultWeight = 1
)

const (
	poolTypeBase        = "curve-base"
	poolTypePlainOracle = "curve-plain-oracle"
	poolTypeMeta        = "curve-meta"
	poolTypeLending     = "curve-lending"
	poolTypeAave        = "curve-aave"
	poolTypeCompound    = "curve-compound"
	poolTypeTricrypto   = "curve-tricrypto"
	poolTypeTwo         = "curve-two"
	poolTypeUnsupported = "unsupported"
)

// Curve pool types
const (
	sourceMainRegistry = iota
	sourceMetaPoolsFactory
	sourceCryptoPoolsRegistry
	sourceCryptoPoolsFactory
)

// Known weth9 implementation addresses, used in our implementation of Ether#wrapped
var weth9 = map[int]string{
	1:          "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
	10001:      "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
	3:          "0xc778417E063141139Fce010982780140Aa0cD5Ab",
	4:          "0xc778417E063141139Fce010982780140Aa0cD5Ab",
	5:          "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
	42:         "0xd0A1E359811322d97991E03f863a0C30C2cF029C",
	56:         "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
	10:         "0x4200000000000000000000000000000000000006",
	69:         "0x4200000000000000000000000000000000000006",
	137:        "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
	80001:      "0x19395624C030A11f58e820C3AeFb1f5960d9742a",
	43114:      "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
	43113:      "0x1D308089a2D1Ced3f1Ce36B1FcaF815b07217be3",
	250:        "0x21be370D5312f44cB42ce377BC9b8a0cEF1A4C83",
	25:         "0x5C7F8A570d578ED84E63fdFA7b1eE72dEae1AE23",
	199:        "0x8D193c6efa90BCFf940A98785d1Ce9D093d3DC8A",
	106:        "0xc579D1f3CF86749E05CD06f7ADe17856c2CE3126",
	1313161554: "0xC9BdeEd33CD01541e1eeD10f90519d2C06Fe3feB",
	42262:      "0x21C718C22D52d0F3a789b752D4c2fD5908a8A733",
	42161:      "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
	421611:     "0xB47e6A5f8b33b3F17603C83a0535A9dcD7E32681",
}

var (
	zeroBI            = big.NewInt(0)
	emptyString       = ""
	zero        int64 = 0
)

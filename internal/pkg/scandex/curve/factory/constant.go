package factory

const (
	AddressProvider = "0x0000000022D53366457F9d5E68Ec105046FC4383"
	AddressZero     = "0x0000000000000000000000000000000000000000"
	AddressEther    = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	ReserveZero     = "0"

	MainRegistryMethodGetLPToken = "get_lp_token"
	MainRegistryMethodGetRates   = "get_rates"
	PoolGetterMethodGetPoolCoins = "get_pool_coins"

	PlainOraclePoolMethodOracle       = "oracle"
	AavePoolMethodOffpegFeeMultiplier = "offpeg_fee_multiplier"
)

var IgnoreDexes = []string{"ellipsis", "rose"}

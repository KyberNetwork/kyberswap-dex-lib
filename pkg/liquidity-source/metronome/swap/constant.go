package metronomeswap

const DexType = "metronome-swap"

const (
	poolMethodIsSwapActive      = "isSwapActive"
	poolMethodGetDebtTokens     = "getDebtTokens"
	poolMethodFeeProvider       = "feeProvider"
	poolMethodMasterOracle      = "masterOracle"
	poolMethodPaused            = "paused"
	poolMethodEverythingStopped = "everythingStopped"

	poolRegistryMethodGetPools          = "getPools"
	poolRegistryMethodPaused            = "paused"
	poolRegistryMethodEverythingStopped = "everythingStopped"

	feeProviderMethodSwapFees = "swapFees"

	masterOracleMethodGetPriceInUsd = "getPriceInUsd"

	debtTokenMethodSyntheticToken = "syntheticToken"

	syntheticTokenMethodIsActive       = "isActive"
	syntheticTokenMethodMaxTotalSupply = "maxTotalSupply"
	syntheticTokenMethodTotalSupply    = "totalSupply"
	syntheticTokenMethodDecimals       = "decimals"
)

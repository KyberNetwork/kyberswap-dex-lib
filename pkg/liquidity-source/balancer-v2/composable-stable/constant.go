package composablestable

const (
	DexType = "balancer-v2-composable-stable"
)

var (
	DefaultGas = Gas{Swap: 80000}
)

const (
	poolTypeComposableStable = "ComposableStable"

	poolTypeVer1 = 1
	poolTypeVer5 = 5

	poolMethodGetSwapFeePercentage              = "getSwapFeePercentage"
	poolMethodGetPausedState                    = "getPausedState"
	poolMethodGetAmplificationParameter         = "getAmplificationParameter"
	poolMethodGetBptIndex                       = "getBptIndex"
	poolMethodTotalSupply                       = "totalSupply"
	poolMethodGetLastJoinExitData               = "getLastJoinExitData"
	poolMethodGetRateProviders                  = "getRateProviders"
	poolMethodGetTokenRateCache                 = "getTokenRateCache"
	poolMethodGetProtocolFeePercentageCache     = "getProtocolFeePercentageCache"
	poolMethodIsTokenExemptFromYieldProtocolFee = "isTokenExemptFromYieldProtocolFee"
	poolMethodInRecoveryMode                    = "inRecoveryMode"
	poolMethodIsExemptFromYieldProtocolFee      = "isExemptFromYieldProtocolFee"
	poolMethodGetRate                           = "getRate"
	poolMethodGetVault                          = "getVault"

	feeTypeSwap  = 0
	feeTypeYield = 2

	zeroAddress = "0x0000000000000000000000000000000000000000"
)

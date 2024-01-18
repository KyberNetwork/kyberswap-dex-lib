package gyroeclp

const (
	DexType = "gyroscope-eclp"

	poolType = "GyroE"

	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"
	poolMethodGetTokenRates        = "getTokenRates"
	poolMethodGetECLPParams        = "getECLPParams"

	defaultWeight = 1
)

var (
	defaultGas = Gas{Swap: 80000} // TODO: rebenchmark gas
)

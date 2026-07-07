package gyroeclp

const (
	DexType = "gyroscope-eclp"

	poolType = "GyroE"

	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"
	poolMethodGetTokenRates        = "getTokenRates"
	poolMethodGetECLPParams        = "getECLPParams"

	poolTypeVer1 = 1
)

var (
	defaultGas = Gas{Swap: 135726}
)

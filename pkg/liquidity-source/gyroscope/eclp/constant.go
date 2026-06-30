package gyroeclp

const (
	DexType = "gyroscope-eclp"

	poolType = "GyroE"

	PoolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	PoolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"
	PoolMethodGetTokenRates        = "getTokenRates"
	PoolMethodGetECLPParams        = "getECLPParams"

	PoolTypeVer1 = 1
)

var (
	defaultGas = Gas{Swap: 135726}
)

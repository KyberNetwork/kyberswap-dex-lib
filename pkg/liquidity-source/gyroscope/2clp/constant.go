package gyro2clp

const (
	DexType = "gyroscope-2clp"

	poolType = "Gyro2"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"
	poolMethodGetSqrtParameters    = "getSqrtParameters"

	defaultWeight = 1
)

var defaultGas = Gas{Swap: 100000} // TODO: benchmark

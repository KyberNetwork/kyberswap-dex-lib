package gyro3clp

const (
	DexType = "gyroscope-3clp"

	poolType = "Gyro3"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"

	defaultWeight = 1
)

var defaultGas = Gas{Swap: 100000} // TODO: benchmark

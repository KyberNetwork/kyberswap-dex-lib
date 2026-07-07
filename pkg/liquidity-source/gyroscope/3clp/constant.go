package gyro3clp

const (
	DexType = "gyroscope-3clp"

	poolType = "Gyro3"
)

const (
	poolMethodGetSwapFeePercentage = "getSwapFeePercentage"
	poolMethodGetPausedState       = "getPausedState"
	poolMethodGetVault             = "getVault"
	poolMethodGetRoot3Alpha        = "getRoot3Alpha"
)

var defaultGas = Gas{Swap: 125660}

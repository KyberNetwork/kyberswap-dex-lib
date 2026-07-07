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
)

var defaultGas = Gas{Swap: 126379}

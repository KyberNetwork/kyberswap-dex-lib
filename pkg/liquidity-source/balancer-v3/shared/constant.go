package shared

type (
	Rounding int
	SwapKind int
)

const (
	RoundUp Rounding = iota
	RoundDown
)

const (
	ExactIn SwapKind = iota
	ExactOut
)

const (
	VaultMethodGetBufferAsset             = "getBufferAsset"
	VaultMethodGetHooksConfig             = "getHooksConfig"
	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolData                = "getPoolData"

	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"

	ERC4626MethodConvertToAssets = "convertToAssets"
)

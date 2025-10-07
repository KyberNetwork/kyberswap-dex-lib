package shared

import "github.com/ethereum/go-ethereum/common"

type (
	Rounding int
	SwapKind int
)

var (
	VaultAddress = common.HexToAddress("0xbA1333333333a1BA1108E8412f11850A5C319bA9")
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
	RelistInterval = 60 // relist every 60 times

	VaultMethodGetBufferAsset             = "getBufferAsset"
	VaultMethodGetHooksConfig             = "getHooksConfig"
	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolData                = "getPoolData"

	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"

	ERC4626MethodConvertToAssets = "convertToAssets"
	ERC4626MethodConvertToShares = "convertToShares"
	ERC4626MethodMaxDeposit      = "maxDeposit"
	ERC4626MethodMaxRedeem       = "maxRedeem"
)

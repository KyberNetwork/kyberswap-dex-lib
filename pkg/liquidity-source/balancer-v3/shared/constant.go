package shared

import (
	"github.com/holiman/uint256"
)

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
	VaultMethodGetHooksConfig             = "getHooksConfig"
	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolData                = "getPoolData"

	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"

	ERC4626MethodTotalAssets = "totalAssets"
	ERC4626MethodTotalSupply = "totalSupply"
)

var (
	DecimalsOffsetPow = uint256.NewInt(1e3) // some buffer has this as 0, but 1e3 is a good blanket value
)

package shared

import (
	"github.com/holiman/uint256"
)

const (
	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolData                = "getPoolData"
	VaultMethodGetHooksConfig             = "getHooksConfig"

	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"

	ERC4626MethodTotalAssets = "totalAssets"
	ERC4626MethodTotalSupply = "totalSupply"
)

type (
	Rounding  int
	TokenType uint8
	SwapKind  int
)

const (
	ROUND_UP Rounding = iota
	ROUND_DOWN

	EXACT_IN SwapKind = iota
	EXACT_OUT
)

var (
	MINIMUM_TRADE_AMOUNT = uint256.NewInt(1000000) // to be more general, this value should be queried from the VaultAdmin contract

	DecimalsOffsetPow = uint256.NewInt(1e3) // some buffer has this as 0, but 1e3 is a good value
)

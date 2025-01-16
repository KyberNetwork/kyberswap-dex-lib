package shared

import "github.com/holiman/uint256"

const (
	PoolMethodGetAmplificationParameter = "getAmplificationParameter"
	PoolMethodVersion                   = "version"

	VaultMethodGetStaticSwapFeePercentage = "getStaticSwapFeePercentage"
	VaultMethodGetAggregateFeePercentages = "getAggregateFeePercentages"
	VaultMethodGetPoolTokenRates          = "getPoolTokenRates"
	VaultMethodGetPoolData                = "getPoolData"
	VaultMethodGetHooksConfig             = "getHooksConfig"

	VaultMethodIsPoolPaused         = "isPoolPaused"
	VaultMethodIsVaultPaused        = "isVaultPaused"
	VaultMethodIsPoolInRecoveryMode = "isPoolInRecoveryMode"
)

type (
	Rounding  int
	TokenType uint8
	SwapKind  int
)

const (
	ROUND_UP Rounding = iota
	ROUND_DOWN

	STANDARD TokenType = iota
	WITH_RATE

	EXACT_IN SwapKind = iota
	EXACT_OUT
)

var (
	MINIMUM_TRADE_AMOUNT = uint256.NewInt(1000000) // to be more general, this value should be queried from the VaultAdmin contract
)

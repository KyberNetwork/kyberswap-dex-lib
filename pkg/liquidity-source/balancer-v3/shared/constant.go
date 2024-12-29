package shared

import "github.com/holiman/uint256"

const (
	VaultMethodGetPoolTokens            = "getPoolTokens"
	PoolMethodGetSwapFeePercentage      = "getSwapFeePercentage"
	PoolMethodGetPausedState            = "getPausedState"
	PoolMethodGetAmplificationParameter = "getAmplificationParameter"
	PoolMethodGetVault                  = "getVault"
	PoolMethodVersion                   = "version"

	PoolVersion1 = 1
	PoolVersion2 = 2
)

type Rounding int

const (
	ROUND_UP Rounding = iota
	ROUND_DOWN
)

var (
	MINIMUM_TRADE_AMOUNT = uint256.NewInt(1000000) // to be more general, this value should be queried from the VaultAdmin contract
)

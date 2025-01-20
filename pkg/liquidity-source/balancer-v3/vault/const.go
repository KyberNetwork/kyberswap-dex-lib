package vault

import (
	"github.com/holiman/uint256"
)

var (
	// to be more general, this value should be queried from the VaultAdmin contract
	MINIMUM_TRADE_AMOUNT = uint256.NewInt(1000000)

	MAX_FEE_PERCENTAGE, _ = uint256.FromDecimal("999999000000000000") // 99.9999e16; // 99.9999%
)

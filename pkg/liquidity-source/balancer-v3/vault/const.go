package vault

import (
	"github.com/holiman/uint256"
)

var (
	// MinimumTradeAmount to be more general, this value should be queried from the VaultAdmin contract
	MinimumTradeAmount = uint256.NewInt(1000000)

	MaxFeePercentage, _ = uint256.FromDecimal("999999000000000000") // 99.9999e16; // 99.9999%

	Type = "balancer-vault-v3"
)

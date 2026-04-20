package aavev3

import (
	"math/big"
)

func parseConfiguration(configuration *big.Int) Extra {
	// Bit 56: reserve is active
	isActive := configuration.Bit(56) == 1

	// Bit 57: reserve is frozen
	isFrozen := configuration.Bit(57) == 1

	// Bit 60: asset is paused
	isPaused := configuration.Bit(60) == 1

	return Extra{
		IsActive: isActive,
		IsFrozen: isFrozen,
		IsPaused: isPaused,
	}
}

func parseSupplyCap(configuration *big.Int) uint64 {
	// Bits 116-151: supply cap (36 bits)
	// Supply cap is in whole tokens (not scaled)
	// supplyCap == 0 means no cap
	var supplyCap uint64
	for i := 116; i <= 151; i++ {
		if configuration.Bit(i) == 1 {
			supplyCap |= 1 << (i - 116)
		}
	}
	return supplyCap
}

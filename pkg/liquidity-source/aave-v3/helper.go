package aavev3

import (
	"math/big"
	"strings"
)

func getPoolID(token0, token1 string) string {
	return strings.Join([]string{DexType, token0, token1}, "_")
}

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

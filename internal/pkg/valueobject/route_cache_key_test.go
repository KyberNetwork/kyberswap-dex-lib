package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteCacheKey_String(t *testing.T) {
	t.Run("it should return correct key", func(t *testing.T) {
		key := RouteCacheKey{
			TokenIn:                "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:               "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:                true,
			CacheMode:              RouteCacheModePoint,
			AmountIn:               "5000000000000000000000",
			Dexes:                  []string{"gmx", "uniswap"},
			GasInclude:             true,
			IsPathGeneratorEnabled: false,
			IsHillClimbingEnabled:  false,
			ExcludedPools:          []string{"0x"},
		}

		assert.Equal(t, key.String("prefix"), "prefix:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1:0:0:0x")
	})
}

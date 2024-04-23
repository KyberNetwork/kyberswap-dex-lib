package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheConfig_Equals(t *testing.T) {
	want := CacheConfig{
		DefaultTTL: 1,
		TTLByAmount: []CachePoint{
			{
				Amount: 0,
				TTL:    0,
			},
		},
		TTLByAmountUSDRange: []CacheRange{
			{
				12, 125,
			},
		},
		PriceImpactThreshold: 2,
		ShrinkFuncName:       "abc",
		ShrinkFuncPowExp:     2,
		ShrinkFuncLogPercent: 2,
		ShrinkAmountInConfig: ShrinkFunctionConfig{
			ShrinkFuncName:       "xyz",
			ShrinkFuncPowExp:     4,
			ShrinkFuncLogPercent: 4,
		},
		ShrinkAmountInThreshold: 60,
	}

	og := CacheConfig{
		DefaultTTL: 1,
		TTLByAmount: []CachePoint{
			{
				Amount: 0,
				TTL:    0,
			},
		},
		TTLByAmountUSDRange: []CacheRange{
			{
				12, 125,
			},
		},
		PriceImpactThreshold: 2,
		ShrinkFuncName:       "abc",
		ShrinkFuncPowExp:     2,
		ShrinkFuncLogPercent: 2,
		ShrinkAmountInConfig: ShrinkFunctionConfig{
			ShrinkFuncName:       "xyz",
			ShrinkFuncPowExp:     4,
			ShrinkFuncLogPercent: 4,
		},
		ShrinkAmountInThreshold: 60,
	}
	assert.Equal(t, want.Equals(og), true)
	og.TTLByAmountUSDRange[0].TTL = 123
	assert.Equal(t, want.Equals(og), false)

}

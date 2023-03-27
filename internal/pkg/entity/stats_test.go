package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStats_Encode(t *testing.T) {
	t.Parallel()

	t.Run("it should encode stats correctly", func(t *testing.T) {
		stats := Stats{
			Pools: map[string]PoolStatsItem{
				"item1": {
					Size:   11,
					Tvl:    11111,
					Tokens: 111,
				},
			},
			TotalTokens: 1,
			TotalPools:  1,
		}

		statsStr, err := stats.Encode()

		assert.Nil(t, err)
		assert.Equal(t, "{\"Pools\":{\"item1\":{\"poolSize\":11,\"tvl\":11111,\"tokenSize\":111}},\"TotalPools\":1,\"TotalTokens\":1}", statsStr)
	})
}

func TestStats_DecodeStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		member        string
		expectedStats Stats
	}{
		{
			name:   "it should decode stats correctly with full data",
			member: "{\"Pools\":{\"item1\":{\"poolSize\":11,\"tvl\":11111,\"tokenSize\":111}},\"TotalPools\":1,\"TotalTokens\":1}",
			expectedStats: Stats{
				Pools: map[string]PoolStatsItem{
					"item1": {
						Size:   11,
						Tvl:    11111,
						Tokens: 111,
					},
				},
				TotalTokens: 1,
				TotalPools:  1,
			},
		},
		{
			name:   "it should decode stats correctly without total pools",
			member: "{\"Pools\":{\"item1\":{\"poolSize\":11,\"tvl\":11111,\"tokenSize\":111}},\"TotalTokens\":1}",
			expectedStats: Stats{
				Pools: map[string]PoolStatsItem{
					"item1": {
						Size:   11,
						Tvl:    11111,
						Tokens: 111,
					},
				},
				TotalTokens: 1,
				TotalPools:  0,
			},
		},
		{
			name:   "it should decode stats correctly without total tokens",
			member: "{\"Pools\":{\"item1\":{\"poolSize\":11,\"tvl\":11111,\"tokenSize\":111}},\"TotalPools\":1}",
			expectedStats: Stats{
				Pools: map[string]PoolStatsItem{
					"item1": {
						Size:   11,
						Tvl:    11111,
						Tokens: 111,
					},
				},
				TotalTokens: 0,
				TotalPools:  1,
			},
		},
		{
			name:   "it should decode stats correctly without pools",
			member: "{\"TotalPools\":1,\"TotalTokens\":1}",
			expectedStats: Stats{
				Pools:       nil,
				TotalTokens: 1,
				TotalPools:  1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats, err := DecodeStats(test.member)

			assert.Nil(t, err)
			assert.Equal(t, test.expectedStats, stats)
		})
	}
}

package getroute

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
)

func TestPoolFilterSources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		sources        []string
		pool           entity.Pool
		expectedResult bool
	}{
		{
			name:    "it should return true  when pool exchange is in sources",
			sources: []string{"uniswap"},
			pool: entity.Pool{
				Exchange: "uniswap",
			},
			expectedResult: true,
		},
		{
			name:    "it should return false when pool exchange is not in sources",
			sources: []string{"uniswap"},
			pool: entity.Pool{
				Exchange: "dodo",
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PoolFilterSources(tc.sources)(tc.pool)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestPoolFilterHasReserveOrAmplifiedTvl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		pool           entity.Pool
		expectedResult bool
	}{
		{
			name: "it should return false when pool neither has reserve or amplifiedTvl",
			pool: entity.Pool{
				Reserves:     []string{"0", "0"},
				AmplifiedTvl: 0,
			},
			expectedResult: false,
		},
		{
			name: "it should return true when pool has reserve",
			pool: entity.Pool{
				Reserves:     []string{"1", "1"},
				AmplifiedTvl: 0,
			},
			expectedResult: true,
		},
		{
			name: "it should return true when pool has reserve",
			pool: entity.Pool{
				Reserves:     []string{"0", "0"},
				AmplifiedTvl: 1,
			},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PoolFilterHasReserveOrAmplifiedTvl(tc.pool)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

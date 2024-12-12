package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_GetLpToken(t *testing.T) {
	tests := []struct {
		name           string
		pool           Pool
		expectedResult string
	}{
		{
			name: "it should return pool's address when static extra has no LpToken",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"111111", "222222"},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: "address1",
		},
		{
			name: "it should return LpToken inside static extra in lowercase when it exists",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "{\"lpToken\":\"LpToken_inside_StaticExtra\"}",
				TotalSupply: "totalSupply1",
			},
			expectedResult: "lptoken_inside_staticextra",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hasReserves := test.pool.GetLpToken()

			assert.Equal(t, test.expectedResult, hasReserves)
		})
	}
}

func TestPool_HasReserves(t *testing.T) {
	tests := []struct {
		name           string
		pool           Pool
		expectedResult bool
	}{
		{
			name: "it should return true when pool's reserves are correct",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"111111", "222222"},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: true,
		},
		{
			name: "it should return false when reserves slice is empty",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: false,
		},
		{
			name: "it should return true when at least one reserve is not empty",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"", "222222"},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: true,
		},
		{
			name: "it should return true when at least one reserve is not 0",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"0", "222222"},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hasReserves := test.pool.HasReserves()

			assert.Equal(t, test.expectedResult, hasReserves)
		})
	}
}

func TestPool_HasAmplifiedTvl(t *testing.T) {
	tests := []struct {
		name           string
		pool           Pool
		expectedResult bool
	}{
		{
			name: "it should return true when pool's AmplifiedTvl > 0",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"111111", "222222"},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: true,
		},
		{
			name: "it should return false when pool's AmplifiedTvl <= 0",
			pool: Pool{
				Address:      "address1",
				ReserveUsd:   0,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{},
				Tokens: []*PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hasReserves := test.pool.HasReserves()

			assert.Equal(t, test.expectedResult, hasReserves)
		})
	}
}

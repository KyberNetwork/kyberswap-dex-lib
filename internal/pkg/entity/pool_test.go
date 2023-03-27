package entity

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_Encode(t *testing.T) {
	t.Parallel()

	t.Run("it should encode pool correctly", func(t *testing.T) {
		pool := Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1", "reserve2"},
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
		}

		poolStr, err := pool.Encode()

		assert.Nil(t, err)
		assert.Equal(t, "{\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"name\":\"poolTokenName1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"name\":\"poolTokenName2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"extra1\",\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}", poolStr)
	})
}

func TestDecodePool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		key          string
		member       string
		expectedPool Pool
	}{
		{
			name:   "it should decode pool correctly with full data",
			key:    "address1",
			member: "{\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"name\":\"poolTokenName1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"name\":\"poolTokenName2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"extra1\",\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
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
		},
		{
			name:   "it should decode pool correctly without pool tokens",
			key:    "address1",
			member: "{\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
				Tokens:       nil,
				Extra:        "",
				StaticExtra:  "staticExtra1",
				TotalSupply:  "totalSupply1",
			},
		},
		{
			name:   "it should decode pool correctly without extra",
			key:    "address1",
			member: "{\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"name\":\"poolTokenName1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"name\":\"poolTokenName2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
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
				Extra:       "",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
		},
		{
			name:   "it should decode pool correctly without staticExtra",
			key:    "address1",
			member: "{\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"name\":\"poolTokenName1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"name\":\"poolTokenName2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"extra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
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
				StaticExtra: "",
				TotalSupply: "totalSupply1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool, err := DecodePool(test.key, test.member)

			assert.Nil(t, err)
			assert.Equal(t, test.expectedPool, pool)
		})
	}
}

func TestPool_GetLpToken(t *testing.T) {
	t.Parallel()

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

	var staticExtra = struct {
		LpToken string `json:"lpToken"`
	}{
		LpToken: "LpToken_inside_StaticExtra",
	}

	a, _ := json.Marshal(staticExtra)
	fmt.Println("======", string(a))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hasReserves := test.pool.GetLpToken()

			assert.Equal(t, test.expectedResult, hasReserves)
		})
	}
}

func TestPool_HasReserves(t *testing.T) {
	t.Parallel()

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
			name: "it should return false when at least one reserve is empty string",
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
			expectedResult: false,
		},
		{
			name: "it should return false when at least one reserve is 0",
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

func TestPool_HasAmplifiedTvl(t *testing.T) {
	t.Parallel()

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

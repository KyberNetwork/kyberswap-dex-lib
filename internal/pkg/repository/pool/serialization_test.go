package pool

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func TestSerialization_EncodePool(t *testing.T) {
	t.Run("it should encode pool correctly", func(t *testing.T) {
		pool := entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1", "reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		poolStr, err := encodePool(pool)

		assert.Nil(t, err)
		assert.Equal(t, "{\"address\":\"address1\",\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"swappable\":true}],\"extra\":\"extra1\",\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}", poolStr)
	})
}

func TestSerialization_DecodePool(t *testing.T) {
	type testInput struct {
		name         string
		key          string
		member       string
		expectedPool *entity.Pool
	}
	tests := []testInput{
		{
			name:   "it should decode pool correctly with full data",
			key:    "address1",
			member: "{\"address\":\"address1\",\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"swappable\":true}],\"extra\":\"extra1\",\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: &entity.Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
		},
		{
			name:   "it should decode pool correctly without extra",
			key:    "address1",
			member: "{\"address\":\"address1\",\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"swappable\":true}],\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: &entity.Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
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
			member: "{\"address\":\"address1\",\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"tokens\":[{\"address\":\"poolTokenAddress1\",\"symbol\":\"poolTokenSymbol1\",\"decimals\":18,\"swappable\":true},{\"address\":\"poolTokenAddress2\",\"symbol\":\"poolTokenSymbol2\",\"decimals\":18,\"swappable\":true}],\"extra\":\"extra1\",\"totalSupply\":\"totalSupply1\"}",
			expectedPool: &entity.Pool{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1", "reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
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
			pool, err := decodePool(test.key, test.member)

			assert.Nil(t, err)
			assert.Equal(t, test.expectedPool, pool)
		})
	}

	// edge test
	edgeTest := testInput{
		name:   "it should decode pool correctly without pool tokens",
		key:    "address1",
		member: "{\"address\":\"address1\",\"reserveUsd\":100,\"amplifiedTvl\":100,\"swapFee\":0.3,\"type\":\"uni\",\"timestamp\":12345,\"reserves\":[\"reserve1\",\"reserve2\"],\"staticExtra\":\"staticExtra1\",\"totalSupply\":\"totalSupply1\"}",
		expectedPool: &entity.Pool{
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
	}
	t.Run(edgeTest.name, func(t *testing.T) {
		pool, err := decodePool(edgeTest.key, edgeTest.member)
		assert.Nil(t, err)
		// in case of that pool is derived from mempool, then tokens has it's len == 0 && cap > 0 => assign to nil
		//t.Log(len(pool.Tokens), cap(pool.Tokens), edgeTest.expectedPool.Tokens)
		if len(pool.Tokens) == 0 && cap(pool.Tokens) > 0 {
			pool.Tokens = nil
		}
		assert.Equal(t, edgeTest.expectedPool, pool)
	})
}

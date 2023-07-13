package pancakev3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestPool_CalcAmountOut(t *testing.T) {
	test1AmountInBI, _ := new(big.Int).SetString("399888202451311482718477", 10)
	test2AmountInBI, _ := new(big.Int).SetString("399888202451311482718477", 10)

	tests := []struct {
		name            string
		entityPool      entity.Pool
		chainId         valueobject.ChainID
		tokenAmountIn   pool.TokenAmount
		tokenOut        string
		expectAmountOut *big.Int
		expectedErr     error
	}{
		{
			name: "it should return correct amount out OPENAI -> WBNB via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountIn: pool.TokenAmount{
				Token:     "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
				Amount:    test1AmountInBI,
				AmountUsd: 0,
			},
			tokenOut:        "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
			expectAmountOut: big.NewInt(10997482374483461),
			expectedErr:     nil,
		},
		{
			name: "it should return correct amount out WBNB -> OPENAI via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountIn: pool.TokenAmount{
				Token:     "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
				Amount:    test2AmountInBI,
				AmountUsd: 0,
			},
			tokenOut:        "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
			expectAmountOut: big.NewInt(90929739),
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewPool(tt.entityPool, tt.chainId)
			calcAmountOutResult, err := p.CalcAmountOut(tt.tokenAmountIn, tt.tokenOut)

			assert.Equal(t, tt.expectAmountOut, calcAmountOutResult.TokenAmountOut.Amount)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

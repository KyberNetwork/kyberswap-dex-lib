package kyberpmm

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_getAmountOut(t *testing.T) {
	type args struct {
		amountIn    *big.Float
		priceLevels []PriceLevel
	}
	tests := []struct {
		name              string
		args              args
		expectedAmountOut *big.Float
		expectedErr       error
	}{
		{
			name: "it should return error when price levels is empty",
			args: args{
				amountIn:    new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{},
			},
			expectedAmountOut: nil,
			expectedErr:       ErrEmptyPriceLevels,
		},
		{
			name: "it should return insufficient liquidity error when the requested amount is greater than available amount in price levels",
			args: args{
				amountIn: new(big.Float).SetFloat64(4),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       ErrInsufficientLiquidity,
		},
		{
			name: "it should return correct amount out when fully filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedAmountOut: new(big.Float).SetFloat64(100),
			expectedErr:       nil,
		},
		{
			name: "it should return correct amount out when partially filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedAmountOut: new(big.Float).SetFloat64(199),
			expectedErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amountOut, err := testutil.MustConcurrentSafe[*big.Float](t, func() (any, error) {
				return getAmountOut(tt.args.amountIn, tt.args.priceLevels)
			})
			assert.Equal(t, tt.expectedErr, err)

			if amountOut != nil {
				assert.Equal(t, tt.expectedAmountOut.Cmp(amountOut), 0)
			}
		})
	}
}

func TestPoolSimulator_getNewPriceLevelsState(t *testing.T) {
	type args struct {
		amountIn    *big.Float
		priceLevels []PriceLevel
	}
	tests := []struct {
		name                string
		args                args
		expectedPriceLevels []PriceLevel
	}{
		{
			name: "it should do nothing when price levels is empty",
			args: args{
				amountIn:    new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when fully filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when the amountIn is greater than the amount available in the single price level",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when the amountIn is greater than the amount available in the all price levels",
			args: args{
				amountIn: new(big.Float).SetFloat64(5),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when partially filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{
				{
					Price:  99,
					Amount: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newPriceLevels := getNewPriceLevelsState(tt.args.amountIn, tt.args.priceLevels)

			assert.ElementsMatch(t, tt.expectedPriceLevels, newPriceLevels)
		})
	}
}

func TestPoolSimulator_swapLimit(t *testing.T) {
	ps, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{
			{
				Address:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202", // KNC
				Decimals: 18,
			},
			{
				Address:  "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
				Decimals: 6,
			},
		},
		StaticExtra: string(jsonify(StaticExtra{
			BaseTokenAddress:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
			QuoteTokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		})),
		Reserves: entity.PoolReserves{
			"10000000000000000000000", // 10_000, dec 18
			"10000000000",             // 10_000, dec 6
		},
		Extra: string(jsonify(
			Extra{
				BaseToQuotePriceLevels: []PriceLevel{
					{
						Price:  0.6,
						Amount: 10,
					},
					{
						Price:  0.5,
						Amount: 10,
					},
				},
				QuoteToBasePriceLevels: []PriceLevel{
					{
						Price:  1,
						Amount: 1,
					},
					{
						Price:  2,
						Amount: 10,
					},
				},
			},
		)),
	})
	require.NoError(t, err)

	// test base -> quote
	{
		limit := swaplimit.NewInventoryWithSwapped("kyber-pmm", ps.CalculateLimit())
		amtIn1, _ := new(big.Int).SetString("10000000000000000000", 10) // 10 KNC
		res1, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: amtIn1,
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "6000000", res1.TokenAmountOut.Amount.String()) // 60$

		ps.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: amtIn1,
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: res1.TokenAmountOut.Amount,
			},
			SwapLimit: limit,
		})

		amtIn2, _ := new(big.Int).SetString("1000000000000000000", 10) // 1 KNC
		res2, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: amtIn2,
			},
			TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "500000", res2.TokenAmountOut.Amount.String()) // 5$
	}

	// test quote -> base
	{
		limit := swaplimit.NewInventoryWithSwapped("kyber-pmm", ps.CalculateLimit())
		amtIn1, _ := new(big.Int).SetString("1000000", 10) // 1 USDT
		res1, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: amtIn1,
			},
			TokenOut: "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "1000000000000000000", res1.TokenAmountOut.Amount.String()) // 1 KNC

		ps.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: amtIn1,
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: res1.TokenAmountOut.Amount,
			},
			SwapLimit: limit,
		})

		amtIn2, _ := new(big.Int).SetString("1000000", 10) // 1 USDT
		res2, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: amtIn2,
			},
			TokenOut: "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "2000000000000000000", res2.TokenAmountOut.Amount.String()) // 2 KNC
	}
}

func jsonify(data any) []byte {
	v, _ := json.Marshal(data)

	return v
}

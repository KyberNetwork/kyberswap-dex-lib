package kyberpmm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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

func TestPoolSimulator_getNewPriceLevelsStateByAmountIn(t *testing.T) {
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
			oldPriceLevels := slices.Clone(tt.args.priceLevels)
			newPriceLevels := getNewPriceLevelsStateByAmountIn(tt.args.amountIn, tt.args.priceLevels)

			assert.ElementsMatch(t, tt.expectedPriceLevels, newPriceLevels)
			assert.ElementsMatch(t, oldPriceLevels, tt.args.priceLevels)
		})
	}
}

func TestPoolSimulator_getNewPriceLevelsStateByAmountOut(t *testing.T) {
	type args struct {
		amountOut   *big.Float
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
				amountOut:   new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when fully filled",
			args: args{
				amountOut: new(big.Float).SetFloat64(100),
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
				amountOut: new(big.Float).SetFloat64(200),
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
				amountOut: new(big.Float).SetFloat64(500),
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
				amountOut: new(big.Float).SetFloat64(199),
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
			oldPriceLevels := slices.Clone(tt.args.priceLevels)
			newPriceLevels := getNewPriceLevelsStateByAmountOut(tt.args.amountOut, tt.args.priceLevels)

			assert.ElementsMatch(t, tt.expectedPriceLevels, newPriceLevels)
			assert.ElementsMatch(t, oldPriceLevels, tt.args.priceLevels)
		})
	}
}

func TestPoolSimulator_swapLimit(t *testing.T) {
	ps, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{
			{
				Address:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Decimals: 18,
				Symbol:   "KNC",
			},
			{
				Address:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Decimals: 6,
				Symbol:   "USDT",
			},
			{
				Address:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Decimals: 6,
				Symbol:   "USDC",
			},
		},
		StaticExtra: string(jsonify(StaticExtra{
			BaseTokenAddress: "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
			QuoteTokenAddresses: []string{
				"0xdac17f958d2ee523a2206206994597c13d831ec7",
				"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
		})),
		Reserves: entity.PoolReserves{
			"10000000000000000000000", // 10_000, dec 18
			"10000000000",             // 10_000, dec 6
			"10000000000",             // 10_000, dec 6
		},
		Extra: string(jsonify(
			Extra{
				PriceLevels: map[string]BaseQuotePriceLevels{
					"KNC/USDT": {
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
					"KNC/USDC": {
						BaseToQuotePriceLevels: []PriceLevel{
							{
								Price:  0.8,
								Amount: 10,
							},
							{
								Price:  0.7,
								Amount: 10,
							},
						},
						QuoteToBasePriceLevels: []PriceLevel{
							{
								Price:  3,
								Amount: 1,
							},
							{
								Price:  4,
								Amount: 10,
							},
						},
					},
				},
			},
		)),
	})
	require.NoError(t, err)

	// test base -> quote
	{
		limit := swaplimit.NewInventory("kyber-pmm", ps.CalculateLimit())
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
		assert.Equal(t, "6000000", res1.TokenAmountOut.Amount.String()) // 60 USDT

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
		assert.Equal(t, "500000", res2.TokenAmountOut.Amount.String()) // 5 USDT

		ps.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: amtIn2,
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: res2.TokenAmountOut.Amount,
			},
			SwapLimit: limit,
		})

		amtIn3, _ := new(big.Int).SetString("1000000000000000000", 10) // 1 KNC
		res3, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: amtIn3,
			},
			TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "700000", res3.TokenAmountOut.Amount.String()) // 8 USDC
	}

	// test quote -> base
	{
		limit := swaplimit.NewInventory("kyber-pmm", ps.CalculateLimit())
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

		ps.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Amount: amtIn2,
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
				Amount: res2.TokenAmountOut.Amount,
			},
			SwapLimit: limit,
		})

		amtIn3, _ := new(big.Int).SetString("1000000", 10) // 1 USDC
		res3, err := ps.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: amtIn3,
			},
			TokenOut: "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202",
			Limit:    limit,
		})
		require.NoError(t, err)
		assert.Equal(t, "4000000000000000000", res3.TokenAmountOut.Amount.String()) // 4 KNC
	}
}

func jsonify(data any) []byte {
	v, _ := json.Marshal(data)

	return v
}

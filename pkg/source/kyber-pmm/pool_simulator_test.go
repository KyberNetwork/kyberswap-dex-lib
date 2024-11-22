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
		limit := swaplimit.NewSwappedInventory("kyber-pmm", ps.CalculateLimit())
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
		limit := swaplimit.NewSwappedInventory("kyber-pmm", ps.CalculateLimit())
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

func TestPoolSimulator_calculateLimit(t *testing.T) {
	data := `
      {
        "address": "kyber_pmm_0x4d224452801aced8b2f0aebe155379bb5d594381_0xdac17f958d2ee523a2206206994597c13d831ec7",
        "reserveUsd": 1360970.7667567069,
        "amplifiedTvl": 1360970.7667567069,
        "exchange": "kyber-pmm",
        "type": "kyber-pmm",
        "timestamp": 1732272598,
        "reserves": [
          "210003216790492160720",
          "1359367795230"
        ],
        "tokens": [
          {
            "address": "0x4d224452801aced8b2f0aebe155379bb5d594381",
            "name": "ApeCoin",
            "symbol": "APE",
            "decimals": 18,
            "weight": 0,
            "swappable": true
          },
          {
            "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
            "name": "Tether USD",
            "symbol": "USDT",
            "decimals": 6,
            "weight": 0,
            "swappable": true
          }
        ],
        "extra": "{\"baseToQuotePriceLevels\":[{\"price\":1.1426086945029605,\"amount\":1448.3925369383157},{\"price\":1.14428955845081,\"amount\":4345.177610814947},{\"price\":1.1441893875816131,\"amount\":4779.695371896442},{\"price\":1.1440687560594114,\"amount\":5257.664909086088},{\"price\":1.143679713857277,\"amount\":5784.6637098782685},{\"price\":1.142752749157162,\"amount\":6367.316457895198},{\"price\":1.142540120616818,\"amount\":7004.0584093461075},{\"price\":1.1422856716849081,\"amount\":7704.464250280718},{\"price\":1.1419786762349307,\"amount\":8474.910675308798},{\"price\":1.1416082956442937,\"amount\":9322.401742839684},{\"price\":1.1411612434242049,\"amount\":10254.641917123634},{\"price\":1.1406214835799104,\"amount\":11280.106108836},{\"price\":1.1395740595704091,\"amount\":12412.425629594145},{\"price\":1.1381886889713548,\"amount\":13660.848853169038},{\"price\":1.1372387982757062,\"amount\":15026.933738485925},{\"price\":1.136091185210735,\"amount\":16529.627112334536},{\"price\":1.1347044942686983,\"amount\":18182.589823567978},{\"price\":1.132319788574757,\"amount\":20013.375000310654},{\"price\":1.130016117550905,\"amount\":22020.16527190755},{\"price\":1.1275704951197274,\"amount\":24222.181799098296},{\"price\":1.1246649343962203,\"amount\":21379.00293962797}],\"quoteToBasePriceLevels\":[{\"price\":0.8567231469341043,\"amount\":243.89816179743246}]}",
        "staticExtra": "{\"pairID\":\"APE/USDT\",\"baseTokenAddress\":\"0x4d224452801ACEd8B2F0aebE155379bb5D594381\",\"quoteTokenAddress\":\"0xdAC17F958D2ee523a2206206994597C13D831ec7\"}"
      }`
	var entPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(data), &entPool))

	pSim, err := NewPoolSimulator(entPool)
	require.NoError(t, err)

	limits := pSim.CalculateLimit()

	t.Log(string(jsonify(limits)))
	// "0x4d224452801aced8b2f0aebe155379bb5d594381":210003216790492160720,"0xdac17f958d2ee523a2206206994597c13d831ec7":1359367795230}
}

func jsonify(data any) []byte {
	v, _ := json.Marshal(data)

	return v
}

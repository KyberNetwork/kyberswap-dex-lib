package dodo

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://github.com/DODOEX/contractV2/blob/1c8d393/test/DPP/trader.test.ts#L27
	// the numbers might be off by 1-4 wei, should be acceptable
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
		expectedRevBase   string
		expectedRevQuote  string
	}{
		// buy at R=1
		{"QUOTE", decStr(100), "BASE", "986174542266106306", "9012836315765723076", decStr(1100)},
		// sell at R=1
		{"BASE", decStr(1), "QUOTE", "98617454226610630667", decStr(11), "901283631576572307517"},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {

			p, err := NewPoolSimulator(entity.Pool{
				Exchange: "",
				Type:     "",
				SwapFee:  0.001 + 0.002,
				Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
				Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
					decStr(10), decStr(1000),
					decStr(10), decStr(1000),
					decStr(100),          // i=100
					"100000000000000000", // k=0.1
					0,
					"0.001",
					"0.002",
				),
				StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
					"BASE", "QUOTE", "DPP", ""),
			})
			require.Nil(t, err)

			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: amountIn,
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  amountIn,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})

			assert.Equal(t, bignumber.NewBig10(tc.expectedRevBase), p.Info.Reserves[0])
			b, _ := new(big.Float).Mul(p.PoolSimulatorState.B, bignumber.BoneFloat).Int(nil)
			assert.Equal(t, bignumber.NewBig10(tc.expectedRevBase), b)

			assert.Equal(t, bignumber.NewBig10(tc.expectedRevQuote), p.Info.Reserves[1])
			q, _ := new(big.Float).Mul(p.PoolSimulatorState.Q, bignumber.BoneFloat).Int(nil)
			assert.Equal(t, bignumber.NewBig10(tc.expectedRevQuote), q)
		})
	}
}

func TestCanSwapTo(t *testing.T) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		SwapFee:  0.001 + 0.002,
		Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
		Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
			decStr(10), decStr(1000),
			decStr(10), decStr(1000),
			decStr(100),          // i=100
			"100000000000000000", // k=0.1
			0,
			"0.001",
			"0.002",
		),
		StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
			"BASE", "QUOTE", "DPP", ""),
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"QUOTE"}, p.CanSwapTo("BASE"))
	assert.Equal(t, []string{"BASE"}, p.CanSwapTo("QUOTE"))
	assert.Equal(t, 0, len(p.CanSwapTo("XX")))
}

func decStr(amt int64) string {
	return new(big.Int).Mul(big.NewInt(amt), bignumber.TenPowInt(18)).String()
}

func TestCalcAmountOut_PoolDepleted(t *testing.T) {
	poolRedis := "{\"address\":\"0x813fddeccd0401c4fa73b092b074802440544e52\",\"reserveUsd\":233579.38593018247,\"amplifiedTvl\":233579.38593018247,\"swapFee\":0.00002,\"exchange\":\"dodo\",\"type\":\"dodo-classical\",\"timestamp\":1621520160,\"reserves\":[\"76749655089\",\"156891845555\"],\"tokens\":[{\"address\":\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\",\"name\":\"USD Coin (PoS)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xc2132d05d31c914a87c6611c10748aeb04b58e8f\",\"name\":\"(PoS) Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":1000000000000000000,\\\"k\\\":200000000000000,\\\"rStatus\\\":1,\\\"mtFeeRate\\\":\\\"2e-05\\\",\\\"lpFeeRate\\\":\\\"0\\\",\\\"swappable\\\":true,\\\"reserves\\\":[76749655089,156891845555],\\\"targetReserves\\\":[104334457223,129305060381]}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0x813fddeccd0401c4fa73b092b074802440544e52\\\",\\\"lpToken\\\":\\\"0x2c5ca709d9593f6fd694d84971c55fb3032b87ab\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\\\",\\\"0xc2132d05d31c914a87c6611c10748aeb04b58e8f\\\"],\\\"dodoV1SellHelper\\\":\\\"0xdfaf9584f5d229a9dbe5978523317820a8897c5a\\\"}\"}"
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	poolSim, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	// 1st swap 500 USDT
	{
		amountIn := pool.TokenAmount{Token: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Amount: bignumber.NewBig10("500000000")}
		out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: amountIn,
				TokenOut:      "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Limit:         nil,
			})
		})
		require.Nil(t, err)

		poolSim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  amountIn,
			TokenAmountOut: *out.TokenAmountOut,
			Fee:            *out.Fee,
			SwapInfo:       out.SwapInfo,
		})
	}

	// 2nd swap 1000 USDT, if there are no reserve check then this will yield negative output
	{
		amountIn := pool.TokenAmount{Token: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Amount: bignumber.NewBig10("1000000000")}
		_, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: amountIn,
				TokenOut:      "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Limit:         nil,
			})
		})
		require.NotNil(t, err)
		fmt.Println(err)
	}
}

package dodo

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			out, err := p.CalcAmountOut(amountIn, tc.out)
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

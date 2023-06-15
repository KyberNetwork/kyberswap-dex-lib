package dodo

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
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

			p, err := NewPool(entity.Pool{
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

			amountIn := pool.TokenAmount{Token: tc.in, Amount: utils.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(amountIn, tc.out)
			require.Nil(t, err)
			assert.Equal(t, utils.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  amountIn,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})

			assert.Equal(t, utils.NewBig10(tc.expectedRevBase), p.Info.Reserves[0])
			b, _ := new(big.Float).Mul(p.PoolState.B, constant.BoneFloat).Int(nil)
			assert.Equal(t, utils.NewBig10(tc.expectedRevBase), b)

			assert.Equal(t, utils.NewBig10(tc.expectedRevQuote), p.Info.Reserves[1])
			q, _ := new(big.Float).Mul(p.PoolState.Q, constant.BoneFloat).Int(nil)
			assert.Equal(t, utils.NewBig10(tc.expectedRevQuote), q)
		})
	}
}

func decStr(amt int64) string {
	return new(big.Int).Mul(big.NewInt(amt), constant.TenPowInt(18)).String()
}

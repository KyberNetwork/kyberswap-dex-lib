package mantisswap

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenIn     string
		amountIn    string
		tokenOut    string
	}
	testcases := []testcase{
		{
			name: "swap USDC.e for DAI",
			poolEncoded: `{
				"address": "0x62ba5e1ab1fa304687f132f67e35bfc5247166ad",
				"type": "mantisswap",
				"timestamp": 1705354354,
				"reserves": [
					"3206954397",
					"4036310239",
					"1749719254748797676026"
				],
				"tokens": [
					{
						"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
						"weight": 1,
						"swappable": true
					},
					{
						"address": "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
						"weight": 1,
						"swappable": true
					},
					{
						"address": "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
						"weight": 1,
						"swappable": true
					}
				],
				"extra": "{\"Paused\":false,\"SwapAllowed\":true,\"BaseFee\":100,\"LpRatio\":50,\"SlippageA\":8,\"SlippageN\":16,\"SlippageK\":1000000000000000000,\"LPs\":{\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":{\"address\":\"0xe03aec0d08b3158350a9ab99f6cea7ba9513b889\",\"decimals\":6,\"asset\":3206954397,\"liability\":3082104986,\"liabilityLimit\":2000000000000},\"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063\":{\"address\":\"0x4b3bfcaa4f8bd4a276b81c110640da634723e64b\",\"decimals\":18,\"asset\":1749719254748797676026,\"liability\":2538765916906832854207,\"liabilityLimit\":2000000000000000000000000},\"0xc2132d05d31c914a87c6611c10748aeb04b58e8f\":{\"address\":\"0xe8a1ead2f4c454e319b76fa3325b754c47ce1820\",\"decimals\":6,\"asset\":4036310239,\"liability\":2921143438,\"liabilityLimit\":2000000000000}}}"
			}`,
			tokenIn:  "0x2791bca1f2de4661ed88a30c99a7a9449aa84174", // USDC.e
			amountIn: "1000000000",                                 // 1000
			tokenOut: "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063", // DAI
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})
	}
}

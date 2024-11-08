package uniswap

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
			name: "swap WETH for USDT",
			poolEncoded: `{
				"address": "0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852",
				"swapFee": 0.003,
				"type": "uniswap",
				"timestamp": 1705356253,
				"reserves": [
					"32981129686811504138006",
					"83362838693979"
				],
				"tokens": [
					{
						"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"weight": 50,
						"swappable": true
					}
				]
			}`,
			tokenIn:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn: "1000000000000000000", // 1
			tokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
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

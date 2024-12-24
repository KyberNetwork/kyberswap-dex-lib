package polmatic

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
			name: "swap MATIC for POL",
			poolEncoded: `{
				"address": "0x29e7df7b6a1b2b07b731457f499e1696c60e2c4e",
				"type": "pol-matic",
				"timestamp": 1705354961,
				"reserves": [
					"22046699825896000703658510",
					"9977954296312119119296341490"
				],
				"tokens": [
					{
						"address": "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0",
						"decimals": 18,
						"swappable": true
					},
					{
						"address": "0x455e53cbb86018ac2b8092fdcd39d8444affc3f6",
						"decimals": 18,
						"swappable": true
					}
				]
			}`,
			tokenIn:  "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0",
			amountIn: "100000000000000000000000", // 100000
			tokenOut: "0x455e53cbb86018ac2b8092fdcd39d8444affc3f6",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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

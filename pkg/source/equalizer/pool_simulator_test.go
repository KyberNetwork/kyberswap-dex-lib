package equalizer

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
			name: "v-WETH/USDbC pool",
			poolEncoded: `{
				"address": "0xf3f1f5760a614b8146eec5d1c94658720c2425b9",
				"swapFee": 0.002666666666666667,
				"type": "equalizer",
				"timestamp": 1705345162,
				"reserves": [
					"173810100394741222630",
					"441959784673"
				],
				"tokens": [
					{
						"address": "0x4200000000000000000000000000000000000006",
						"decimals": 18,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					}
				],
				"staticExtra": "{\"stable\":false}"
			}`,
			tokenIn:  "0x4200000000000000000000000000000000000006",
			amountIn: "1000000000000000000",
			tokenOut: "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
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

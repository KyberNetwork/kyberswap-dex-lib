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

func TestStablePoolAmountOut(t *testing.T) {
	poolEncoded := `{
		"address": "0x86de9fa6faecd1c5e05d7612c626f9063da4506e",
		"swapFee": 0.0189,
		"exchange": "equal",
		"type": "equalizer",
		"reserves": ["139388053889230476", "735588392599391348"],
		"tokens": [
			{"address": "0x50c42deacd8fc9773493ed674b675be577f2634b", "symbol": "WETH", "decimals": 18, "swappable": true},
			{"address": "0xdc2de2f2c0122ff7cb8482dc47da75a6a5d1a88b", "symbol": "eliteRingsScETH", "decimals": 18, "swappable": true}
		],
		"staticExtra": "{\"stable\":true}"
	}`

	poolEntity := new(entity.Pool)
	require.NoError(t, json.Unmarshal([]byte(poolEncoded), poolEntity))

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenIn := "0x50c42deacd8fc9773493ed674b675be577f2634b"
	tokenOut := "0xdc2de2f2c0122ff7cb8482dc47da75a6a5d1a88b"

	validAmounts := []string{
		"1000",
		"1000000",
		"1000000000000000",
		"10000000000000000",
		"50000000000000000",
		"100000000000000000",
	}

	for _, amtStr := range validAmounts {
		result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(amtStr)},
			TokenOut:      tokenOut,
		})
		require.NoErrorf(t, err, "amountIn=%s should not error", amtStr)
		require.Positivef(t, result.TokenAmountOut.Amount.Sign(), "amountIn=%s should produce positive amountOut", amtStr)
	}
}

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	t.Parallel()
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

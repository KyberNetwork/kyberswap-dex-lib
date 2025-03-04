package ekubo

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TODO Oracle & exact out

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
			name: "test pool",
			poolEncoded: `{
				"staticExtra": "{
					\"poolKey\": {
						\"token0\": \"0x0000000000000000000000000000000000000001\",
						\"token1\": \"0x0000000000000000000000000000000000000002\",
						\"config\": {
							\"tickSpacing\": 100,
							\"fee\": 922337203685477,
							\"extension\": \"0x0000000000000000000000000000000000000000\"
						}
					},
					\"extension\": 0
				}",
				"extra": "{
					\"state\": {
						\"liquidity\": 99999,
						\"activeTick\": -20201601,
						\"sqrtRatio\": 13967539110995781342936001321080700,
						\"tickBounds\": [-88722000, 88722000],
						\"ticks\": [
							{
								\"number\": -88722000,
								\"liquidityDelta\": 99999
							},
							{
								\"number\": -24124600,
								\"liquidityDelta\": 103926982998885
							},
							{
								\"number\": -24124500,
								\"liquidityDelta\": -103926982998885
							},
							{
								\"number\": -20236100,
								\"liquidityDelta\": 20192651866847
							},
							{
								\"number\": -20235900,
								\"liquidityDelta\": 676843433645
							},
							{
								\"number\": -20235400,
								\"liquidityDelta\": 620315686813
							},
							{
								\"number\": -20235000,
								\"liquidityDelta\": 3899271022058
							},
							{
								\"number\": -20234900,
								\"liquidityDelta\": 1985516133391
							},
							{
								\"number\": -20233000,
								\"liquidityDelta\": 2459469409600
							},
							{
								\"number\": -20232100,
								\"liquidityDelta\": -20192651866847
							},
							{
								\"number\": -20231900,
								\"liquidityDelta\": -663892969024
							},
							{
								\"number\": -20231400,
								\"liquidityDelta\": -620315686813
							},
							{
								\"number\": -20231000,
								\"liquidityDelta\": -3516445235227
							},
							{
								\"number\": -20230900,
								\"liquidityDelta\": -1985516133391
							},
							{
								\"number\": -20229000,
								\"liquidityDelta\": -2459469409600
							},
							{
								\"number\": -20227900,
								\"liquidityDelta\": -12950464621
							},
							{
								\"number\": -20227000,
								\"liquidityDelta\": -382825786831
							},
							{
								\"number\": -2000,
								\"liquidityDelta\": 140308196
							},
							{
								\"number\": 2000,
								\"liquidityDelta\": -140308196
							},
							{
								\"number\": 88722000,
								\"liquidityDelta\": -99999
							}
						]
					}
				}"
			}`,
			tokenIn:  "0x0000000000000000000000000000000000000002",
			amountIn: "1000000",
			tokenOut: "0x0000000000000000000000000000000000000001",
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

			require.True(t, result.TokenAmountOut.Amount.Cmp(new(big.Int).SetUint64(2436479431)) == 0)

			_ = result
		})
	}
}

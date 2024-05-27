package uniswapv3

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name     string
		tokenIn  string
		amountIn string
		tokenOut string
	}
	testcases := []testcase{
		{
			name:     "swap WETH for UNI",
			tokenIn:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn: "1000000000000000000", // 1
			tokenOut: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
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

		t.Run(tc.name+"new sim", func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
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

func TestComparePoolSimulatorV2(t *testing.T) {
	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(poolEncoded), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	poolSimV2, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	for i := 0; i < 500; i++ {
		amt := RandNumberString(24)

		t.Run(fmt.Sprintf("test %s WETH -> UNI %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s WETH -> UNI (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
				Limit:   nil,
			})

			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s UNI -> WETH %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s UNI -> WETH (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Limit:   nil,
			})
			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})
	}
}

// not really random but should be enough for testing
func RandNumberString(maxLen int) string {
	sLen := rand.Intn(maxLen-1) + 1
	var s string
	for i := 0; i < sLen; i++ {
		var c int
		if i == 0 {
			c = rand.Intn(9) + 1
		} else {
			c = rand.Intn(10)
		}
		s = fmt.Sprintf("%s%d", s, c)
	}
	return s
}

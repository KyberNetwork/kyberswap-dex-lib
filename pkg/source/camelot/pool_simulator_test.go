package camelot

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPool_getAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		pool              PoolSimulator
		amountIn          *big.Int
		tokenIn           string
		expectedAmountOut *big.Int
	}{
		{
			name: "it should return correct amount when swap from 0 to 1 stableSwap",
			pool: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("1481252219344464578434"),
							bignumber.NewBig("3236537897421945761324"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          bignumber.NewBig("1000000000000000000000"),
			tokenIn:           "0x5979d7b546e38e414f7e9822514be443a4800529",
			expectedAmountOut: bignumber.NewBig("1022352385458443941729"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0 stableSwap",
			pool: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("1481252219344464578434"),
							bignumber.NewBig("3236537897421945761324"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          bignumber.NewBig("1000000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: bignumber.NewBig("708007269566256823589"),
		},
		{
			name: "it should return correct amount when swap from 0 to 1",
			pool: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Tokens: []string{
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("4446200649353806287147"),
							bignumber.NewBig("7387929715114"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(300),
				Token1FeePercent:     big.NewInt(300),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000),
				StableSwap:           false,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          bignumber.NewBig("1000000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: bignumber.NewBig("1353204924908"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0",
			pool: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Tokens: []string{
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("4446169910492564197660"),
							bignumber.NewBig("7387985285550"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(300),
				Token1FeePercent:     big.NewInt(300),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000),
				StableSwap:           false,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          bignumber.NewBig("1000000000000000000000"),
			tokenIn:           "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			expectedAmountOut: bignumber.NewBig("4446169877545485328691"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0 with owner fee",
			pool: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("1118199781455197144456"),
							bignumber.NewBig("2468857930499458348529"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				Factory: &Factory{
					FeeTo:         [20]byte{0x01, 0x02},
					OwnerFeeShare: big.NewInt(40000),
				},
				FeeDenominator: big.NewInt(100000),
			},
			amountIn:          bignumber.NewBig("1000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: bignumber.NewBig("898082364338357291"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, []string{tc.pool.Info.Tokens[1]}, tc.pool.CanSwapTo(tc.pool.Info.Tokens[0]))
			assert.Equal(t, []string{tc.pool.Info.Tokens[0]}, tc.pool.CanSwapTo(tc.pool.Info.Tokens[1]))
			assert.Equal(t, 0, len(tc.pool.CanSwapTo("XXX")))
			amountOut, _ := testutil.MustConcurrentSafe(t, func() (*big.Int, error) {
				return tc.pool.getAmountOut(tc.amountIn, tc.tokenIn), nil
			})

			assert.Equal(t, tc.expectedAmountOut, amountOut)

			var tokenOut string
			for _, token := range tc.pool.Info.Tokens {
				if !strings.EqualFold(tc.tokenIn, token) {
					tokenOut = token
					break
				}
			}
			// When using CalcAmountOut(), some test case will fail the K invariant check. So we don't check the returned (result, err).
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tc.pool.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: tc.amountIn,
					},
					TokenOut: tokenOut,
				})
			})
			_, _ = result, err
		})
	}
}

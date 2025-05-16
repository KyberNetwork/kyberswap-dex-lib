package uniswapv1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens:   []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576")},
					},
				},
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountOut: utils.NewBig("124570062"),
			expectedError:     nil,
		},
		{
			name: "[swap1to0] it should return correct amountOut",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens:   []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005")},
					},
				},
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountOut: utils.NewBig("161006857684289764421"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

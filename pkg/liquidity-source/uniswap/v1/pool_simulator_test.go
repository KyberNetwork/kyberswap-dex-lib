package uniswapv1

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("10089138480746"), bignumber.NewBig("10066716097576")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("10089138480746"),
					uint256.MustFromDecimal("10066716097576")},
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountOut: bignumber.NewBig("124570062"),
			expectedError:     nil,
		},
		{
			name: "[swap1to0] it should return correct amountOut",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("70361282326226590645832"),
							bignumber.NewBig("54150601005")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("70361282326226590645832"),
					uint256.MustFromDecimal("54150601005")},
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountOut: bignumber.NewBig("161006857684289764421"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
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

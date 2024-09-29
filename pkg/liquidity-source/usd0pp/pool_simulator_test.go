package usd0pp

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     *PoolSimulator
		param             poolpkg.CalcAmountOutParams
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 1718105400,
				endTime:   1844335800,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("106100000431432000000000"),
					Token:  USD0,
				},
				TokenOut: USD0PP,
			},
			expectedAmountOut: bignumber.NewBig("106100000431432000000000"),
		},
		{
			name: "it should return error when bond not started",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("40654517980271452478787"),
							bignumber.NewBig("40654517980271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 17018105400,
				endTime:   17108105410,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10610010000000000"),
					Token:  USD0,
				},
				TokenOut: USD0PP,
			},
			expectedError: ErrBondNotStarted,
		},
		{
			name: "it should return error when pool is paused",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("40654517980271452478787"),
							bignumber.NewBig("40654517980271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 1718105400,
				endTime:   1718105410,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10610010000000000"),
					Token:  USD0,
				},
				TokenOut: USD0PP,
			},
			expectedError: ErrBondEnded,
		},
		{
			name: "it should return error when tokenIn is invalid",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("40654517980271452478787"),
							bignumber.NewBig("40654517980271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 1718105400,
				endTime:   1844335800,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10610010000000000"),
					Token:  USD0PP,
				},
				TokenOut: USD0,
			},
			expectedError: ErrorInvalidTokenIn,
		},
		{
			name: "it should return error when tokenIn amount is invalid",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("40654517980271452478787"),
							bignumber.NewBig("40654517980271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 1718105400,
				endTime:   1844335800,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("-10610010000000000"),
					Token:  USD0,
				},
				TokenOut: USD0PP,
			},
			expectedError: ErrorInvalidTokenInAmount,
		},
		{
			name: "it should return error when tokenIn is invalid",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{USD0, USD0PP},
						Reserves: []*big.Int{
							bignumber.NewBig("40654517980271452478787"),
							bignumber.NewBig("40654517980271452478787"),
						},
					},
				},
				paused:    false,
				startTime: 1718105400,
				endTime:   1844335800,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10610010000000000"),
					Token:  USD0PP,
				},
				TokenOut: USD0,
			},
			expectedError: ErrorInvalidTokenIn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			assert.Equal(t, tc.poolSimulator.CanSwapTo(tc.poolSimulator.Info.Tokens[0]), []string{})
			assert.Equal(t, tc.poolSimulator.CanSwapTo(tc.poolSimulator.Info.Tokens[1]), []string{USD0})

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}

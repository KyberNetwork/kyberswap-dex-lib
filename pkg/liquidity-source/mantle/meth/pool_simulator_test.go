package meth

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
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
						Tokens: []string{WETH, METH},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				isStakingPaused:        false,
				minimumStakeBound:      uint256.MustFromDecimal("20000000000000000"),
				maximumMETHSupply:      uint256.MustFromDecimal("3000000000000000000000000"),
				totalControlled:        uint256.MustFromDecimal("491321321208383495845117"),
				exchangeAdjustmentRate: 4,
				mETHTotalSupply:        uint256.MustFromDecimal("469448183427363384875942"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("20000000000000000"),
					Token:  WETH,
				},
				TokenOut: METH,
			},
			expectedAmountOut: bignumber.NewBig("19101975994034486"),
		},
		{
			name: "it should return error when maximum METH supply exceeded",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, METH},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				isStakingPaused:        false,
				minimumStakeBound:      uint256.MustFromDecimal("20000000000000000"),
				maximumMETHSupply:      uint256.MustFromDecimal("3000000000000000000000000"),
				totalControlled:        uint256.MustFromDecimal("491321321208383495845117"),
				exchangeAdjustmentRate: 4,
				mETHTotalSupply:        uint256.MustFromDecimal("469448183427363384875942"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("3000000000000000000000000"),
					Token:  WETH,
				},
				TokenOut: METH,
			},
			expectedError: ErrMaximumMETHSupplyExceeded,
		},
		{
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, METH},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				isStakingPaused:        false,
				minimumStakeBound:      uint256.MustFromDecimal("20000000000000000"),
				maximumMETHSupply:      uint256.MustFromDecimal("3000000000000000000000000"),
				totalControlled:        uint256.MustFromDecimal("491321321208383495845117"),
				exchangeAdjustmentRate: 4,
				mETHTotalSupply:        uint256.MustFromDecimal("0"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("47852783468734432213"),
					Token:  WETH,
				},
				TokenOut: METH,
			},
			expectedAmountOut: bignumber.NewBig("47852783468734432213"),
		},
		{
			name: "it should return error when token in is invalid",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, METH},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				isStakingPaused: false,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("20000000000000000"),
					Token:  METH,
				},
				TokenOut: WETH,
			},
			expectedError: ErrorInvalidTokenIn,
		},
		{
			name: "it should return error when amount in is less than minimum stake bound",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, METH},
						Reserves: []*big.Int{
							bignumber.NewBig("406545179820271452478787"),
							bignumber.NewBig("406545179820271452478787"),
						},
					},
				},
				isStakingPaused:        false,
				minimumStakeBound:      uint256.MustFromDecimal("20000000000000000"),
				maximumMETHSupply:      uint256.MustFromDecimal("3000000000000000000000000"),
				totalControlled:        uint256.MustFromDecimal("491321321208383495845117"),
				exchangeAdjustmentRate: 4,
				mETHTotalSupply:        uint256.MustFromDecimal("469448183427363384875942"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("19999999999999999"),
					Token:  WETH,
				},
				TokenOut: METH,
			},
			expectedError: ErrMinimumStakeBoundNotSatisfied,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else if tc.expectedAmountOut != nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

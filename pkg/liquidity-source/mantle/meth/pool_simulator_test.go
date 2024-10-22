package meth

import (
	"math/big"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

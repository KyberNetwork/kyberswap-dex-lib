package ondo_usdy

import (
	"math/big"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
			name: "should return error when pool is paused",
			poolSimulator: &PoolSimulator{
				paused: true,
			},
			expectedError: ErrPoolPaused,
		},
		{
			name: "should return success when wrap",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
					Address: "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
					Tokens:  []string{"0x5be26527e817998a7206475496fde1e68957c5a6", "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3"},
				}},
				paused:      false,
				totalShares: uint256.MustFromDecimal("100000000000000000000000000000"),
				oraclePrice: uint256.NewInt(1064060720000000000),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0x5be26527e817998a7206475496fde1e68957c5a6",
					Amount: big.NewInt(2),
				},
				TokenOut: "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
			},
			expectedAmountOut: big.NewInt(2),
		},
		{
			name: "should return success when unwrap",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
					Address: "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
					Tokens:  []string{"0x5be26527e817998a7206475496fde1e68957c5a6", "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3"},
				}},
				paused:      false,
				totalShares: uint256.MustFromDecimal("100000000000000000000000000000"),
				oraclePrice: uint256.NewInt(1064060720000000000),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
					Amount: big.NewInt(2),
				},
				TokenOut: "0x5be26527e817998a7206475496fde1e68957c5a6",
			},
			expectedAmountOut: big.NewInt(1),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

package unieth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
			// tx: 0x3e278a310bfa787d0c05a9ec7007b10c6f655a5273a9126bc8844512887fc3d3
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, UNIETH},
					},
				},
				paused:         false,
				totalSupply:    bignumber.NewBig("40654517980271452478787"),
				currentReserve: bignumber.NewBig("43102498463014375406128"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("106100000000000000"),
					Token:  WETH,
				},
				TokenOut: UNIETH,
			},
			expectedAmountOut: bignumber.NewBig("100074114297761758"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Run("it should update balance correctly", func(t *testing.T) {
		poolSimulator := &PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{WETH, UNIETH},
				},
			},
			paused:         false,
			totalSupply:    bignumber.NewBig("40654517980271452478787"),
			currentReserve: bignumber.NewBig("43102498463014375406128"),
		}

		params := poolpkg.UpdateBalanceParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Amount: bignumber.NewBig("106100000000000000"),
			},
			TokenAmountOut: poolpkg.TokenAmount{
				Amount: bignumber.NewBig("100074114297761758"),
			},
		}

		poolSimulator.UpdateBalance(params)

		assert.Zero(t, poolSimulator.currentReserve.Cmp(bignumber.NewBig("43102604563014375406128")))
		assert.Zero(t, poolSimulator.totalSupply.Cmp(bignumber.NewBig("40654618054385750240545")))
	})
}

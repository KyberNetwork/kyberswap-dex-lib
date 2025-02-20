package overnightusdp

import (
	"math/big"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
			name: "Mint USD+ from USDC",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{
							"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
							"0xB79DD08EA68A908A97220C76d19A6aA9cBDE4376",
						},
						Reserves: []*big.Int{
							bignumber.NewBig(defaultReserves),
							bignumber.NewBig(defaultReserves),
						},
					},
				},
				isPaused:        false,
				buyFee:          bignumber.Ten,
				redeemFee:       bignumber.Ten,
				usdPlusDecimals: 6,
				assetDecimals:   18,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1000000000"),
					Token:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
				},
				TokenOut: "0xB79DD08EA68A908A97220C76d19A6aA9cBDE4376",
			},
			expectedAmountOut: bignumber.NewBig("999900000000000000000"),
		},
		{
			name: "Redeem USDC from USD+",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{
							"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
							"0xB79DD08EA68A908A97220C76d19A6aA9cBDE4376",
						},
						Reserves: []*big.Int{
							bignumber.NewBig(defaultReserves),
							bignumber.NewBig(defaultReserves),
						},
					},
				},
				isPaused:        false,
				buyFee:          bignumber.Ten,
				redeemFee:       bignumber.Ten,
				usdPlusDecimals: 6,
				assetDecimals:   6,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1000000000"),
					Token:  "0xB79DD08EA68A908A97220C76d19A6aA9cBDE4376",
				},
				TokenOut: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			},
			expectedAmountOut: bignumber.NewBig("999900000"),
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

package uniswap

import (
	"math/big"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulatorCalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens:   []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576")},
						SwapFee:  utils.NewBig("3000000000000000"),
					},
				},
				Weights: []uint{50, 50},
				gas:     defaultGas,
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountOut: utils.NewBig("124570062"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func BenchmarkPoolSimulatorCalcAmountOut(b *testing.B) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens:   []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576")},
						SwapFee:  utils.NewBig("3000000000000000"),
					},
				},
				Weights: []uint{50, 50},
				gas:     defaultGas,
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountOut: utils.NewBig("124570062"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tc.poolSimulator.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)
			}
		})
	}
}

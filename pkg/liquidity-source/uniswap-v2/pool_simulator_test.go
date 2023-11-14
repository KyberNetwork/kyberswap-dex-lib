package uniswapv2

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/blockchain-toolkit/number"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
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
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
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
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens:   []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
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
			result, err := tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tc.tokenAmountIn,
				TokenOut:      tc.tokenOut,
				Limit:         nil,
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           poolpkg.UpdateBalanceParams
		expectedReserves []*big.Int
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens:   []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: utils.NewBig("125224746")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7", Amount: utils.NewBig("124570062")},
				Fee:            poolpkg.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: utils.NewBig("375674")},
			},
			expectedReserves: []*big.Int{utils.NewBig("10089263705492"), utils.NewBig("10066591527514")},
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens:   []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7", Amount: utils.NewBig("124570062")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x32a7c02e79c4ea1008dd6564b35f131428673c41", Amount: utils.NewBig("161006857684289764421")},
			},
			expectedReserves: []*big.Int{utils.NewBig("70200275468542300881411"), utils.NewBig("54275171067")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[0].Cmp(tc.expectedReserves[0]))
			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[1].Cmp(tc.expectedReserves[1]))
		})
	}
}

func TestPoolSimulator_getAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		reserveIn         *uint256.Int
		reserveOut        *uint256.Int
		amountIn          *uint256.Int
		expectedAmountOut *uint256.Int
	}{
		{
			name:              "it should return correct amountOut",
			poolSimulator:     PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:         number.NewUint256("10089138480746"),
			reserveOut:        number.NewUint256("10066716097576"),
			amountIn:          number.NewUint256("125224746"),
			expectedAmountOut: number.NewUint256("124570062"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut := tc.poolSimulator.getAmountOut(tc.amountIn, tc.reserveIn, tc.reserveOut)

			assert.Equal(t, 0, tc.expectedAmountOut.Cmp(amountOut))
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
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens:   []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut: "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			}
		})
	}
}

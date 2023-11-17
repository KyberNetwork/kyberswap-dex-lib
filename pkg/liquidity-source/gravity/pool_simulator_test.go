package gravity

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/blockchain-toolkit/number"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// [0to1] https://polygonscan.com/tx/0xc3ae65a69af4130525d1fd6b0176bcb2bd47b5aa845c9420a929ef6dce8346bb
// [1to0] https://polygonscan.com/tx/0x0b8b28d9af5c291d15eaa5a71ccd29250482ca15560ba27b40e6c07affa70b52
func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedFee       *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x0dfbf1a50bdcb570bd0ff7bb307313b553a02598",
						Tokens:   []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a"},
						Reserves: []*big.Int{utils.NewBig("3676486287875125230347"), utils.NewBig("1595427043783762088")},
					},
				},
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("4541550000000000000"),
				Token:  "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
			},
			tokenOut:          "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a",
			expectedAmountOut: utils.NewBig("1961514449578860"),
			expectedFee:       utils.NewBig("981247848714"),
			expectedError:     nil,
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x0dfbf1a50bdcb570bd0ff7bb307313b553a02598",
						Tokens:   []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a"},
						Reserves: []*big.Int{utils.NewBig("3677564416437944883949"), utils.NewBig("1594968989213601445")},
					},
				},
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("667086035111400"),
				Token:  "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a",
			},
			tokenOut:          "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
			expectedAmountOut: utils.NewBig("1532098871305218355"),
			expectedFee:       utils.NewBig("766432651978599"),
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
				assert.Equal(t, tc.expectedFee, result.Fee.Amount)
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
						Address:  "0x0dfbf1a50bdcb570bd0ff7bb307313b553a02598",
						Tokens:   []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a"},
						Reserves: []*big.Int{utils.NewBig("3676486287875125230347"), utils.NewBig("1595427043783762088")},
					},
				},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Amount: utils.NewBig("4541550000000000000")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a", Amount: utils.NewBig("1961514449578860")},
				Fee:            poolpkg.TokenAmount{Token: "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a", Amount: utils.NewBig("981247848714")},
			},
			expectedReserves: []*big.Int{utils.NewBig("3681027837875125230347"), utils.NewBig("1593464548086334514")},
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x0dfbf1a50bdcb570bd0ff7bb307313b553a02598",
						Tokens:   []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a"},
						Reserves: []*big.Int{utils.NewBig("3677564416437944883949"), utils.NewBig("1594968989213601445")},
					},
				},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619a", Amount: utils.NewBig("667086035111400")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Amount: utils.NewBig("1532098871305218355")},
				Fee:            poolpkg.TokenAmount{Token: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Amount: utils.NewBig("766432651978599")},
			},
			expectedReserves: []*big.Int{utils.NewBig("3676031551133987686995"), utils.NewBig("1595636075248712845")},
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
			poolSimulator:     PoolSimulator{},
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

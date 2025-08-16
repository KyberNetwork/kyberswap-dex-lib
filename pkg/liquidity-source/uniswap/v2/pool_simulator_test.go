package uniswapv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolEncoded = `{"address":"0x9eb0bc7a207f77811ee365729d00152622a745b7","exchange":"pancake","type":"uniswap-v2","timestamp":1739501947,"reserves":["5789592094546501478373016","793623036600773033475"],"tokens":[{"address":"0x6d5ad1592ed9d6d1df9b93c793ab759573ed6714","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"fee\":25,\"feePrecision\":10000}"}`
	poolEntity  entity.Pool
	_           = lo.Must(0, json.Unmarshal([]byte(poolEncoded), &poolEntity))
	poolSim     = lo.Must(NewPoolSimulator(poolEntity))
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
			name: "[swap0to1] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
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
			name: "[swap1to0] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
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

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "[swap0to1] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenIn:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountIn: bignumber.NewBig("25075226"),
			expectedError:    nil,
		},
		{
			name: "[swap1to0] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000000000000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000000000000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenIn:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountIn: bignumber.NewBig("25075225677031093280"),
			expectedError:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: tc.tokenAmountOut,
				TokenIn:        tc.tokenIn,
				Limit:          nil,
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount)
			}
		})
	}

	testutil.TestCalcAmountIn(t, poolSim)
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           pool.UpdateBalanceParams
		expectedReserves []*uint256.Int
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: pool.UpdateBalanceParams{
				TokenAmountIn: pool.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig("125224746")},
				TokenAmountOut: pool.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: bignumber.NewBig("124570062")},
				Fee: pool.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig("375674")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("10089263705492"),
				uint256.MustFromDecimal("10066591527514")},
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: pool.UpdateBalanceParams{
				TokenAmountIn: pool.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: bignumber.NewBig("124570062")},
				TokenAmountOut: pool.TokenAmount{Token: "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
					Amount: bignumber.NewBig("161006857684289764421")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("70200275468542300881411"),
				uint256.MustFromDecimal("54275171067")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, tc.expectedReserves[0], tc.poolSimulator.reserves[0])
			assert.Equal(t, tc.expectedReserves[1], tc.poolSimulator.reserves[1])
		})
	}
}

func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
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

func TestPoolSimulator_getAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		reserveIn        *uint256.Int
		reserveOut       *uint256.Int
		amountOut        *uint256.Int
		expectedAmountIn *uint256.Int
		expectedErr      error
	}{
		{
			name:             "it should return correct amountIn",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("100000000"),
			reserveOut:       number.NewUint256("100000000000000000000"),
			amountOut:        number.NewUint256("20000000000000000000"),
			expectedAmountIn: number.NewUint256("25075226"),
			expectedErr:      nil,
		},
		{
			name:             "it should return correct ErrDSMathSubUnderflow error",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("1160689189059097452"),
			reserveOut:       number.NewUint256("1161607"),
			amountOut:        number.NewUint256("500000000"),
			expectedAmountIn: nil,
			expectedErr:      ErrDSMathSubUnderflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountIn, err := tc.poolSimulator.getAmountIn(tc.amountOut, tc.reserveIn, tc.reserveOut)
			assert.ErrorIs(t, err, tc.expectedErr)

			if err == nil {
				fmt.Printf("amountIn: %s\n", amountIn.String())
				assert.Equal(t, 0, tc.expectedAmountIn.Cmp(amountIn))
			}
		})
	}
}

func BenchmarkPoolSimulatorCalcAmountOut(b *testing.B) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
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
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut: "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = tc.poolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			}
		})
	}
}

func BenchmarkPoolSimulatorCalcAmountIn(b *testing.B) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "[swap0to1] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenIn:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountIn: bignumber.NewBig("25075226"),
			expectedError:    nil,
		},
		{
			name: "[swap1to0] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000000000000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000000000000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenIn:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountIn: bignumber.NewBig("25075225677031093280"),
			expectedError:    nil,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = tc.poolSimulator.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			}
		})
	}
}

package uniswapv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/blockchain-toolkit/number"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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
			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
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

var (
	poolRedisBench = `{
		"address": "0x3f3ee751ab00246cb0beec2e904ef51e18ac4d77",
		"reserveUsd": 0.036380575233191714,
		"amplifiedTvl": 0.036380575233191714,
		"exchange": "uniswap",
		"type": "uniswap-v2",
		"timestamp": 1705851986,
		"reserves": ["7535596323597", "8596033702378"],
		"tokens": [
			{ "address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "swappable": true },
			{ "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "swappable": true }
		],
		"extra": "{\"fee\":3,\"feePrecision\":1000}",
		"blockNumber": 19056224
	}`

	poolEnt entity.Pool
	_       = json.Unmarshal([]byte(poolRedisBench), &poolEnt)
)

func BenchmarkNewPoolSimulator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewPoolSimulator(poolEnt)
	}
}

func BenchmarkNewPoolSimulatorV2(b *testing.B) {
	sim, _ := NewPoolSimulatorV2(poolEnt)
	for i := 0; i < b.N; i++ {
		_ = InitPoolSimulator(poolEnt, sim)
	}
}

func TestComparePoolSimulatorV2(t *testing.T) {
	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)
	simV2, err := NewPoolSimulatorV2(poolEnt)
	require.Nil(t, err)

	for i := 0; i < 500; i++ {
		amt := testutil.RandNumberString(15)
		t.Run(fmt.Sprintf("test %s", amt), func(t *testing.T) {
			in := poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: utils.NewBig(amt),
					Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				},
				TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Limit:    nil,
			}
			res, err := sim.CalcAmountOut(in)
			resV2, errV2 := simV2.CalcAmountOut(in)

			if err == nil {
				require.Nil(t, errV2)
				assert.Equal(t, res, resV2)
			} else {
				require.NotNil(t, errV2)
			}
		})
	}
}

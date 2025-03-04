package alphafee

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	dexlibEntity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	poolMocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/pool"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAlphaFee_Calculation(t *testing.T) {
	t.Parallel()

	prices := map[string]float64{
		"I":  1.0,
		"W1": 1.0,
		"W2": 1.0,
		"W3": 1.0,
		"W4": 1.0,
		"W5": 1.0,
		"O":  1.0,
		"O1": 1.0,
		"O2": 3.0,
	}
	tokens := map[string]dexlibEntity.Token{
		"I": {
			Address:  "I",
			Decimals: 6,
		},
		"W1": {
			Address:  "W1",
			Decimals: 6,
		},
		"W2": {
			Address:  "W2",
			Decimals: 6,
		},
		"W3": {
			Address:  "W3",
			Decimals: 6,
		},
		"W4": {
			Address:  "W4",
			Decimals: 6,
		},
		"W5": {
			Address:  "W5",
			Decimals: 0,
		},
		"O": {
			Address:  "O",
			Decimals: 0,
		},
		"O1": {
			Address:  "O1",
			Decimals: 6,
		},
		"O2": {
			Address:  "O2",
			Decimals: 0,
		},
		"NonPrice": {
			Address:  "NonPrice",
			Decimals: 6,
		},
	}

	// mock amount out base on key pool-tokenIn-tokenOut
	amountOut := map[string]*big.Int{
		"I-W1":        big.NewInt(100),
		"W1-W2":       big.NewInt(200),
		"W2-W3":       big.NewInt(1000),
		"W3-W4":       big.NewInt(50),
		"W4-W5":       big.NewInt(50),
		"W5-O":        big.NewInt(30),
		"W3-W5":       big.NewInt(30),
		"W1-W3":       big.NewInt(40),
		"W2-W4":       big.NewInt(190),
		"W5-O1":       big.NewInt(10),
		"W4-O":        big.NewInt(150),
		"I-W3":        big.NewInt(40),
		"W3-O1":       big.NewInt(150),
		"W1-O":        big.NewInt(150),
		"W4-NonPrice": big.NewInt(150),
		"W5-NonPrice": big.NewInt(30),
		"W4-O2":       big.NewInt(150),
		"W5-O2":       big.NewInt(50),
	}
	exchanges := map[string]string{
		"pool-I-W1-I-W1":               "uniswapv2",
		"pool-W1-W2-W1-W2":             "kyber-pmm",
		"pool-W2-W3-W2-W3":             "kyber-pmm",
		"pool-W3-W4-W3-W4":             "synswap",
		"pool-W4-W5-W4-W5":             "kyber-pmm",
		"pool-W5-O-W5-O":               "kyber-pmm",
		"pool-W3-W5-W3-W5":             "kyber-pmm",
		"pool-W1-W3-W1-W3":             "pancake",
		"pool-W2-W4-W2-W4":             "mx_trading",
		"pool-W5-O1-W5-O1":             "uniswap-v3",
		"pool-W4-O-W4-O":               "uniswap-v3",
		"pool-I-W3-I-W3":               "uniswapv2",
		"pool-W3-O1-W3-O1":             "pancake",
		"pool-W1-O-W1-O":               "pancake",
		"pool-W5-NonPrice-W5-NonPrice": "kyber-pmm",
		"pool-W4-NonPrice-W4-NonPrice": "pancake",
		"pool-W4-O2-W4-O2":             "uniswap-v3",
		"pool-W5-O2-W5-O2":             "smardex",
	}
	poolTypes := map[string]string{
		"pool-I-W1-I-W1":               "uniswapv2",
		"pool-W1-W2-W1-W2":             "kyber-pmm",
		"pool-W2-W3-W2-W3":             "kyber-pmm",
		"pool-W3-W4-W3-W4":             "synswap",
		"pool-W4-W5-W4-W5":             "kyber-pmm",
		"pool-W5-O-W5-O":               "kyber-pmm",
		"pool-W3-W5-W3-W5":             "kyber-pmm",
		"pool-W1-W3-W1-W3":             "pancake",
		"pool-W2-W4-W2-W4":             "mx_trading",
		"pool-W5-O1-W5-O1":             "uniswap-v3",
		"pool-W4-O-W4-O":               "uniswap-v3",
		"pool-I-W3-I-W3":               "uniswapv2",
		"pool-W3-O1-W3-O1":             "pancake",
		"pool-W1-O-W1-O":               "pancake",
		"pool-W5-NonPrice-W5-NonPrice": "kyber-pmm",
		"pool-W4-NonPrice-W4-NonPrice": "pancake",
		"pool-W4-O2-W4-O2":             "uniswap-v3",
		"pool-W5-O2-W5-O2":             "smardex",
	}

	testCases := []struct {
		name             string
		bestRoute        *finderCommon.ConstructRoute
		bestAmmRoute     *finderCommon.ConstructRoute
		config           valueobject.AlphaFeeReductionConfig
		expectedAlphaFee *routerEntity.AlphaFee
		err              error
	}{
		{
			name: "Best route contains PMM swap that give greater amount than amm must return correct alpha fee, with all tokens have prices",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "O",
				AmountIn:       big.NewInt(120),
				AmountOut:      big.NewInt(180),
				AmountOutPrice: 180,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(90),
						AmountOut: big.NewInt(150),

						// 90 I -- amm -- 100 W1 -- pmm -- 200 W2 -- rfq -- 190 W4 -- amm -- 150 O
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-W2-W1-W2", "pool-W2-W4-W2-W4", "pool-W4-O-W4-O"},
						TokensOrder: []string{"I", "W1", "W2", "W4", "O"},
					},
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(30),

						//30 I -- amm -- 40 W3 -- pmm -- 20 W5 -- pmm -- 30 O
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-O-W5-O"},
						TokensOrder: []string{"I", "W3", "W5", "O"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(170),
				AmountOutPrice: 170,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 5000, //50%
				},
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "O",
				Amount: big.NewInt(5),
				Pool:   "pool-W5-O-W5-O",
			},
		},
		{
			name: "Best route contains PMM swap that give greater amount than amm must return correct alpha fee, with all tokens have prices, choose the shorter path",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "O",
				AmountIn:       big.NewInt(120),
				AmountOut:      big.NewInt(160),
				AmountOutPrice: 160,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(10),

						//30 I -- amm -- 40 W3 -- pmm -- 20 W5 -- amm -- 10 O1
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-O1-W5-O1"},
						TokensOrder: []string{"I", "W3", "W5", "O1"},
					},
					{
						AmountIn:  big.NewInt(90),
						AmountOut: big.NewInt(150),

						// 90 I -- amm -- 100 W1 -- pmm -- 200 W2 -- pmm -- 1000 W3 -- amm -- 150 O1
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-W2-W1-W2", "pool-W2-W3-W2-W3", "pool-W3-O1-W3-O1"},
						TokensOrder: []string{"I", "W1", "W2", "W3", "O1"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(100),
				AmountOutPrice: 100,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 1000, //10%
				},
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "W5",
				Amount: big.NewInt(6),
				Pool:   "pool-W3-W5-W3-W5",
			},
		},
		{
			name: "Return error because best route is AMM best route",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "O",
				AmountIn:       big.NewInt(120),
				AmountOut:      big.NewInt(300),
				AmountOutPrice: 300,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(150),

						//30 W3 -- amm -- 50 W4 -- amm -- 150 O
						PoolsOrder:  []string{"pool-W3-W4-W3-W4", "pool-W4-O-W4-O"},
						TokensOrder: []string{"W3", "W4", "O"},
					},
					{
						AmountIn:  big.NewInt(90),
						AmountOut: big.NewInt(150),

						// 90 I -- amm -- 100 W1 -- amm -- 150 O
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-O-W1-O"},
						TokensOrder: []string{"I", "W1", "O"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(300),
				AmountOutPrice: 300,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 1000, //10%
				},
			},
			err: ErrAlphaFeeNotExists,
		},
		{
			name: "Best route contains PMM swap that give greater amount than amm must return correct alpha fee, with all tokens have no prices",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "NonPrice",
				AmountIn:       big.NewInt(100),
				AmountOut:      big.NewInt(180),
				AmountOutPrice: 180,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(70),
						AmountOut: big.NewInt(150),

						// 70 I -- amm -- 100 W1 -- pmm -- 200 W2 -- rfq -- 190 W4 -- amm -- 150 NonPrice
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-W2-W1-W2", "pool-W2-W4-W2-W4", "pool-W4-NonPrice-W4-NonPrice"},
						TokensOrder: []string{"I", "W1", "W2", "W4", "NonPrice"},
					},
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(30),

						//30 I -- amm -- 40 W3 -- pmm -- 20 W5 -- pmm -- 30 NonPrice
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-NonPrice-W5-NonPrice"},
						TokensOrder: []string{"I", "W3", "W5", "NonPrice"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(80),
				AmountOutPrice: 80,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 2000, //20%
				},
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "NonPrice",
				Amount: big.NewInt(20),
				Pool:   "pool-W5-NonPrice-W5-NonPrice",
			},
		},
		{
			name: "Return error because pmm swap amount out is not enough to cover alpha fee",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "NonPrice",
				AmountIn:       big.NewInt(100),
				AmountOut:      big.NewInt(180),
				AmountOutPrice: 180,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(70),
						AmountOut: big.NewInt(150),

						// 70 I -- amm -- 100 W1 -- pmm -- 200 W2 -- rfq -- 190 W4 -- amm -- 150 NonPrice
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-W2-W1-W2", "pool-W2-W4-W2-W4", "pool-W4-NonPrice-W4-NonPrice"},
						TokensOrder: []string{"I", "W1", "W2", "W4", "NonPrice"},
					},
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(30),

						//30 I -- amm -- 40 W3 -- pmm -- 20 W5 -- pmm -- 30 NonPrice
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-NonPrice-W5-NonPrice"},
						TokensOrder: []string{"I", "W3", "W5", "NonPrice"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(80),
				AmountOutPrice: 80,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 4000, //40%
				},
			},
			err: ErrPMMSwapNotEnoughToCoverAlphaFee,
		},
		{
			name: "Still reduce alpha fee if the amount yeild by applying alpha fee is equal to amm best amount out",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "NonPrice",
				AmountIn:       big.NewInt(100),
				AmountOut:      big.NewInt(180),
				AmountOutPrice: 180,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(90),
						AmountOut: big.NewInt(150),

						// 90 I -- amm -- 100 W1 -- pmm -- 200 W2 -- rfq -- 190 W4 -- amm -- 150 NonPrice
						PoolsOrder:  []string{"pool-I-W1-I-W1", "pool-W1-W2-W1-W2", "pool-W2-W4-W2-W4", "pool-W4-NonPrice-W4-NonPrice"},
						TokensOrder: []string{"I", "W1", "W2", "W4", "NonPrice"},
					},
					{
						AmountIn:  big.NewInt(10),
						AmountOut: big.NewInt(30),

						//10 I -- amm -- 40 W3 -- pmm -- 20 W5 -- pmm -- 30 NonPrice
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-NonPrice-W5-NonPrice"},
						TokensOrder: []string{"I", "W3", "W5", "NonPrice"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(160),
				AmountOutPrice: 160,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 10000,
				},
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "NonPrice",
				Amount: big.NewInt(20),
				Pool:   "pool-W5-NonPrice-W5-NonPrice",
			},
		},
		{
			name: "Return correct alpha fee with different prices",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "O2",
				AmountIn:       big.NewInt(100),
				AmountOut:      big.NewInt(100),
				AmountOutPrice: 100,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(70),
						AmountOut: big.NewInt(50),

						// 70 I -- amm -- 40 W3 -- amm -- 50 W4 -- pmm -- 50 W4 -- amm -- 50 O2
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W4-W3-W4", "pool-W4-W5-W4-W5", "pool-W5-O2-W5-O2"},
						TokensOrder: []string{"I", "W3", "W4", "W5", "O2"},
					},
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(50),

						//30 I -- amm -- 40 W3 -- pmm -- 30 W5 -- amm -- 50 O2
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-O2-W5-O2"},
						TokensOrder: []string{"I", "W3", "W5", "O2"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(80),
				AmountOutPrice: 80,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 4000, //40%
				},
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "W5",
				Amount: big.NewInt(24),
				Pool:   "pool-W3-W5-W3-W5",
			},
		},
		{
			name: "Still return correct alpha fee if amm amount out below threshold",
			bestRoute: &finderCommon.ConstructRoute{
				TokenIn:        "I",
				TokenOut:       "O2",
				AmountIn:       big.NewInt(100),
				AmountOut:      big.NewInt(100),
				AmountOutPrice: 100,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:  big.NewInt(70),
						AmountOut: big.NewInt(50),

						// 70 I -- amm -- 40 W3 -- amm -- 50 W4 -- pmm -- 50 W4 -- amm -- 50 O2
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W4-W3-W4", "pool-W4-W5-W4-W5", "pool-W5-O2-W5-O2"},
						TokensOrder: []string{"I", "W3", "W4", "W5", "O2"},
					},
					{
						AmountIn:  big.NewInt(30),
						AmountOut: big.NewInt(50),

						//30 I -- amm -- 40 W3 -- pmm -- 30 W5 -- amm -- 50 O2
						PoolsOrder:  []string{"pool-I-W3-I-W3", "pool-W3-W5-W3-W5", "pool-W5-O2-W5-O2"},
						TokensOrder: []string{"I", "W3", "W5", "O2"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(10),
				AmountOutPrice: 10,
			},
			config: valueobject.AlphaFeeReductionConfig{
				ReductionFactorInBps: map[string]float64{
					"kyber-pmm": 4000, //40%
				},
				MaxThresholdPercentageInBps: int64(8000),
			},
			expectedAlphaFee: &routerEntity.AlphaFee{
				Token:  "W5",
				Amount: big.NewInt(24),
				Pool:   "pool-W3-W5-W3-W5",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			simulators := map[string]pool.IPoolSimulator{
				"pool-I-W1-I-W1":               poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W1-W2-W1-W2":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W2-W3-W2-W3":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W3-W4-W3-W4":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W4-W5-W4-W5":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W5-O-W5-O":               poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W3-W5-W3-W5":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W1-W3-W1-W3":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W2-W4-W2-W4":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W5-O1-W5-O1":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W1-W5-W1-W5":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W4-O-W4-O":               poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-I-W3-I-W3":               poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W3-O1-W3-O1":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W1-O-W1-O":               poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W4-NonPrice-W4-NonPrice": poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W5-NonPrice-W5-NonPrice": poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W4-O2-W4-O2":             poolMocks.NewMockIPoolSimulator(ctrl),
				"pool-W5-O2-W5-O2":             poolMocks.NewMockIPoolSimulator(ctrl),
			}
			for key, simulator := range simulators {
				simulator.(*poolMocks.MockIPoolSimulator).EXPECT().CalcAmountOut(gomock.Any()).DoAndReturn(func(arg0 interface{}) (*pool.CalcAmountOutResult, error) {
					param := arg0.(pool.CalcAmountOutParams)
					return &pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  param.TokenOut,
							Amount: amountOut[fmt.Sprintf("%s-%s", param.TokenAmountIn.Token, param.TokenOut)],
						},
						RemainingTokenAmountIn: &pool.TokenAmount{
							Token:  param.TokenOut,
							Amount: big.NewInt(10),
						},
					}, nil
				}).AnyTimes()

				simulator.(*poolMocks.MockIPoolSimulator).EXPECT().GetExchange().DoAndReturn(func() string {
					return exchanges[key]
				}).AnyTimes()

				simulator.(*poolMocks.MockIPoolSimulator).EXPECT().GetType().DoAndReturn(func() string {
					return poolTypes[key]
				}).AnyTimes()
			}
			poolSimulatorBucket := finderCommon.NewSimulatorBucket(
				simulators,
				map[string]pool.SwapLimit{}, finderCommon.DefaultCustomFuncs)

			alphaFeeCalculation := NewAlphaFeeCalculation(tc.config, finderCommon.DefaultCustomFuncs)
			res, err := alphaFeeCalculation.Calculate(context.TODO(), AlphaFeeParams{
				BestRoute:    tc.bestRoute,
				BestAmmRoute: tc.bestAmmRoute,

				Prices:              prices,
				Tokens:              tokens,
				PoolSimulatorBucket: poolSimulatorBucket,
			})

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.Equal(t, tc.expectedAlphaFee.Token, res.Token)
				assert.True(t, res.Amount.Cmp(tc.expectedAlphaFee.Amount) == 0)
			}

		})
	}
}

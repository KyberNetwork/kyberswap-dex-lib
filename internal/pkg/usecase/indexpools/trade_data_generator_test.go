package indexpools

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	poolMocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTradeDataGenerator_GenerateTradeData(t *testing.T) {
	tokens := map[string]*entity.SimplifiedToken{
		"token1": {
			Address:  "token1",
			Decimals: 1,
		},
		"token2": {
			Address:  "token2",
			Decimals: 1,
		},
		"token3": {
			Address:  "token3",
			Decimals: 1,
		},
		"token4": {
			Address:  "token4",
			Decimals: 1,
		},
		"token5": {
			Address: "token5",
		},
		"token6": {
			Address: "token6",
		},
		"token7": {
			Address: "token7",
		},
	}
	prices := map[string]*routerEntity.OnchainPrice{
		"token1": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1.5),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token2": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(20),
				Sell: big.NewFloat(0),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token3": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(10),
				Sell: big.NewFloat(50),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token4": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(15),
				Sell: big.NewFloat(12),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token5": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token6": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(8),
				Sell: big.NewFloat(6),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"token7": {
			USDPrice: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(10),
				Sell: big.NewFloat(10),
			},
		},
	}
	type testInput struct {
		name              string
		expectedTradeData map[float64]TradeData
		tokenIn           string
		tokenOut          string
		prepare           func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator
	}
	tests := []testInput{
		{
			name:     "it should generate maximum data points because no trade executions failed",
			tokenIn:  "token1",
			tokenOut: "token2",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				output := map[float64]int64{
					1:        1,
					10:       9,
					100:      99,
					1000:     999,
					10000:    999,
					100000:   999,
					1000000:  999,
					10000000: 999,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
					param := arg0.(poolpkg.CalcAmountOutParams)
					return &poolpkg.CalcAmountOutResult{
						TokenAmountOut: &poolpkg.TokenAmount{
							Token:  "token2",
							Amount: big.NewInt(output[param.TokenAmountIn.AmountUsd]),
						},
					}, nil
				}).Times(config.MaxDataPointNumber + 1)

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1,
					AmountOutUsd: 2,
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10,
					AmountOutUsd: 18,
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  100,
					AmountOutUsd: 198,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1000,
					AmountOutUsd: 1998,
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10000,
					AmountOutUsd: 1998,
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  100000,
					AmountOutUsd: 1998,
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1000000,
					AmountOutUsd: 1998,
				},
				10000000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10000000,
					AmountOutUsd: 1998,
				},
			},
		},
		{
			name:     "it should generate minimum data points because there are error during calcAmountOut",
			tokenIn:  "token1",
			tokenOut: "token3",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				amountOutMapping := map[float64]int64{
					1:     1,
					10:    8,
					100:   99,
					1000:  999,
					10000: 991,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).
					DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
						param := arg0.(poolpkg.CalcAmountOutParams)
						if param.TokenAmountIn.AmountUsd == float64(100000) || param.TokenAmountIn.AmountUsd == float64(1000000) {
							return nil, errors.New("mock error")
						}
						return &poolpkg.CalcAmountOutResult{
							TokenAmountOut: &poolpkg.TokenAmount{
								Token:  "token3",
								Amount: big.NewInt(amountOutMapping[param.TokenAmountIn.AmountUsd]),
							},
						}, nil
					}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1,
					AmountOutUsd: 1,
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10,
					AmountOutUsd: 8,
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100,
					AmountOutUsd: 99,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1000,
					AmountOutUsd: 999,
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10000,
					AmountOutUsd: 991,
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 991,
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1000000,
					AmountOutUsd: 991,
				},
			},
		},
		{
			name:     "it should continue generate trade data although fist trade generation (1USD) failed",
			tokenIn:  "token1",
			tokenOut: "token3",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				amountOutMapping := map[float64]int64{
					10:     8,
					100:    99,
					1000:   999,
					10000:  991,
					100000: 9999,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).
					DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
						param := arg0.(poolpkg.CalcAmountOutParams)
						if param.TokenAmountIn.AmountUsd == float64(1) || param.TokenAmountIn.AmountUsd == float64(1000000) {
							return nil, errors.New("mock error")
						}
						return &poolpkg.CalcAmountOutResult{
							TokenAmountOut: &poolpkg.TokenAmount{
								Token:  "token3",
								Amount: big.NewInt(amountOutMapping[param.TokenAmountIn.AmountUsd]),
							},
						}, nil
					}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10,
					AmountOutUsd: 8,
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100,
					AmountOutUsd: 99,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1000,
					AmountOutUsd: 999,
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10000,
					AmountOutUsd: 991,
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 9999,
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 9999,
				},
			},
		},
		{
			name:     "it should continue return ErrNotEnoughSuccessTradeData because all trades are failed",
			tokenIn:  "token1",
			tokenOut: "token3",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				simulator.EXPECT().CalcAmountOut(gomock.Any()).
					DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
						return nil, errors.New("mock error")
					}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				0.0: {
					TokenIn:  "token1",
					TokenOut: "token3",
					Err:      ErrNotEnoughSuccessTradeData,
				},
			},
		},
		{
			name:     "it should continue generate trade data although all trade amount out are invalid",
			tokenIn:  "token1",
			tokenOut: "token3",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				simulator.EXPECT().CalcAmountOut(gomock.Any()).
					DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
						param := arg0.(poolpkg.CalcAmountOutParams)
						if param.TokenAmountIn.AmountUsd == float64(1) || param.TokenAmountIn.AmountUsd == float64(100) {
							return &poolpkg.CalcAmountOutResult{
								TokenAmountOut: &poolpkg.TokenAmount{
									Token:  "token3",
									Amount: big.NewInt(0),
								},
							}, nil
						}
						return nil, nil

					}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				0.0: {
					TokenIn:  "token1",
					TokenOut: "token3",
					Err:      ErrNotEnoughSuccessTradeData,
				},
			},
		},
		{
			name:     "it should generate extra points",
			tokenIn:  "token5",
			tokenOut: "token7",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				output := map[float64]int64{
					1:        1,
					2:        2,
					5:        4,
					10:       16,
					20:       16,
					50:       40,
					100:      100,
					1000:     999,
					10000:    999,
					100000:   999,
					1000000:  999,
					10000000: 999,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
					param := arg0.(poolpkg.CalcAmountOutParams)
					return &poolpkg.CalcAmountOutResult{
						TokenAmountOut: &poolpkg.TokenAmount{
							Token:  "token7",
							Amount: big.NewInt(output[param.TokenAmountIn.AmountUsd]),
						},
					}, nil
				}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  1,
					AmountOutUsd: 1,
				},
				2: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  2,
					AmountOutUsd: 2,
				},
				5: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  4,
					AmountOutUsd: 4,
				},
				10: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  16,
					AmountOutUsd: 16,
				},
				20: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  16,
					AmountOutUsd: 16,
				},
				50: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  16,
					AmountOutUsd: 40,
				},
				100: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  100,
					AmountOutUsd: 100,
				},
				1000: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  1000,
					AmountOutUsd: 999,
				},
				10000: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  10000,
					AmountOutUsd: 999,
				},
				100000: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  100000,
					AmountOutUsd: 999,
				},
				1000000: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  1000000,
					AmountOutUsd: 999,
				},
				10000000: {
					TokenIn:      "token5",
					TokenOut:     "token7",
					AmountInUsd:  10000000,
					AmountOutUsd: 999,
				},
			},
		},
		{
			name:     "it should return correct data when token out has no price",
			tokenIn:  "token1",
			tokenOut: "token8",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				output := map[float64]int64{
					1:       1,
					10:      16,
					100:     100,
					1000:    999,
					10000:   999,
					100000:  999,
					1000000: 999,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
					param := arg0.(poolpkg.CalcAmountOutParams)
					return &poolpkg.CalcAmountOutResult{
						TokenAmountOut: &poolpkg.TokenAmount{
							Token:  "token8",
							Amount: big.NewInt(output[param.TokenAmountIn.AmountUsd]),
						},
					}, nil
				}).Times(MIN_DATA_POINT_NUMBER_DEFAULT + 1)

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  1,
					AmountOutUsd: 0,
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  16,
					AmountOutUsd: 0,
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  100,
					AmountOutUsd: 0,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  1000,
					AmountOutUsd: 0,
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  10000,
					AmountOutUsd: 0,
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  100000,
					AmountOutUsd: 0,
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token8",
					AmountInUsd:  1000000,
					AmountOutUsd: 0,
				},
			},
		},
		{
			name:     "it should generate start amount because token in has no price",
			tokenIn:  "token8",
			tokenOut: "token2",
			prepare: func(ctrl *gomock.Controller, config TradeDataGeneratorConfig) poolpkg.IPoolSimulator {
				simulator := poolMocks.NewMockIPoolSimulator(ctrl)
				simulator.EXPECT().GetType().Return("uniswap").AnyTimes()
				simulator.EXPECT().GetExchange().Return("pancake").AnyTimes()
				simulator.EXPECT().GetAddress().Return("0xabc").AnyTimes()
				output := map[float64]int64{
					2:        2,
					20:       19,
					200:      199,
					2000:     1990,
					20000:    2999,
					200000:   9999,
					2000000:  9999,
					20000000: 9999,
				}
				simulator.EXPECT().CalcAmountOut(gomock.Any()).DoAndReturn(func(arg0 interface{}) (*poolpkg.CalcAmountOutResult, error) {
					param := arg0.(poolpkg.CalcAmountOutParams)
					if param.TokenOut == "token8" {
						return &poolpkg.CalcAmountOutResult{
							TokenAmountOut: &poolpkg.TokenAmount{
								Token:  "token8",
								Amount: big.NewInt(2),
							},
						}, nil
					}
					amountIn, _ := param.TokenAmountIn.Amount.Float64()
					return &poolpkg.CalcAmountOutResult{
						TokenAmountOut: &poolpkg.TokenAmount{
							Token:  "token2",
							Amount: big.NewInt(output[amountIn]),
						},
					}, nil
				}).AnyTimes()

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				2: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 4,
				},
				20: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 38,
				},
				200: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 398,
				},
				2000: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 3980,
				},
				20000: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 5998,
				},
				200000: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 19998,
				},
				2000000: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 19998,
				},
				20000000: {
					TokenIn:      "token8",
					TokenOut:     "token2",
					AmountOutUsd: 19998,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config := TradeDataGeneratorConfig{
				MaxDataPointNumber:          7,
				WhitelistedTokenSet:         map[string]bool{"token1": true, "token2": true, "token3": true, "token4": true, "token5": true, "token6": true},
				InvalidPriceImpactThreshold: 10.0,
			}
			generator := NewTradeDataGenerator(nil, nil, nil, nil, nil, nil, &config)
			poolSimulator := test.prepare(ctrl, config)

			result := generator.generateTradeData(context.TODO(), test.tokenIn, test.tokenOut, tokens, prices, poolSimulator, nil, valueobject.WHITELIST_WHITELIST)
			assert.Equal(t, len(test.expectedTradeData), len(result))
			for _, res := range result {
				var expected TradeData
				if prices[test.tokenIn] != nil {
					expected = test.expectedTradeData[res.AmountInUsd]
				} else {
					amountIn, _ := strconv.ParseFloat(res.AmountIn, 64)
					expected = test.expectedTradeData[amountIn]
				}

				assert.Equal(t, expected.AmountOutUsd, res.AmountOutUsd)
				assert.Equal(t, expected.TokenIn, res.TokenIn)
				assert.Equal(t, expected.TokenOut, res.TokenOut)
				if prices[test.tokenIn] == nil {
					assert.True(t, expected.AmountInUsd == 0.0)
				}
				if expected.Err != nil {
					assert.Equal(t, expected.Err.Error(), res.Err.Error())
				}
			}
		})
	}
}

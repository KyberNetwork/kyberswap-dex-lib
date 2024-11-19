package indexpools

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolMocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/pool"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTradeDataGenerator_GenerateTradeData(t *testing.T) {
	tokens := map[string]*entity.Token{
		"token1": {
			Address: "token1",
		},
		"token2": {
			Address: "token2",
		},
		"token3": {
			Address: "token3",
		},
		"token4": {
			Address: "token4",
		},
		"token5": {
			Address: "token5",
		},
		"token6": {
			Address: "token6",
		},
	}
	prices := map[string]*price{
		"token1": {
			buyPrice:  1,
			sellPrice: 1.5,
		},
		"token2": {
			buyPrice:  0,
			sellPrice: 2,
		},
		"token3": {
			buyPrice:  1,
			sellPrice: 5,
		},
		"token4": {
			buyPrice:  15,
			sellPrice: 12,
		},
		"token5": {
			buyPrice:  10,
			sellPrice: 15,
		},
		"token6": {
			buyPrice:  8,
			sellPrice: 6,
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
				simulator.EXPECT().CalcAmountOut(gomock.Any()).Return(
					&poolpkg.CalcAmountOutResult{
						TokenAmountOut: &poolpkg.TokenAmount{
							Token:  "token2",
							Amount: big.NewInt(100),
						},
					}, nil).Times(config.MaxDataPointNumber + 1)

				return simulator
			},
			expectedTradeData: map[float64]TradeData{
				1: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1,
					AmountOutUsd: 200,
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10,
					AmountOutUsd: 200,
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  100,
					AmountOutUsd: 200,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1000,
					AmountOutUsd: 200,
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10000,
					AmountOutUsd: 200,
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  100000,
					AmountOutUsd: 200,
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  1000000,
					AmountOutUsd: 200,
				},
				10000000: {
					TokenIn:      "token1",
					TokenOut:     "token2",
					AmountInUsd:  10000000,
					AmountOutUsd: 200,
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
			name:     "it should continue generate trade data although all trades are failed",
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
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1000,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10000,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 0,
					Err:          errors.New("mock error"),
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
				1: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1,
					AmountOutUsd: 0,
				},
				10: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10,
					AmountOutUsd: 0,
					Err:          fmt.Errorf("calcAmountOut error %v amountOut <nil>", ErrAmountOutNotValid),
				},
				100: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100,
					AmountOutUsd: 0,
				},
				1000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  1000,
					AmountOutUsd: 0,
					Err:          fmt.Errorf("calcAmountOut error %v amountOut <nil>", ErrAmountOutNotValid),
				},
				10000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  10000,
					AmountOutUsd: 0,
					Err:          fmt.Errorf("calcAmountOut error %v amountOut <nil>", ErrAmountOutNotValid),
				},
				100000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 0,
					Err:          fmt.Errorf("calcAmountOut error %v amountOut <nil>", ErrAmountOutNotValid),
				},
				1000000: {
					TokenIn:      "token1",
					TokenOut:     "token3",
					AmountInUsd:  100000,
					AmountOutUsd: 0,
					Err:          fmt.Errorf("calcAmountOut error %v amountOut <nil>", ErrAmountOutNotValid),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config := TradeDataGeneratorConfig{
				MaxDataPointNumber: 7,
			}
			generator := NewTradeDataGenerator(nil, nil, nil, nil, nil, nil, nil, config)
			poolSimulator := test.prepare(ctrl, config)

			result := generator.generateTradeData(context.TODO(), test.tokenIn, test.tokenOut, tokens, prices, poolSimulator, nil)
			assert.Equal(t, len(test.expectedTradeData), len(result))
			for _, res := range result {
				expected := test.expectedTradeData[res.AmountInUsd]
				assert.Equal(t, expected.AmountOutUsd, res.AmountOutUsd)
				assert.Equal(t, expected.TokenIn, res.TokenIn)
				assert.Equal(t, expected.TokenOut, res.TokenOut)
				if expected.Err != nil {
					assert.Equal(t, expected.Err.Error(), res.Err.Error())

				}
			}
		})
	}
}

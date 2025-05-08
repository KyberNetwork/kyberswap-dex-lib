package alphafee

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	dexlibEntity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexlibValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/stretchr/testify/assert"
)

func TestAlphaFeeV2Calculation(t *testing.T) {
	tokenIDs := []string{"a", "b", "c"}

	alphaFeeSource := dexlibValueObject.ExchangeKyberPMM
	nonAlphaFeeSource := dexlibValueObject.ExchangeUniSwapV3

	defaultAlphaFeeConfig := valueobject.AlphaFeeConfig{
		ReductionConfig: valueobject.AlphaFeeReductionConfig{
			ReductionFactorInBps:        map[string]float64{string(alphaFeeSource): 10000},
			MaxThresholdPercentageInBps: 8000,
			MinDifferentThresholdBps:    0,
			MinDifferentThresholdUSD:    0.001,
		},
	}

	pools := map[string]dexlibPool.IPoolSimulator{
		"tshared_rate_1-1_a_b": test.NewFixRatePoolWithID("tshared_rate_1-1_a_b", "a", "b", 1.0, alphaFeeSource),

		"t1_rate0.9_a_b": test.NewFixRatePoolWithID("t1_rate0.9_a_b", "a", "b", 0.9, alphaFeeSource),
		"t1_rate0.9_b_c": test.NewFixRatePoolWithID("t1_rate0.9_b_c", "b", "c", 0.9, alphaFeeSource),

		"t2_rate1-1_a_b": test.NewFixRatePoolWithID("t2_rate1-1_a_b", "a", "b", 1.0, alphaFeeSource),
		"t2_rate2-3_b_c": test.NewFixRatePoolWithID("t2_rate2-3_b_c", "b", "c", 1.5, alphaFeeSource),
		"t2_rate3-4_c_d": test.NewFixRatePoolWithID("t2_rate3-4_c_d", "c", "d", 1.33, alphaFeeSource),

		"t3_rate2-3_b_c": test.NewFixRatePoolWithID("t3_rate2-3_b_c", "b", "c", 1.5, nonAlphaFeeSource),

		"t4_rate1-1_a_b":   test.NewFixRatePoolWithID("t4_rate1-1_a_b", "a", "b", 1.0, nonAlphaFeeSource),
		"t4_rate1-1_b_c":   test.NewFixRatePoolWithID("t4_rate1-1_b_c", "b", "c", 1.0, alphaFeeSource),
		"t4_rate1-1_a_b#2": test.NewFixRatePoolWithID("t4_rate1-1_a_b#2", "a", "b", 1.0, alphaFeeSource),
		"t4_rate1-1_b_c#2": test.NewFixRatePoolWithID("t4_rate1-1_b_c#2", "b", "c", 1.0, nonAlphaFeeSource),
	}

	tests := []struct {
		name             string
		bestRoute        *finderCommon.ConstructRoute
		bestAmmRoute     *finderCommon.ConstructRoute
		config           valueobject.AlphaFeeConfig
		expectedAlphaFee *routerEntity.AlphaFeeV2
		expectedError    error
	}{
		{
			name: "[t1] swap $1000 through 2 pools, rate 0.9 per pool, taking $30 alpha fee",
			bestRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(810_000_000),
				AmountOutPrice: 810,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:    big.NewInt(1000_000_000),
						AmountOut:   big.NewInt(810_000_000),
						PoolsOrder:  []string{"t1_rate0.9_a_b", "t1_rate0.9_b_c"},
						TokensOrder: []string{"a", "b", "c"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(780_000_000),
				AmountOutPrice: 780,
			},
			config: defaultAlphaFeeConfig,
			expectedAlphaFee: &routerEntity.AlphaFeeV2{
				AMMAmount: big.NewInt(780_000_000),
				SwapReductions: []routerEntity.AlphaFeeV2SwapReduction{
					{
						ExecutedId:   0,
						PoolAddress:  "t1_rate0.9_a_b",
						TokenIn:      "a",
						TokenOut:     "b",
						ReduceAmount: big.NewInt(16823914),
					},
					{
						ExecutedId:   1,
						PoolAddress:  "t1_rate0.9_b_c",
						TokenIn:      "b",
						TokenOut:     "c",
						ReduceAmount: big.NewInt(14858478),
					},
				},
			},
			expectedError: nil,
		},

		{
			name: "[t2] swap $100 through 3 pools, taking $30 alpha fee",
			bestRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(200_000_000),
				AmountOutPrice: 200,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:    big.NewInt(100_000_000),
						AmountOut:   big.NewInt(200_000_000),
						PoolsOrder:  []string{"t2_rate1-1_a_b", "t2_rate2-3_b_c", "t2_rate3-4_c_d"},
						TokensOrder: []string{"a", "b", "c", "d"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(170_000_000),
				AmountOutPrice: 180,
			},
			config: defaultAlphaFeeConfig,
			expectedAlphaFee: &routerEntity.AlphaFeeV2{
				AMMAmount: big.NewInt(170_000_000),
				SwapReductions: []routerEntity.AlphaFeeV2SwapReduction{
					{
						ExecutedId:   0,
						PoolAddress:  "t2_rate1-1_a_b",
						TokenIn:      "a",
						TokenOut:     "b",
						ReduceAmount: big.NewInt(5_273_177),
					},
					{
						ExecutedId:   1,
						PoolAddress:  "t2_rate2-3_b_c",
						TokenIn:      "b",
						TokenOut:     "c",
						ReduceAmount: big.NewInt(7_492_669),
					},
					{
						ExecutedId:   2,
						PoolAddress:  "t2_rate3-4_c_d",
						TokenIn:      "c",
						TokenOut:     "d",
						ReduceAmount: big.NewInt(9_439_764),
					},
				},
			},
			expectedError: nil,
		},

		{
			name: "[t3] swap $100 through 3 pools, taking $30 alpha fee, only 1st and 3rd pools are alpha fee sources",
			bestRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(200_000_000),
				AmountOutPrice: 200,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:    big.NewInt(100_000_000),
						AmountOut:   big.NewInt(200_000_000),
						PoolsOrder:  []string{"t2_rate1-1_a_b", "t3_rate2-3_b_c", "t2_rate3-4_c_d"},
						TokensOrder: []string{"a", "b", "c", "d"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(170_000_000),
				AmountOutPrice: 170,
			},
			config: defaultAlphaFeeConfig,
			expectedAlphaFee: &routerEntity.AlphaFeeV2{
				AMMAmount: big.NewInt(170_000_000),
				SwapReductions: []routerEntity.AlphaFeeV2SwapReduction{
					{
						ExecutedId:   0,
						PoolAddress:  "t2_rate1-1_a_b",
						TokenIn:      "a",
						TokenOut:     "b",
						ReduceAmount: big.NewInt(7_804_556),
					},
					{
						ExecutedId:   2,
						PoolAddress:  "t2_rate3-4_c_d",
						TokenIn:      "c",
						TokenOut:     "d",
						ReduceAmount: big.NewInt(14_354_912),
					},
				},
			},
			expectedError: nil,
		},

		{
			name: "[t4] alpha fee taking through 2 paths",
			bestRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(400_000_000),
				AmountOutPrice: 400,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:    big.NewInt(200_000_000),
						AmountOut:   big.NewInt(200_000_000),
						PoolsOrder:  []string{"t4_rate1-1_a_b", "t4_rate1-1_b_c"},
						TokensOrder: []string{"a", "b", "c"},
					},
					{
						AmountIn:    big.NewInt(200_000_000),
						AmountOut:   big.NewInt(200_000_000),
						PoolsOrder:  []string{"t4_rate1-1_a_b#2", "t4_rate1-1_b_c#2"},
						TokensOrder: []string{"a", "b", "c"},
					},
				},
			},
			bestAmmRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(350_000_000),
				AmountOutPrice: 350,
			},
			config: defaultAlphaFeeConfig,
			expectedAlphaFee: &routerEntity.AlphaFeeV2{
				AMMAmount: big.NewInt(350_000_000),
				SwapReductions: []routerEntity.AlphaFeeV2SwapReduction{
					{
						ExecutedId:   1,
						PoolAddress:  "t4_rate1-1_b_c",
						TokenIn:      "b",
						TokenOut:     "c",
						ReduceAmount: big.NewInt(25_000_000),
					},
					{
						ExecutedId:   2,
						PoolAddress:  "t4_rate1-1_a_b#2",
						TokenIn:      "a",
						TokenOut:     "b",
						ReduceAmount: big.NewInt(25_000_000),
					},
				},
			},
			expectedError: nil,
		},

		{
			name: "[t5] BestAMMRoute is not available, handle MaxThresholdPercentageInBps",
			bestRoute: &finderCommon.ConstructRoute{
				AmountOut:      big.NewInt(1000_000_000),
				AmountOutPrice: 1000,
				Paths: []*finderCommon.ConstructPath{
					{
						AmountIn:    big.NewInt(1000_000_000),
						AmountOut:   big.NewInt(1000_000_000),
						PoolsOrder:  []string{"tshared_rate_1-1_a_b"},
						TokensOrder: []string{"a", "b"},
					},
				},
			},
			bestAmmRoute: nil,
			config:       defaultAlphaFeeConfig,
			expectedAlphaFee: &routerEntity.AlphaFeeV2{
				AMMAmount: big.NewInt(800_000_000),
				SwapReductions: []routerEntity.AlphaFeeV2SwapReduction{
					{
						ExecutedId:   0,
						PoolAddress:  "tshared_rate_1-1_a_b",
						TokenIn:      "a",
						TokenOut:     "b",
						ReduceAmount: big.NewInt(200_000_000),
					},
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("Running test:", tt.name)

			alphaFeeV2Calculation := NewAlphaFeeV2Calculation(
				tt.config,
				finderCommon.DefaultCustomFuncs,
			)

			prices := map[string]float64{}
			tokens := map[string]dexlibEntity.Token{}

			for _, tokenID := range tokenIDs {
				tokens[tokenID] = dexlibEntity.Token{
					Address:  tokenID,
					Symbol:   tokenID,
					Decimals: 6,
				}
				prices[tokenID] = 1.0
			}

			simulatorBucket := finderCommon.NewSimulatorBucket(pools, nil, finderCommon.DefaultCustomFuncs)

			params := AlphaFeeParams{
				BestRoute:           tt.bestRoute,
				BestAmmRoute:        tt.bestAmmRoute,
				Prices:              prices,
				Tokens:              tokens,
				PoolSimulatorBucket: simulatorBucket,
			}

			res, err := alphaFeeV2Calculation.Calculate(context.Background(), params)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NotNil(t, res)
				assert.Equal(t, tt.expectedAlphaFee.AMMAmount, res.AMMAmount)
				assert.Equal(t, len(tt.expectedAlphaFee.SwapReductions), len(res.SwapReductions))
				for i, expectedSwapReduction := range tt.expectedAlphaFee.SwapReductions {
					fmt.Println(res.SwapReductions[i])
					assert.Equal(t, expectedSwapReduction.ExecutedId, res.SwapReductions[i].ExecutedId)
					assert.Equal(t, expectedSwapReduction.PoolAddress, res.SwapReductions[i].PoolAddress)
					assert.Equal(t, expectedSwapReduction.TokenIn, res.SwapReductions[i].TokenIn)
					assert.Equal(t, expectedSwapReduction.TokenOut, res.SwapReductions[i].TokenOut)
					assert.Equal(t, expectedSwapReduction.ReduceAmount, res.SwapReductions[i].ReduceAmount)
				}
			}
		})
	}
}

func TestAlphaFeeV2_GetFairPrice(t *testing.T) {
	defaultAlphaFeeConfig := valueobject.AlphaFeeConfig{
		WhitelistPrices: map[string]bool{
			"tokenIn": true,
		},
	}

	alphaFeeV2Calculation := NewAlphaFeeV2Calculation(
		defaultAlphaFeeConfig,
		finderCommon.DefaultCustomFuncs,
	)

	param := AlphaFeeParams{
		Prices: map[string]float64{
			"tokenIn":  1.0,
			"tokenOut": 100.0,
		},
		Tokens: map[string]dexlibEntity.Token{
			"tokenIn": {
				Address:  "tokenIn",
				Symbol:   "tokenIn",
				Decimals: 0,
			},
			"tokenOut": {
				Address:  "tokenOut",
				Symbol:   "tokenOut",
				Decimals: 1,
			},
		},
	}

	tokenIn := "tokenIn"
	tokenOut := "tokenOut"
	amountIn := big.NewInt(100)
	amountOut := big.NewInt(2000)
	alphaFeeAmount := big.NewInt(200)

	price := alphaFeeV2Calculation.GetFairPrice(
		context.Background(),
		tokenIn, tokenOut,
		param.Prices[tokenIn], param.Prices[tokenOut],
		param.Tokens[tokenIn].Decimals, param.Tokens[tokenOut].Decimals,
		amountIn, amountOut, alphaFeeAmount,
	)

	// 100 tokenIn receives 200 tokenOut (notice that tokenOut has 1 decimal).
	// tokenInPrice = 1.0 -> amountInUsd = 100
	// tokenOutPrice (non-whitelisted) = 100.0 -> amountOutUsd = 20_000
	// With fair price, amountOutUsd should be = 100
	// -> tokenOutPrice = 100.0 * 100 / 20_000 = 0.5
	// -> alphaFeeAmountUsd = 20 * 0.5 = 10
	assert.Equal(t, float64(10), price)
}

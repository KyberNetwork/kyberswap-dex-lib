package safetyquote

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestSafetyQuoteReduction_rand(t *testing.T) {
	t.Parallel()
	var r *Reduction
	var lowOk, highOk bool
	for i := range 99 {
		f := r.rand(string(rune(i)))
		if f < 0.3 {
			lowOk = true
		} else if f > 0.7 {
			highOk = true
		}
		if lowOk && highOk {
			return
		}
	}
	assert.Fail(t, "expected to generate some number lower than 0.3 and higher than 0.7")
}

func TestSafetyQuoteReduction_Reduce(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		amount               *pool.TokenAmount
		poolType             string
		result               pool.TokenAmount
		config               *valueobject.SafetyQuoteReductionConfig
		applyDeductionFactor bool
		clientId             string
		err                  error
	}{
		{
			name:     "Reduce safety quote amount with PMM pool, amount is not changed",
			poolType: pooltypes.PoolTypes.KyberPMM,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000036475678),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000036475678),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "testClient",
		},
		{
			name:     "Reduce safety quote amount with RFQ pool, amount is not changed",
			poolType: pooltypes.PoolTypes.LimitOrder,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1012336475678),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1012336475678),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "testClient",
		},
		{
			name:     "Reduce safety quote amount with RFQ pool, amount is not changed",
			poolType: pooltypes.PoolTypes.HashflowV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "testClient",
		},
		{
			name:     "Reduce safety quote amount with Stable pairs, amount is reduced correctly",
			poolType: pooltypes.PoolTypes.UniswapV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000000),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000000),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "testClient",
		},
		{
			name:     "Reduce safety quote amount with Stable pairs, amount is reduced correctly",
			poolType: pooltypes.PoolTypes.UniswapV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: utils.NewBig10("12345678923455678999999999"),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: utils.NewBig10("12345678923455678999999999"),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "testClient",
		},
		{
			name:     "Should not reduce safety quote amount with Stable pairs because client is non-whitelist",
			poolType: pooltypes.PoolTypes.UniswapV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: utils.NewBig10("12345678923455678999999999"),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: utils.NewBig10("12345061639509506602827776"),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: false,
			clientId:             "nonWhitelist",
		},
		{
			name:     "Do not reduce safety quote amount when ExcludeOneSwapEnable is false",
			poolType: pooltypes.PoolTypes.UniswapV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000000),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(1000000),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: false,
				Factor: map[string]float64{
					"Default":        0,
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: true,
			clientId:             "testClient",
		},
		{
			name:     "Reduce safety quote amount with Different Token Group, amount is reduced correctly",
			poolType: pooltypes.PoolTypes.UniswapV3,
			amount: &pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(100),
			},
			result: pool.TokenAmount{
				Token:  "0xabc",
				Amount: big.NewInt(99),
			},
			config: &valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: false,
				Factor: map[string]float64{
					"Default":    0,
					"Stable":     0,
					"Correlated": 1,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
				TokenGroupConfig: &valueobject.TokenGroupConfig{
					CorrelatedGroup1: map[string]bool{
						"0xabc": true,
						"0xdef": true,
					},
				},
			},
			applyDeductionFactor: true,
			clientId:             "nonWhitelist",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			safetyQuotingParams := types.SafetyQuotingParams{
				PoolType:       tc.poolType,
				TokenIn:        "0xabc",
				TokenOut:       "0xdef",
				Amount:         tc.amount.Amount,
				HasOnlyOneSwap: tc.applyDeductionFactor,
				ClientId:       tc.clientId,
			}

			safetyQuoteReduction := NewSafetyQuoteReduction(tc.config)
			res := safetyQuoteReduction.Reduce(safetyQuotingParams)

			assert.Equal(t, tc.result.Amount, res)
		})
	}
}

package safetyquote

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSafetyQuoteReduction_Reduce(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		amount               *pool.TokenAmount
		poolType             string
		result               pool.TokenAmount
		config               valueobject.SafetyQuoteReductionConfig
		excludeSafetyQuoting bool
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
				Amount: utils.NewBig10("12345061639509506216049999"),
			},
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: true,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
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
			config: valueobject.SafetyQuoteReductionConfig{
				ExcludeOneSwapEnable: false,
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: true,
			clientId:             "testClient",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			safetyQuoteReduction := NewSafetyQuoteReduction(tc.config)
			res := safetyQuoteReduction.Reduce(tc.amount,
				safetyQuoteReduction.GetSafetyQuotingRate(tc.poolType, tc.excludeSafetyQuoting), tc.clientId)

			assert.Equal(t, res.Token, tc.result.Token)
			assert.Equal(t, res.AmountUsd, tc.result.AmountUsd)
			assert.True(t, res.Amount.Cmp(tc.result.Amount) == 0)
		})
	}
}

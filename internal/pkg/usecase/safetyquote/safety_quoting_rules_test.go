package safetyquote

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
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
				Amount: big.NewInt(999950),
			},
			config: valueobject.SafetyQuoteReductionConfig{
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
				Amount: utils.NewBig10("12345061639509506216049999"),
			},
			config: valueobject.SafetyQuoteReductionConfig{
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
				Amount: utils.NewBig10("12345678923455678999999999"),
			},
			config: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"testClient", "testBetaClient"},
			},
			excludeSafetyQuoting: false,
			clientId:             "nonWhitelist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestSafetyQuoteReduction_ApplyConfig(t *testing.T) {
	testCases := []struct {
		name              string
		oldConfig         valueobject.SafetyQuoteReductionConfig
		newConfig         valueobject.SafetyQuoteReductionConfig
		expectedFactor    map[SafetyQuoteCategory]float64
		expectedWhitelist mapset.Set[string]
		err               error
	}{
		{
			name: "Should apply correct config when remote config was changed",
			oldConfig: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
				WhitelistedClient: []string{"oldClient1", "oldClient2"},
			},
			newConfig: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStable": 10,
					"Stable":         5.5,
				},
				WhitelistedClient: []string{"NEWCLIENT"},
			},
			expectedFactor: map[SafetyQuoteCategory]float64{
				StrictlyStable: 10,
				Stable:         5.5,
			},
			expectedWhitelist: mapset.NewSet("newclient"),
		},
		{
			name: "Should apply correct config when remote config was changed but contains invalid key",
			oldConfig: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         0.5,
				},
			},
			newConfig: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStableInvalid": 10,
					"Stable":                5.5,
					"AdditionalStable":      1.5,
				},
			},
			expectedFactor: map[SafetyQuoteCategory]float64{
				StrictlyStable: 0,
				Stable:         5.5,
			},
			expectedWhitelist: mapset.NewSet[string](),
		},
		{
			name: "Keep the old config because remote config is empty",
			oldConfig: valueobject.SafetyQuoteReductionConfig{
				Factor: map[string]float64{
					"StrictlyStable": 0,
					"Stable":         1.5,
				},
			},
			newConfig: valueobject.SafetyQuoteReductionConfig{},
			expectedFactor: map[SafetyQuoteCategory]float64{
				StrictlyStable: 0,
				Stable:         0.5,
			},
			expectedWhitelist: mapset.NewSet[string](),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			safetyQuoteReduction := NewSafetyQuoteReduction(tc.oldConfig)
			safetyQuoteReduction.ApplyConfig(tc.newConfig)
			assert.Equal(t, tc.expectedFactor, safetyQuoteReduction.deductionFactorInBps)
			assert.True(t, safetyQuoteReduction.whiteListClients.Equal(tc.expectedWhitelist))
		})
	}
}

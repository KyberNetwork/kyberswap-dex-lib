package setting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func mockHandleSuccess(w http.ResponseWriter, r *http.Request) {
	jsonResponse := ConfigResponse{
		Code:    0,
		Message: "Successfully",
		Data: ConfigResponseData{
			Hash: "xyz",
			Config: ConfigResponseDataConfig{
				AvailableSources: []valueobject.Source{"uniswap", "uniswapv3", "dmm"},
				WhitelistedTokens: []valueobject.WhitelistedToken{
					{
						Address:  "address1",
						Name:     "name1",
						Symbol:   "symbol1",
						Decimals: 18,
						CgkId:    "cgkId1",
					},
				},
				BlacklistedPools: []string{"0x00"},
				GetBestPoolsOptions: valueobject.GetBestPoolsOptions{
					DirectPoolsCount:                100,
					WhitelistPoolsCount:             500,
					TokenInPoolsCount:               200,
					TokenOutPoolCount:               200,
					AmplifiedTvlDirectPoolsCount:    50,
					AmplifiedTvlWhitelistPoolsCount: 200,
					AmplifiedTvlTokenInPoolsCount:   100,
					AmplifiedTvlTokenOutPoolCount:   100,
				},
				FinderOptions: valueobject.FinderOptions{
					MaxHops:                 3,
					DistributionPercent:     5,
					MaxPathsInRoute:         20,
					MaxPathsToGenerate:      5,
					MaxPathsToReturn:        200,
					MinPartUSD:              500,
					MinThresholdAmountInUSD: 0,
					MaxThresholdAmountInUSD: 100000000,

					HillClimbDistributionPercent: 1,
					HillClimbIteration:           2,
					HillClimbMinPartUSD:          500,
				},
				CacheConfig: valueobject.CacheConfig{
					DefaultTTL:           15,
					TTLByAmount:          nil,
					TTLByAmountUSDRange:  nil,
					PriceImpactThreshold: 5,
					ShrinkFuncName:       "decimal",
					ShrinkFuncPowExp:     0.7,
					ShrinkFuncLogPercent: 1.01,
					ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
						{
							ShrinkFuncName:     "logarithm",
							ShrinkFuncConstant: 1.5,
						},
						{
							ShrinkFuncName:     "logarithm",
							ShrinkFuncConstant: 2,
						},
						{
							ShrinkFuncName:     "logarithm",
							ShrinkFuncConstant: 2.5,
						},
					},
				},
				FeatureFlags: valueobject.FeatureFlags{
					IsGasEstimatorEnabled:      true,
					IsFaultyPoolDetectorEnable: true,
				},
				BlacklistedRecipients: []string{"0xaa"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse)
}

func TestGetConfigs(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch strings.TrimSpace(r.URL.Path) {
				case "/api/v1/configurations":
					mockHandleSuccess(w, r)
				default:
					http.NotFoundHandler().ServeHTTP(w, r)
				}
			},
		),
	)
	defer server.Close()

	s := NewRestRepository(server.URL + "/api/v1/configurations")

	result, err := s.GetConfigs(context.Background(), "aggregator-1", "")

	if err != nil {
		t.Errorf("TestGetConfigs failed, err: %v", err)
		return
	}

	want := valueobject.RemoteConfig{
		Hash:             "xyz",
		AvailableSources: []valueobject.Source{"uniswap", "uniswapv3", "dmm"},
		WhitelistedTokens: []valueobject.WhitelistedToken{
			{
				Address:  "address1",
				Name:     "name1",
				Symbol:   "symbol1",
				Decimals: 18,
				CgkId:    "cgkId1",
			},
		},
		BlacklistedPools: []string{"0x00"},
		GetBestPoolsOptions: valueobject.GetBestPoolsOptions{
			DirectPoolsCount:                100,
			WhitelistPoolsCount:             500,
			TokenInPoolsCount:               200,
			TokenOutPoolCount:               200,
			AmplifiedTvlDirectPoolsCount:    50,
			AmplifiedTvlWhitelistPoolsCount: 200,
			AmplifiedTvlTokenInPoolsCount:   100,
			AmplifiedTvlTokenOutPoolCount:   100,
		},
		FinderOptions: valueobject.FinderOptions{
			MaxHops:                 3,
			DistributionPercent:     5,
			MaxPathsInRoute:         20,
			MaxPathsToGenerate:      5,
			MaxPathsToReturn:        200,
			MinPartUSD:              500,
			MinThresholdAmountInUSD: 0,
			MaxThresholdAmountInUSD: 100000000,

			HillClimbDistributionPercent: 1,
			HillClimbIteration:           2,
			HillClimbMinPartUSD:          500,
		},
		CacheConfig: valueobject.CacheConfig{
			DefaultTTL:           15,
			TTLByAmount:          nil,
			TTLByAmountUSDRange:  nil,
			PriceImpactThreshold: 5,
			ShrinkFuncName:       "decimal",
			ShrinkFuncPowExp:     0.7,
			ShrinkFuncLogPercent: 1.01,
			ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
				{
					ShrinkFuncName:     "logarithm",
					ShrinkFuncConstant: 1.5,
				},
				{
					ShrinkFuncName:     "logarithm",
					ShrinkFuncConstant: 2,
				},
				{
					ShrinkFuncName:     "logarithm",
					ShrinkFuncConstant: 2.5,
				},
			},
		},
		FeatureFlags: valueobject.FeatureFlags{
			IsGasEstimatorEnabled:      true,
			IsFaultyPoolDetectorEnable: true,
		},
		BlacklistedRecipients: []string{"0xaa"},
	}

	assert.Equal(t, want, result)
}

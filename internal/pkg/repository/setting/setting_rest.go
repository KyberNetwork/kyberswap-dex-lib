package setting

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	QueryParamServiceCode = "serviceCode"
	QueryParamHash        = "hash"

	EmptyConfigHash = ""
)

type ConfigResponseDataConfig struct {
	AvailableSources        []valueobject.Source            `json:"availableSources"`
	UnscalableSources       []valueobject.Source            `json:"unscalableSources"`
	ExcludedSourcesByClient map[string][]valueobject.Source `json:"excludedSourcesByClient"`

	WhitelistedTokens     []valueobject.WhitelistedToken  `json:"whitelistedTokens"`
	BlacklistedPools      []string                        `json:"blacklistedPools"`
	FeatureFlags          valueobject.FeatureFlags        `json:"featureFlags"`
	Log                   valueobject.Log                 `json:"log"`
	GetBestPoolsOptions   valueobject.GetBestPoolsOptions `json:"getBestPoolsOptions"`
	FinderOptions         valueobject.FinderOptions       `json:"finderOptions"`
	PregenFinderOptions   valueobject.FinderOptions       `json:"pregenFinderOptions"`
	CacheConfig           valueobject.CacheConfig         `json:"cache"`
	FaultyPoolsConfig     valueobject.FaultyPoolsConfig   `json:"faultyPoolsConfig"`
	BlacklistedRecipients []string                        `json:"blacklistedRecipients"`

	TokensThresholdForOnchainPrice uint32                                 `json:"tokensThresholdForOnchainPrice"`
	RFQAcceptableSlippageFraction  int64                                  `json:"rfqAcceptableSlippageFraction"`
	SafetyQuoteReduction           valueobject.SafetyQuoteReductionConfig `json:"safetyQuoteReduction"`
	AFConfig                       valueobject.AlphaFeeConfig             `json:"afConfig"`
}

type ConfigResponseData struct {
	Hash   string                   `json:"hash"`
	Config ConfigResponseDataConfig `json:"config"`
}

type ConfigResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    ConfigResponseData
}

type RestRepository struct {
	Url    string
	client *resty.Client
}

func NewRestRepository(url string) *RestRepository {
	return &RestRepository{
		Url:    url,
		client: resty.New(),
	}
}

func (f *RestRepository) GetConfigs(_ context.Context, serviceCode string,
	currentHash string) (valueobject.RemoteConfig, error) {

	var (
		err     error
		resp    *resty.Response
		cfgResp ConfigResponse
	)

	if currentHash == EmptyConfigHash {
		resp, err = f.client.R().
			SetQueryParam(QueryParamServiceCode, serviceCode).
			SetResult(&cfgResp).
			Get(f.Url)
	} else {
		resp, err = f.client.R().
			SetQueryParam(QueryParamServiceCode, serviceCode).
			SetQueryParam(QueryParamHash, currentHash).
			SetResult(&cfgResp).
			Get(f.Url)
	}

	if err != nil {
		return valueobject.RemoteConfig{}, err
	}
	if f.hasError(resp) {
		return valueobject.RemoteConfig{}, fmt.Errorf("fetch remote config error cause by %s", string(resp.Body()))
	}
	if resp.StatusCode() == http.StatusNoContent {
		return valueobject.RemoteConfig{
			Hash: currentHash,
		}, nil
	}

	return valueobject.RemoteConfig{
		Hash:                    cfgResp.Data.Hash,
		AvailableSources:        cfgResp.Data.Config.AvailableSources,
		UnscalableSources:       cfgResp.Data.Config.UnscalableSources,
		ExcludedSourcesByClient: cfgResp.Data.Config.ExcludedSourcesByClient,
		WhitelistedTokens:       cfgResp.Data.Config.WhitelistedTokens,
		BlacklistedPools:        cfgResp.Data.Config.BlacklistedPools,
		FeatureFlags:            cfgResp.Data.Config.FeatureFlags,
		Log:                     cfgResp.Data.Config.Log,
		GetBestPoolsOptions:     cfgResp.Data.Config.GetBestPoolsOptions,
		FinderOptions:           cfgResp.Data.Config.FinderOptions,
		PregenFinderOptions:     cfgResp.Data.Config.PregenFinderOptions,
		CacheConfig:             cfgResp.Data.Config.CacheConfig,
		BlacklistedRecipients:   cfgResp.Data.Config.BlacklistedRecipients,
		FaultyPoolsConfig:       cfgResp.Data.Config.FaultyPoolsConfig,
		SafetyQuoteReduction:    cfgResp.Data.Config.SafetyQuoteReduction,

		TokensThresholdForOnchainPrice: cfgResp.Data.Config.TokensThresholdForOnchainPrice,
		RFQAcceptableSlippageFraction:  cfgResp.Data.Config.RFQAcceptableSlippageFraction,
		AlphaFeeConfig:                 cfgResp.Data.Config.AFConfig,
	}, nil
}

func (f *RestRepository) hasError(resp *resty.Response) bool {
	return resp.StatusCode() >= 400 || resp.StatusCode() < 200
}

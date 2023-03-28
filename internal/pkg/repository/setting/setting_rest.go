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
	EnabledDexes      []valueobject.Dex              `json:"enabledDexes"`
	WhitelistedTokens []valueobject.WhitelistedToken `json:"whitelistedTokens"`
	BlacklistedPools  []string                       `json:"blacklistedPools"`
	FeatureFlags      valueobject.FeatureFlags       `json:"featureFlags"`
	Log               valueobject.Log                `json:"log"`
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

func (f *RestRepository) GetConfigs(_ context.Context, serviceCode string, currentHash string) (valueobject.RemoteConfig, error) {

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
		Hash:              cfgResp.Data.Hash,
		EnabledDexes:      cfgResp.Data.Config.EnabledDexes,
		WhitelistedTokens: cfgResp.Data.Config.WhitelistedTokens,
		BlacklistedPools:  cfgResp.Data.Config.BlacklistedPools,
		FeatureFlags:      cfgResp.Data.Config.FeatureFlags,
		Log:               cfgResp.Data.Config.Log,
	}, nil
}

func (f *RestRepository) hasError(resp *resty.Response) bool {
	return resp.StatusCode() >= 400 || resp.StatusCode() < 200
}

package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
)

const (
	pathQuote    = "api/rfq/firm"
	headerApiKey = "x-apikey"
)

var (
	ErrRFQFailed = errors.New("rfq failed")
)

type HTTPClient struct {
	config *dexalot.HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *dexalot.HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader(headerApiKey, config.APIKey)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) Quote(ctx context.Context, params dexalot.FirmQuoteParams,
	upscalePercent int) (dexalot.FirmQuoteResult, error) {
	// token address case-sensitive
	req := c.client.R().
		SetContext(ctx).
		// the SellTokens address must follow the HEX format
		SetBody(map[string]interface{}{
			dexalot.ParamsChainID:     params.ChainID,
			dexalot.ParamsTakerAsset:  common.HexToAddress(params.TakerAsset).Hex(),
			dexalot.ParamsMakerAsset:  common.HexToAddress(params.MakerAsset).Hex(),
			dexalot.ParamsTakerAmount: params.TakerAmount,
			dexalot.ParamsUserAddress: params.UserAddress,
			dexalot.ParamsExecutor:    params.Executor,
		})
	var result dexalot.FirmQuoteResult
	var fail dexalot.FirmQuoteFail
	resp, err := req.SetResult(&result).SetError(&fail).Post(pathQuote)
	if err != nil {
		return dexalot.FirmQuoteResult{}, err
	}

	if !resp.IsSuccess() || fail.Failed() {
		klog.WithFields(ctx, klog.Fields{
			"client":       dexalot.DexType,
			"dexalot_resp": fail,
		}).Error("quote failed")
		return dexalot.FirmQuoteResult{}, ErrRFQFailed
	}

	return result, nil
}

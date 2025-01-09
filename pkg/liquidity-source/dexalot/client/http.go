package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
)

const (
	pathQuote    = "api/rfq/firm"
	headerApiKey = "x-apikey"
)

var (
	ErrRFQFailed = errors.New("rfq failed")
)

type client struct {
	restyClient *resty.Client
}

func NewClient(config *dexalot.HTTPClientConfig) *client {
	restyClient := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader(headerApiKey, config.APIKey)

	return &client{
		restyClient: restyClient,
	}
}

func (c *client) Quote(ctx context.Context, params dexalot.FirmQuoteParams, upscalePercent int) (dexalot.FirmQuoteResult, error) {
	// token address case-sensitive
	req := c.restyClient.R().
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

	respBytes := resp.Body()
	_ = json.Unmarshal(respBytes, &result)
	_ = json.Unmarshal(respBytes, &fail)

	if !resp.IsSuccess() || fail.Failed() {
		logger.
			WithFields(logger.Fields{"dexalot_resp": string(respBytes)}).
			Error("dexalot rfq failed")
		return dexalot.FirmQuoteResult{}, ErrRFQFailed
	}

	return result, nil
}

package client

import (
	"context"
	"encoding/json"
	"errors"

	bebop "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
)

const (
	headerApiKey = "apiKey"

	pathQuote = "v2/quote"

	errMsgThrottled           = "ThrottlerException: Too Many Requests"
	errMsgInternalServerError = "Internal server error"
	errMsgBadRequest          = "Bad Request"
	errMsgAllPricerFailed     = "All pricer failed"
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrRFQRateLimit           = errors.New("rfq: rate limited")
	ErrRFQInternalServerError = errors.New("rfq: internal server error")
	ErrRFQBadRequest          = errors.New("rfq: bad request")
	ErrRFQAllPricerFailed     = errors.New("rfq: all pricer failed")
)

type HTTPClient struct {
	config *bebop.HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *bebop.HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeaderVerbatim(headerApiKey, config.APIKey)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) Quote(ctx context.Context, params bebop.QuoteParams) (bebop.QuoteResult, error) {
	// token address case-sensitive
	req := c.client.R().
		SetContext(ctx).
		SetQueryParam(bebop.ParamsSellTokens, common.HexToAddress(params.SellTokens).Hex()).
		SetQueryParam(bebop.ParamsBuyTokens, common.HexToAddress(params.BuyTokens).Hex()).
		SetQueryParam(bebop.ParamsSellAmounts, params.SellAmounts).
		SetQueryParam(bebop.ParamsTakerAddress, params.TakerAddress).
		SetQueryParam(bebop.ParamsReceiverAddress, params.ReceiverAddress).
		SetQueryParam(bebop.ParamsApproveType, "Standard").
		SetQueryParam(bebop.ParamsSkipValidation, "true").
		SetQueryParam(bebop.ParamsGasLess, "false")

	var result bebop.QuoteResult
	var fail bebop.QuoteFail
	resp, err := req.Get(pathQuote)
	if err != nil {
		return bebop.QuoteResult{}, err
	}
	bytes := resp.Body()
	_ = json.Unmarshal(bytes, &result)
	_ = json.Unmarshal(bytes, &fail)
	if !resp.IsSuccess() || fail.Failed() {
		return bebop.QuoteResult{}, parseRFQError(fail.Error.Message)
	}

	return result, nil
}

func parseRFQError(errorMessage string) error {
	switch errorMessage {
	case errMsgThrottled:
		return ErrRFQRateLimit
	case errMsgInternalServerError:
		return ErrRFQInternalServerError
	case errMsgBadRequest:
		return ErrRFQBadRequest
	case errMsgAllPricerFailed:
		return ErrRFQAllPricerFailed
	default:
		return ErrRFQFailed
	}
}

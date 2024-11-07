package client

import (
	"context"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
)

const (
	headerApiKey = "apiKey"

	pathCalldata = "v1/firm-quote/calldata"

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
	config *nativev1.HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *nativev1.HTTPClientConfig) *HTTPClient {
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

func (c *HTTPClient) Quote(ctx context.Context, params nativev1.QuoteParams) (nativev1.QuoteResult, error) {
	req := c.client.R().
		SetContext(ctx).
		SetQueryParams(params.ToMap())

	var result nativev1.QuoteResult
	resp, err := req.SetResult(&result).SetError(&result).Get(pathCalldata)
	if err != nil {
		return nativev1.QuoteResult{}, err
	}

	if !resp.IsSuccess() {
		return nativev1.QuoteResult{}, parseRFQError(result.Message)
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
		logger.WithFields(logger.Fields{"errMsg": errorMessage}).Error("unknown native rfq error")
		return ErrRFQFailed
	}
}

package client

import (
	"context"
	"strings"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
)

const (
	headerApiKey    = "apiKey"
	headerRequestId = "x-native-request-id"

	pathFirmQuote = "v1/firm-quote"

	errMsgExceedsMaxBalance = "quantity exceeds max balance"
	errMsgInvalidParameter  = "invalid parameter"

	errMsgThrottled           = "throttlerexception: too many requests"
	errMsgInternalServerError = "internal server error"
	errMsgBadRequest          = "bad request"
	errMsgAllPricerFailed     = "all pricer failed"
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
	resp, err := req.SetResult(&result).SetError(&result).Get(pathFirmQuote)
	if err != nil {
		return nativev1.QuoteResult{}, err
	}

	if !resp.IsSuccess() || !result.IsSuccess() {
		klog.WithFields(ctx, klog.Fields{
			"client":        nativev1.DexType,
			"response":      result,
			headerRequestId: resp.Header().Get(headerRequestId),
		}).Error("quote failed")
		return nativev1.QuoteResult{}, parseRFQError(result.Message)
	}

	return result, nil
}

func parseRFQError(errorMessage string) error {
	switch strings.ToLower(errorMessage) {
	case errMsgThrottled:
		return ErrRFQRateLimit
	case errMsgInternalServerError:
		return ErrRFQInternalServerError
	case errMsgBadRequest, errMsgInvalidParameter:
		return ErrRFQBadRequest
	case errMsgAllPricerFailed, errMsgExceedsMaxBalance:
		return ErrRFQAllPricerFailed
	default:
		return ErrRFQFailed
	}
}

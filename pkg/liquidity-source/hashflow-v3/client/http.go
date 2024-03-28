package client

import (
	"context"

	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const (
	authorizationHeaderKey = "Authorization"
	rfqPath                = "/taker/v3/rfq"

	errRFQRateLimitText              = "Rate Limit"
	errRFQBelowMinimumAmountText     = "Below minimum amount"
	errRFQExceedsSupportedAmountText = "Exceeds supported amounts"
	errRFQNoMakerSupportsText        = "No maker supports this request"
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrRFQRateLimit               = errors.New(errRFQRateLimitText)
	ErrRFQBelowMinimumAmount      = errors.New(errRFQBelowMinimumAmountText)
	ErrRFQExceedsSupportedAmounts = errors.New(errRFQExceedsSupportedAmountText)
	ErrRFQNoMakerSupports         = errors.New(errRFQNoMakerSupportsText)
)

type httpClient struct {
	client *resty.Client
	config *hashflowv3.HTTPClientConfig
}

func NewHTTPClient(config *hashflowv3.HTTPClientConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader(authorizationHeaderKey, config.APIKey)

	return &httpClient{
		client: client,
		config: config,
	}
}

func (c *httpClient) RFQ(ctx context.Context, params hashflowv3.QuoteParams) (hashflowv3.QuoteResult, error) {
	params.Source = c.config.Source
	req := c.client.R().SetContext(ctx).SetBody(params)

	var result hashflowv3.QuoteResult
	resp, err := req.SetResult(&result).Post(rfqPath)
	if err != nil {
		return hashflowv3.QuoteResult{}, err
	}

	if !resp.IsSuccess() || result.Status != "success" {
		logger.Errorf("hashflow rfq failed: status code(%d), body(%s)", resp.StatusCode(), resp.Body())
		return hashflowv3.QuoteResult{}, parseRFQError(result.Error.Message)
	}

	return result, nil
}

func parseRFQError(errorMessage string) error {
	switch errorMessage {
	case errRFQRateLimitText:
		return ErrRFQRateLimit
	case errRFQBelowMinimumAmountText:
		return ErrRFQBelowMinimumAmount
	case errRFQExceedsSupportedAmountText:
		return ErrRFQExceedsSupportedAmounts
	case errRFQNoMakerSupportsText:
		return ErrRFQNoMakerSupports
	default:
		return ErrRFQFailed
	}
}

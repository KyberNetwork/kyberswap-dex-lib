package client

import (
	"context"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
)

const (
	authorizationHeaderKey = "Authorization"
	rfqPath                = "/taker/v3/rfq"

	errRFQRateLimitText              = "Rate Limit"
	errRFQBelowMinimumAmountText     = "Below minimum amount"
	errRFQExceedsSupportedAmountText = "Exceeds supported amounts"
	errRFQNoMakerSupportsText        = "No maker supports this request"
	errRFQMarketsTooVolatile         = "Markets too volatile"
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrRFQRateLimit               = errors.New(errRFQRateLimitText)
	ErrRFQBelowMinimumAmount      = errors.New(errRFQBelowMinimumAmountText)
	ErrRFQExceedsSupportedAmounts = errors.New(errRFQExceedsSupportedAmountText)
	ErrRFQNoMakerSupports         = errors.New(errRFQNoMakerSupportsText)
	ErrRFQMarketsTooVolatile      = errors.New(errRFQMarketsTooVolatile)
)

type client struct {
	restyClient *resty.Client
	source      string
}

func NewClient(config *hashflowv3.HTTPClientConfig) *client {
	restyClient := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader(authorizationHeaderKey, config.APIKey)

	return &client{
		restyClient: restyClient,
		source:      config.Source,
	}
}

func (c *client) RFQ(ctx context.Context, params hashflowv3.QuoteParams) (hashflowv3.QuoteResult, error) {
	params.Source = c.source
	req := c.restyClient.R().SetContext(ctx).SetBody(params)

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
	case errRFQMarketsTooVolatile:
		return ErrRFQMarketsTooVolatile
	default:
		return ErrRFQFailed
	}
}

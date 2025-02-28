package client

import (
	"context"

	"github.com/KyberNetwork/kutils/klog"
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
	resp, err := req.SetResult(&result).SetError(&result).Post(rfqPath)
	if err != nil {
		return hashflowv3.QuoteResult{}, err
	}

	if !resp.IsSuccess() || result.Status != "success" {
		klog.WithFields(ctx, klog.Fields{
			"client":   hashflowv3.DexType,
			"response": result,
		}).Error("quote failed")
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

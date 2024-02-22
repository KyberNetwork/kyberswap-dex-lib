package client

import (
	"context"

	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const (
	authorizationHeaderKey = "Authorization"
	rfqPath                = "/taker/v3/rfq"
)

var (
	ErrRFQFailed    = errors.New("rfq failed")
	ErrInvalidValue = errors.New("invalid value")
)

type httpClient struct {
	client *resty.Client
	config *hashflowv3.HTTPClientConfig
}

func NewHTTPClient(config *hashflowv3.HTTPClientConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout).
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
		return hashflowv3.QuoteResult{}, nil
	}

	if !resp.IsSuccess() || result.Status != "success" {
		return hashflowv3.QuoteResult{}, errors.Wrapf(ErrRFQFailed, "status code(%d), body(%s)", resp.StatusCode(), resp.Body())
	}

	return hashflowv3.QuoteResult{}, nil
}

package client

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const (
	quoteEndpoint = "v1/rfq/quote"

	headerAPIKey = "X-API-KEY"
)

var (
	ErrQuoteFailed = errors.New("quote failed")
)

type HTTPClient struct {
	config *HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetHeader(headerAPIKey, config.APIKey).
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) Quote(ctx context.Context, params QuoteParams) (QuoteResult, error) {
	req := c.client.R().
		SetContext(ctx).
		SetBody(params)

	var result QuoteResult
	resp, err := req.SetResult(&result).Post(quoteEndpoint)
	if err != nil {
		return QuoteResult{}, err
	}

	if !resp.IsSuccess() {
		return QuoteResult{}, errors.Wrapf(ErrQuoteFailed, "status code(%d), body(%s)", resp.StatusCode(), resp.Body())
	}

	if !result.Success {
		return QuoteResult{}, ErrQuoteFailed
	}

	return result, nil
}

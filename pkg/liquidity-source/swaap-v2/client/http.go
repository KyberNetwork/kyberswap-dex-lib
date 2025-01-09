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

type client struct {
	restyClient *resty.Client
}

func NewClient(config *HTTPClientConfig) *client {
	restyClient := resty.New().
		SetHeader(headerAPIKey, config.APIKey).
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &client{
		restyClient: restyClient,
	}
}

func (c *client) Quote(ctx context.Context, params QuoteParams) (QuoteResult, error) {
	req := c.restyClient.R().
		SetContext(ctx).
		SetBody(params)

	var result QuoteResult
	resp, err := req.SetResult(&result).Post(quoteEndpoint)
	if err != nil {
		return QuoteResult{}, err
	}

	if !resp.IsSuccess() {
		return QuoteResult{}, errors.WithMessagef(ErrQuoteFailed, "[swaap-v2] status code(%d), body(%s)", resp.StatusCode(), resp.Body())
	}

	if !result.Success {
		return QuoteResult{}, ErrQuoteFailed
	}

	return result, nil
}

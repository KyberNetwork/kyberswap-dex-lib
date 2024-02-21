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
	baseChainType          = "evm" // Only fetch for evm chains for now
)

var (
	ErrRFQFailed    = errors.New("rfq failed")
	ErrInvalidValue = errors.New("invalid value")
)

type httpClient struct {
	client *resty.Client
	config *HTTPClientConfig
}

func NewHTTPClient(config *HTTPClientConfig) *httpClient {
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
	return hashflowv3.QuoteResult{}, nil
}

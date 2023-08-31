package client

import (
	"context"

	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
)

const (
	listTokens = "/kyberswap/v1/tokens"
	listPairs  = "/kyberswap/v1/pairs"
	listPrices = "/kyberswap/v1/prices"
)

type httpClient struct {
	client *resty.Client
	config *kyberpmm.HTTPConfig
}

func NewHTTPClient(config *kyberpmm.HTTPConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &httpClient{
		client: client,
		config: config,
	}
}

func (c *httpClient) ListTokens(ctx context.Context) (map[string]kyberpmm.TokenItem, error) {
	req := c.client.R().
		SetContext(ctx)

	var result kyberpmm.ListTokensResult
	resp, err := req.SetResult(&result).Get(listTokens)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, ErrListTokensFailed
	}

	return result.Tokens, nil
}

func (c *httpClient) ListPairs(ctx context.Context) (map[string]kyberpmm.PairItem, error) {
	req := c.client.R().
		SetContext(ctx)

	var result kyberpmm.ListPairsResult
	resp, err := req.SetResult(&result).Get(listPairs)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, ErrListPairsFailed
	}

	return result.Pairs, nil
}

func (c *httpClient) ListPriceLevels(ctx context.Context) (map[string]kyberpmm.PriceItem, error) {
	req := c.client.R().
		SetContext(ctx)

	var result kyberpmm.ListPriceLevelsResult
	resp, err := req.SetResult(&result).Get(listPrices)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, ErrListPriceLevelsFailed
	}

	return result.Prices, nil
}

package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/go-resty/resty/v2"
)

const (
	quotePath = "/rfq/quote"
	signPath  = "/rfq/sign"
)

var (
	ErrQuoteFailed = errors.New("quote failed")
	ErrSignFailed  = errors.New("sign failed")
)

type httpClient struct {
	client *resty.Client
	config clipper.HTTPClientConfig
}

func NewHTTPClient(config clipper.HTTPClientConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader("Authorization", "Basic "+config.BasicAuthKey)

	return &httpClient{
		client: client,
		config: config,
	}
}

func (c *httpClient) RFQ(ctx context.Context, params clipper.QuoteParams) (clipper.SignResponse, error) {
	// 1. Call quote endpoint
	req := c.client.R().SetContext(ctx).SetBody(params)

	var quoteRes clipper.QuoteResponse
	res, err := req.SetResult(&quoteRes).Post(quotePath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !res.IsSuccess() {
		return clipper.SignResponse{}, ErrQuoteFailed
	}

	// 2. Call sign endpoint with `quote_id` received from step 1
	req = c.client.R().SetContext(ctx).SetBody(clipper.SignParams{
		QuoteID:            quoteRes.ID,
		DestinationAddress: params.DestinationAddress,
		SenderAddress:      params.SenderAddress,
		NativeInput:        false,
		NativeOutput:       false,
	})

	var signRes clipper.SignResponse
	res, err = req.SetResult(&signRes).Post(signPath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !res.IsSuccess() {
		return clipper.SignResponse{}, ErrSignFailed
	}

	return signRes, nil
}

package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

const (
	quotePath = "/rfq/quote"
	signPath  = "/rfq/sign"

	errQuoteConflictText = "Quote conflicts with latest prices. Please request a new quote."
)

var (
	ErrQuoteFailed = errors.New("quote failed")
	ErrSignFailed  = errors.New("sign failed")

	ErrQuoteConflict = errors.New(errQuoteConflictText)
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
	var failRes clipper.FailResponse
	resp, err := req.SetResult(&quoteRes).SetError(&failRes).Post(quotePath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !resp.IsSuccess() {
		klog.WithFields(ctx, klog.Fields{
			"rfq.client": clipper.DexType,
			"rfq.resp":   util.MaxBytesToString(resp.Body(), 256),
			"rfq.status": resp.StatusCode(),
		}).Error("quote failed")
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
	resp, err = req.SetResult(&signRes).SetError(&failRes).Post(signPath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !resp.IsSuccess() {
		klog.WithFields(ctx, klog.Fields{
			"rfq.client": clipper.DexType,
			"rfq.resp":   util.MaxBytesToString(resp.Body(), 256),
			"rfq.status": resp.StatusCode(),
		}).Error("sign failed")
		return clipper.SignResponse{}, parseSignError(failRes.ErrorMessage)
	}

	return signRes, nil
}

func parseSignError(errorMessage string) error {
	switch errorMessage {
	case errQuoteConflictText:
		return ErrQuoteConflict
	default:
		return ErrSignFailed
	}
}

package client

import (
	"context"
	"errors"
	"strconv"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

const (
	quoteSignPath = "/rfq/v2/quote-sign/{chain_id}"

	errQuoteConflictText = "Quote conflicts with latest prices. Please request a new quote."
)

var (
	ErrQuoteSignFailed = errors.New("quote sign failed")

	ErrQuoteConflict = errors.New(errQuoteConflictText) // nolint:staticcheck
)

type httpClient struct {
	client *resty.Client
	config clipper.HTTPClientConfig
}

func NewHTTPClient(config clipper.HTTPClientConfig) *httpClient {
	if config.Client == nil {
		config.Client = resty.New()
	}
	config.Client.SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeader("x-api-key", config.BasicAuthKey)

	return &httpClient{
		client: config.Client,
		config: config,
	}
}

func (c *httpClient) RFQ(ctx context.Context, params clipper.QuoteParams) (clipper.SignResponse, error) {
	// 1. Call quote endpoint
	req := c.client.R().SetContext(ctx).
		SetPathParam("chain_id", strconv.Itoa(int(params.ChainID))).
		SetQueryParams(map[string]string{
			"time_in_seconds":     strconv.Itoa(params.TimeInSeconds),
			"input_amount":        params.InputAmount,
			"input_asset_symbol":  params.InputAssetSymbol,
			"output_asset_symbol": params.OutputAssetSymbol,
			"destination_address": params.DestinationAddress,
			"sender_address":      params.SenderAddress,
		})

	var quoteSignRes clipper.SignResponse
	var failRes clipper.FailResponse
	resp, err := req.SetResult(&quoteSignRes).SetError(&failRes).Get(quoteSignPath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !resp.IsSuccess() {
		klog.WithFields(ctx, klog.Fields{
			"rfq.client": clipper.DexType,
			"rfq.resp":   util.MaxBytesToString(resp.Body(), 256),
			"rfq.status": resp.StatusCode(),
		}).Error("quote failed")
		return clipper.SignResponse{}, parseSignError(failRes.ErrorMessage)
	}

	return quoteSignRes, nil
}

func parseSignError(errorMessage string) error {
	switch errorMessage {
	case errQuoteConflictText:
		return ErrQuoteConflict
	default:
		return ErrQuoteSignFailed
	}
}

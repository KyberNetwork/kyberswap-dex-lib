package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"

	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mx-trading"
	"github.com/go-resty/resty/v2"
)

const (
	orderEndpoint = "/order"

	errMsgOrderIsTooSmall = "order is too small"
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrOrderIsTooSmall = errors.New("rfq: order is too small")
)

type HTTPClient struct {
	client *resty.Client
	config *mxtrading.HTTPClientConfig
}

func NewHTTPClient(config *mxtrading.HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c HTTPClient) Quote(ctx context.Context, params mxtrading.OrderParams) (mxtrading.SignedOrderResult, error) {
	req := c.client.R().SetContext(ctx).SetBody(params)

	var result mxtrading.SignedOrderResult
	var errResult any
	resp, err := req.SetResult(&result).SetError(&errResult).Post(orderEndpoint)
	if err != nil {
		return mxtrading.SignedOrderResult{}, err
	}

	if !resp.IsSuccess() {
		return mxtrading.SignedOrderResult{}, parseOrderError(errResult)
	}

	return result, nil
}

func parseOrderError(errResult any) error {
	logger.Errorf("mx-trading rfq error: %v", errResult)

	switch errResult {
	case errMsgOrderIsTooSmall:
		return ErrOrderIsTooSmall
	default:
		logger.WithFields(logger.Fields{"body": errResult}).Errorf("unknown mx-trading rfq error")
		return ErrRFQFailed
	}
}

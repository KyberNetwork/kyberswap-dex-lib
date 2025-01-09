package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"

	"github.com/go-resty/resty/v2"

	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mx-trading"
)

const (
	orderEndpoint = "/order"

	errMsgOrderIsTooSmall = "order is too small"
)

var (
	ErrRFQFailed = errors.New("rfq failed")

	ErrOrderIsTooSmall = errors.New("rfq: order is too small")
)

type client struct {
	restyClient *resty.Client
}

func NewClient(config *mxtrading.HTTPClientConfig) *client {
	restyClient := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &client{
		restyClient: restyClient,
	}
}

func (c *client) Quote(ctx context.Context, params mxtrading.OrderParams) (mxtrading.SignedOrderResult, error) {
	req := c.restyClient.R().SetContext(ctx).SetBody(params)

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

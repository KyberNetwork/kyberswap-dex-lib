package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
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
		SetDebug(config.Debug).
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

	var failRes clipper.FailResponse

	var quoteRes clipper.QuoteResponse
	res, err := req.SetResult(&quoteRes).SetError(&failRes).Post(quotePath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !res.IsSuccess() {
		logger.WithFields(logger.Fields{
			"client":       clipper.DexType,
			"errorMessage": failRes.ErrorMessage,
			"errorType":    failRes.ErrorType,
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
	res, err = req.SetResult(&signRes).SetError(&failRes).Post(signPath)
	if err != nil {
		return clipper.SignResponse{}, err
	}

	if !res.IsSuccess() {
		logger.WithFields(logger.Fields{
			"client":       clipper.DexType,
			"errorMessage": failRes.ErrorMessage,
			"errorType":    failRes.ErrorType,
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

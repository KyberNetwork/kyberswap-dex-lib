package client

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	headerApiKey = "apiKey"

	pathFirmQuote = "v1/firm-quote"
)

var (
	ErrQuoteFailed = errors.New("quote failed")
)

type HTTPClient struct {
	config *nativev1.HTTPClientConfig
	client *resty.Client
}

func NewHTTPClient(config *nativev1.HTTPClientConfig) *HTTPClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount).
		SetHeaderVerbatim(headerApiKey, config.APIKey)

	return &HTTPClient{
		config: config,
		client: client,
	}
}

func (c *HTTPClient) Quote(ctx context.Context, params nativev1.QuoteParams) (nativev1.QuoteResult, error) {
	req := c.client.R().
		SetContext(ctx).
		SetQueryParam(nativev1.ParamsChain, nativev1.ChainById(valueobject.ChainID(params.ChainID))).
		SetQueryParam(nativev1.ParamsTokenIn, params.TokenIn).
		SetQueryParam(nativev1.ParamsTokenOut, params.TokenOut).
		SetQueryParam(nativev1.ParamsAmountWei, params.AmountWei).
		SetQueryParam(nativev1.ParamsFromAddress, params.FromAddress).
		SetQueryParam(nativev1.ParamsBeneficiaryAddress, params.BeneficiaryAddress).
		SetQueryParam(nativev1.ParamsToAddress, params.ToAddress).
		SetQueryParam(nativev1.ParamsExpiryTime, params.ExpiryTime).
		SetQueryParam(nativev1.ParamsSlippage, params.Slippage)

	var result nativev1.QuoteResult
	resp, err := req.SetResult(&result).Get(pathFirmQuote)
	if err != nil {
		return nativev1.QuoteResult{}, err
	}

	if !resp.IsSuccess() {
		return nativev1.QuoteResult{}, errors.Wrapf(ErrQuoteFailed, "status code(%d), body(%s)", resp.StatusCode(),
			resp.Body())
	}

	if !result.Success {
		return nativev1.QuoteResult{}, ErrQuoteFailed
	}

	return result, nil
}

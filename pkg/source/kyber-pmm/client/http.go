package client

import (
	"context"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
)

const (
	listTokensEndpoint = "/kyberswap/v1/tokens"
	listPairsEndpoint  = "/kyberswap/v1/pairs"
	listPricesEndpoint = "/kyberswap/v1/prices"
	firmEndpoint       = "/kyberswap/v1/firm"
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
	resp, err := req.SetResult(&result).Get(listTokensEndpoint)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.Wrapf(ErrListTokensFailed, "response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result.Tokens, nil
}

func (c *httpClient) ListPairs(ctx context.Context) (map[string]kyberpmm.PairItem, error) {
	req := c.client.R().
		SetContext(ctx)

	var result kyberpmm.ListPairsResult
	resp, err := req.SetResult(&result).Get(listPairsEndpoint)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.Wrapf(ErrListPairsFailed, "response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result.Pairs, nil
}

func (c *httpClient) ListPriceLevels(ctx context.Context) (kyberpmm.ListPriceLevelsResult, error) {
	req := c.client.R().
		SetContext(ctx)

	var result kyberpmm.ListPriceLevelsResult
	resp, err := req.SetResult(&result).Get(listPricesEndpoint)
	if err != nil {
		return result, err
	}

	if !resp.IsSuccess() {
		return result, errors.Wrapf(ErrListPriceLevelsFailed, "response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result, nil
}

func (c *httpClient) Firm(ctx context.Context, params kyberpmm.FirmRequestParams) (kyberpmm.FirmResult, error) {
	req := c.client.R().
		SetContext(ctx).
		SetBody(params)

	var result kyberpmm.FirmResult
	resp, err := req.SetResult(&result).Post(firmEndpoint)
	if err != nil {
		return kyberpmm.FirmResult{}, err
	}

	if !resp.IsSuccess() {
		return kyberpmm.FirmResult{}, errors.Wrapf(ErrFirmQuoteFailed, "response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	if result.Error != "" {
		parsedErr := parseFirmQuoteError(result.Error)
		logger.Errorf("firm quote failed with error: %v", result.Error)

		return kyberpmm.FirmResult{}, parsedErr
	}

	return result, nil
}

func parseFirmQuoteError(errorMessage string) error {
	switch errorMessage {
	case ErrFirmQuoteInternalErrorText:
		return ErrFirmQuoteInternalError
	case ErrFirmQuoteBlacklistText:
		return ErrFirmQuoteBlacklist
	case ErrFirmQuoteInsufficientLiquidityText:
		return ErrFirmQuoteInsufficientLiquidity
	case ErrFirmQuoteMarketConditionText:
		return ErrFirmQuoteMarketCondition
	default:
		return ErrFirmQuoteInternalError
	}
}

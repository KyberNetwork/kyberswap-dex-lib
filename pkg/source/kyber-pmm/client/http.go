package client

import (
	"context"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
)

const (
	listTokensEndpoint = "/kyberswap/v1/tokens"
	listPairsEndpoint  = "/kyberswap/v1/pairs"
	listPricesEndpoint = "/kyberswap/v1/prices"
	firmEndpoint       = "/kyberswap/v1/firm"
)

type client struct {
	restyClient *resty.Client
}

func NewClient(config *kyberpmm.HTTPConfig) *client {
	restyClient := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &client{
		restyClient: restyClient,
	}
}

func (c *client) ListTokens(ctx context.Context) (map[string]kyberpmm.TokenItem, error) {
	req := c.restyClient.R().
		SetContext(ctx)

	var result kyberpmm.ListTokensResult
	resp, err := req.SetResult(&result).Get(listTokensEndpoint)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.WithMessagef(ErrListTokensFailed, "[kyberPMM] response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result.Tokens, nil
}

func (c *client) ListPairs(ctx context.Context) (map[string]kyberpmm.PairItem, error) {
	req := c.restyClient.R().
		SetContext(ctx)

	var result kyberpmm.ListPairsResult
	resp, err := req.SetResult(&result).Get(listPairsEndpoint)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.WithMessagef(ErrListPairsFailed, "[kyberPMM] response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result.Pairs, nil
}

func (c *client) ListPriceLevels(ctx context.Context) (kyberpmm.ListPriceLevelsResult, error) {
	req := c.restyClient.R().
		SetContext(ctx)

	var result kyberpmm.ListPriceLevelsResult
	resp, err := req.SetResult(&result).Get(listPricesEndpoint)
	if err != nil {
		return result, err
	}

	if !resp.IsSuccess() {
		return result, errors.WithMessagef(ErrListPriceLevelsFailed, "[kyberPMM] response status: %v, response error: %v", resp.Status(), resp.Error())
	}

	return result, nil
}

func (c *client) Firm(ctx context.Context, params kyberpmm.FirmRequestParams) (kyberpmm.FirmResult, error) {
	req := c.restyClient.R().
		SetContext(ctx).
		SetBody(params)

	var result kyberpmm.FirmResult
	resp, err := req.SetResult(&result).Post(firmEndpoint)
	if err != nil {
		return kyberpmm.FirmResult{}, err
	}

	if !resp.IsSuccess() {
		return kyberpmm.FirmResult{}, errors.WithMessagef(ErrFirmQuoteFailed, "[kyberPMM] response status: %v, response error: %v", resp.Status(), resp.Error())
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

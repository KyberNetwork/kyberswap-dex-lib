package client

import "errors"

const (
	ErrFirmQuoteInternalErrorText         = "internal_error"
	ErrFirmQuoteBlacklistText             = "blacklist"
	ErrFirmQuoteInsufficientLiquidityText = "insufficient_liquidity"
	ErrFirmQuoteMarketConditionText       = "market_condition"

	// multi firm with alpha fee
	ErrAmountOutLessThanMinText = "out_amount_less_than_min"
	ErrMinGreaterExpectText     = "min_greater_expected"
)

var (
	ErrListTokensFailed               = errors.New("listTokens failed")
	ErrListPairsFailed                = errors.New("listPairs failed")
	ErrListPriceLevelsFailed          = errors.New("listPriceLevels failed")
	ErrFirmQuoteFailed                = errors.New("firm quote failed")
	ErrFirmQuoteInternalError         = errors.New(ErrFirmQuoteInternalErrorText)
	ErrFirmQuoteBlacklist             = errors.New(ErrFirmQuoteBlacklistText)
	ErrFirmQuoteInsufficientLiquidity = errors.New(ErrFirmQuoteInsufficientLiquidityText)
	ErrFirmQuoteMarketCondition       = errors.New(ErrFirmQuoteMarketConditionText)
	ErrAmountOutLessThanMin           = errors.New(ErrAmountOutLessThanMinText)
	ErrMinGreaterExpect               = errors.New(ErrMinGreaterExpectText)
)

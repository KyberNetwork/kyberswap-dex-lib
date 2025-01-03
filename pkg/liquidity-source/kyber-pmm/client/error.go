package client

import "errors"

const (
	ErrFirmQuoteInternalErrorText         = "internal_error"
	ErrFirmQuoteBlacklistText             = "blacklist"
	ErrFirmQuoteInsufficientLiquidityText = "insufficient_liquidity"
	ErrFirmQuoteMarketConditionText       = "market_condition"
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
)

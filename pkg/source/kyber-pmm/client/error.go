package client

import "errors"

var (
	ErrListTokensFailed      = errors.New("listTokens failed")
	ErrListPairsFailed       = errors.New("listPairs failed")
	ErrListPriceLevelsFailed = errors.New("listPriceLevels failed")
	ErrFirmQuoteFailed       = errors.New("firm quote failed")
)

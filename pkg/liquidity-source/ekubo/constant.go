package ekubo

import (
	"errors"
)

const DexType = "ekubo"

const (
	maxBatchSize                         = 100
	minTickSpacingsPerPool        uint32 = 2
	dataFetcherMethodGetQuoteData        = "getQuoteData"

	getPoolKeysEndpoint = "/v1/poolKeys"
)

var (
	ErrGetPoolKeysFailed = errors.New("get pool keys failed")
	ErrZeroSwapAmount    = errors.New("zero swap amount")
)

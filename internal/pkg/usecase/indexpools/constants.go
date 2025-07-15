package indexpools

import "errors"

var (
	MIN_DATA_POINT_NUMBER_DEFAULT     = 6
	MAX_DATA_POINT_NUMBER_DEFAULT     = 12
	MAX_EXPONENT_GENERATE_EXTRA_POINT = 3
	PRICE_IMPACT_THRESHOLD            = float64(0.5)
	INVALID_PRICE_IMPACT_THRESHOLD    = float64(1)
	DEFAULT_RFQ_SCORE                 = float64(1)
	ErrAmountOutNotValid              = errors.New("amount out is not valid")
)

const PRICE_CHUNK_SIZE = 100
const MAX_AMOUNT_OUT_USD = 10_000_000_000_000
const WHITELIST_FILENAME = "whitelist-whitelist.txt"
const WHITELIST_SCORE_FILENAME = "whitelist-whitelist.txt-Score"

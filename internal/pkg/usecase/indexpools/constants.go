package indexpools

import "errors"

const PRICE_CHUNK_SIZE = 100
const MAX_AMOUNT_OUT_USD = 10_000_000_000_000
const WHITELIST_FILENAME = "whitelist-whitelist.txt"
const WHITELIST_SCORE_FILENAME = "whitelist-whitelist.txt-Score"
const ZERO_SCORES_FILENAME = "zeroScores.txt-Score"
const MIN_DATA_POINT_NUMBER_DEFAULT = 6
const MAX_DATA_POINT_NUMBER_DEFAULT = 12
const MAX_EXPONENT_GENERATE_EXTRA_POINT = 3
const PRICE_IMPACT_THRESHOLD = 0.5
const INVALID_PRICE_IMPACT_THRESHOLD = 1.0
const DEFAULT_RFQ_SCORE = 1.0
const DEFAULT_AEVM_POOL_SCORE = 4000.0
const MIN_TVL_USD_AEVM_POOL_THRESHOLD = 5.0

var ErrAmountOutNotValid = errors.New("amount out is not valid")

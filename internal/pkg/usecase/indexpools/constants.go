package indexpools

import "errors"

var (
	MIN_DATA_POINT_NUMBER_DEFAULT     = 6
	MAX_DATA_POINT_NUMBER_DEFAULT     = 12
	MAX_EXPONENT_GENERATE_EXTRA_POINT = 3
	PRICE_IMPACT_THRESHOLD            = float64(0.5)
	ErrAmountOutNotValid              = errors.New("amount out is not valid")
)

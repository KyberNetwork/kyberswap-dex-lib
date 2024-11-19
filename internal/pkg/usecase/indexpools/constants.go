package indexpools

import "errors"

var (
	MIN_DATA_POINT_NUMBER_DEFAULT = 6
	MAX_DATA_POINT_NUMBER_DEFAULT = 12
	ErrAmountOutNotValid          = errors.New("amount out is not valid")
)

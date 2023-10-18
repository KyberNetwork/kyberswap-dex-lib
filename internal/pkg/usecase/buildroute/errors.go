package buildroute

import "errors"

var (
	ErrTokenNotFound                    = errors.New("token not found")
	ErrQuotedAmountSmallerThanEstimated = errors.New("quoted amount is smaller than estimated")
	ErrEstimateGasFailed                = errors.New("estimate gas failed")
)

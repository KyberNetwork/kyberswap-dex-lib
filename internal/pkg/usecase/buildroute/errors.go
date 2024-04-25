package buildroute

import (
	"errors"
	"strings"
)

var (
	ErrTokenNotFound                    = errors.New("token not found")
	ErrQuotedAmountSmallerThanEstimated = errors.New("quoted amount is smaller than estimated")
	ErrEstimateGasFailed                = errors.New("estimate gas failed")
	ErrSenderEmptyWhenEnableEstimateGas = errors.New("sender address is empty when enable estimate gas")
	ErrReturnAmountIsNotEnough          = errors.New("execution reverted: Return amount is not enough")
)

func IsErrReturnAmountIsNotEnough(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrReturnAmountIsNotEnough.Error())
}

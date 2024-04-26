package buildroute

import (
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/pkg/errors"
)

var (
	ErrTokenNotFound                    = errors.New("token not found")
	ErrQuotedAmountSmallerThanEstimated = errors.New("quoted amount is smaller than estimated")
	ErrSenderEmptyWhenEnableEstimateGas = errors.New("sender address is empty when enable estimate gas")
	ErrReturnAmountIsNotEnough          = errors.New("execution reverted: Return amount is not enough")

	ErrEstimateGasFailedCode = 4227
)

func IsErrReturnAmountIsNotEnough(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrReturnAmountIsNotEnough.Error())
}

func ErrEstimateGasFailed(err error) utils.WrappedError {
	return utils.NewWrappedError(errors.WithMessage(err, "estimate gas failed"), ErrEstimateGasFailedCode)
}

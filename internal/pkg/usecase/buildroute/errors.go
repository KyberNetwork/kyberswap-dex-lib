package buildroute

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrTokenNotFound                    = errors.New("token not found")
	ErrQuotedAmountSmallerThanEstimated = errors.New("quoted amount is smaller than estimated")
	ErrSenderEmptyWhenEnableEstimateGas = errors.New("sender address is empty when enable estimate gas")
	ErrReturnAmountIsNotEnough          = errors.New("return amount is not enough")
	ErrRFQTimeout                       = errors.New("rfq timed out")
	ErrCannotKeepDustTokenOut           = errors.New("cannot keep dust tokenOut")
	ErrRouteNotFound                    = errors.New("route not found")
	ErrInvalidRouteChecksum             = errors.New("invalid route checksum")

	ErrEstimateGasFailedCode = 4227
)

func ErrEstimateGasFailed(err error) utils.WrappedError {
	return utils.NewWrappedError(errors.WithMessage(err, "estimate gas failed"), ErrEstimateGasFailedCode)
}

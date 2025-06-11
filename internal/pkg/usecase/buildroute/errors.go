package buildroute

import (
	"strconv"
	"strings"

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
	ErrInvalidRouteChecksum             = errors.New("invalid route checksum")

	ErrEstimateGasFailedCode = 4227
)

func ErrEstimateGasFailed(err error) utils.WrappedError {
	return utils.NewWrappedError(errors.WithMessage(err, "estimate gas failed"), ErrEstimateGasFailedCode)
}

func IsSwapSinglePoolFailed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "swapSinglePool failed")
}

// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/src/contracts/AggregationExecutor.sol#L310-L318
func ExtractPoolIndexFromError(err error) (sequenceIndex, hopIndex int, ok bool) {
	if err == nil {
		return 0, 0, false
	}

	msg := err.Error()

	parts := strings.Split(msg, ":")
	if len(parts) < 3 {
		return 0, 0, false
	}

	var sequence, hop int
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "sequence ") {
			seqStr := strings.TrimSpace(strings.Split(part, "sequence ")[1])
			sequence, err = strconv.Atoi(strings.Fields(seqStr)[0])
			if err != nil {
				return 0, 0, false
			}
		}

		if strings.Contains(part, "hop ") {
			hopStr := strings.TrimSpace(strings.Split(part, "hop ")[1])
			hop, err = strconv.Atoi(hopStr)
			if err != nil {
				return 0, 0, false
			}
		}
	}

	return sequence, hop, true
}

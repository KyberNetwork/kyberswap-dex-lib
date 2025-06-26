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

const swapSinglePoolErrorPattern = "swapSinglePool failed at sequence:"

func ErrEstimateGasFailed(err error) utils.WrappedError {
	return utils.NewWrappedError(errors.WithMessage(err, "estimate gas failed"), ErrEstimateGasFailedCode)
}

func IsSwapSinglePoolFailed(err error) bool {
	return err != nil && strings.Contains(err.Error(), swapSinglePoolErrorPattern)
}

func ExtractPoolIndexFromError(err error) (sequenceIndex, hopIndex int, ok bool) {
	if err == nil {
		return 0, 0, false
	}

	msg := err.Error()

	pos := strings.Index(msg, swapSinglePoolErrorPattern)
	if pos == -1 {
		return 0, 0, false
	}

	msg = msg[pos+len(swapSinglePoolErrorPattern):]

	hopIdx := strings.Index(msg, "hop:")
	if hopIdx == -1 {
		return 0, 0, false
	}

	seqStr := strings.TrimSpace(msg[:hopIdx])

	hopStr := strings.TrimSpace(msg[hopIdx+len("hop:"):])

	if colonIdx := strings.Index(hopStr, ":"); colonIdx != -1 {
		hopStr = strings.TrimSpace(hopStr[:colonIdx])
	}

	seq, err1 := strconv.Atoi(seqStr)
	hop, err2 := strconv.Atoi(hopStr)

	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return seq, hop, true
}

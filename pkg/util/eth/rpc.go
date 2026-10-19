package eth

import (
	"context"
	"math/big"
	"strings"
	"time"
)

func CallWithBlockRetry[T any](ctx context.Context, call func() (T, error), blockNumber ...*big.Int) (T, error) {
	if len(blockNumber) > 0 && BlockNumberOrLatest(blockNumber[0]) == nil {
		return call()
	}
	var zero T
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		result, err := call()
		if err == nil || !isRetryableBlockError(err) {
			return result, err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 300 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		}
		if attempt == 2 {
			return result, err
		}
	}
}

func isRetryableBlockError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unknown block") || strings.Contains(message, "header not found")
}

func BlockNumberOrLatest(blockNumber *big.Int) *big.Int {
	if blockNumber == nil || blockNumber.Sign() <= 0 {
		return nil
	}
	return blockNumber
}

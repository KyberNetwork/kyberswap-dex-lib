package eth

import (
	"context"
	"strings"
	"time"
)

func CallWithBlockRetry[T any](ctx context.Context, call func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; attempt < 3; attempt++ {
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
	return zero, nil
}

func isRetryableBlockError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unknown block") || strings.Contains(message, "header not found")
}

package backoff

import (
	"math"
	"time"

	ebo "github.com/cenkalti/backoff/v4"
)

const (
	DefaultMaxRetry = 10
	DefaultStep     = 1.5
	DefaultInterval = 60 * time.Second
)

type Options struct {
	MaxRetry int
	Step     float64
	Interval time.Duration
}

// Retry still is RetryE but no return error
func Retry(fn func() error) {
	_ = RetryE(fn)
}

// RetryE return an error if Retry exceeds MaxElapsedTime
// with default options, this function will retry up to 15 minutes
func RetryE(fn func() error) error {
	opts := Options{
		MaxRetry: DefaultMaxRetry,
		Step:     DefaultStep,
		Interval: DefaultInterval,
	}

	return RetryWithOptions(fn, opts)
}

func RetryWithOptions(fn func() error, opts Options) error {
	setting := &ebo.ExponentialBackOff{
		InitialInterval:     ebo.DefaultInitialInterval,
		RandomizationFactor: 0, // no random
		Multiplier:          opts.Step,
		MaxInterval:         opts.Interval,
		MaxElapsedTime:      time.Duration(int(math.Round(float64(opts.MaxRetry)*opts.Step))) * opts.Interval,
		Clock:               ebo.SystemClock,
	}
	setting.Reset()
	return ebo.Retry(fn, setting)
}

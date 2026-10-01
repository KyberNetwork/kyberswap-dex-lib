package pool

import "errors"

var (
	ErrTokenNotAvailable  = errors.New("token is not available")
	ErrNotEnoughInventory = errors.New("not enough token balance in inventory")

	ErrUnsupported = errors.New("unsupported") // use this error to try other pool factories
	// ErrSwapAmountBelowMin: swap amount (in or out) is under the pool's minimum size; a larger swap may succeed.
	// Wrap it via errors.WithMessage so callers can detect it generically.
	ErrSwapAmountBelowMin = errors.New("swap amount below pool minimum")
)

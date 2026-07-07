package pool

import "errors"

var (
	ErrTokenNotAvailable  = errors.New("token is not available")
	ErrNotEnoughInventory = errors.New("not enough token balance in inventory")

	ErrUnsupported = errors.New("unsupported") // use this error to try other pool factories
)

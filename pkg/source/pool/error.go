package pool

import "errors"

var (
	ErrTokenNotAvailable  = errors.New("token is not available")
	ErrNotEnoughInventory = errors.New("not enough token balance in inventory")
)

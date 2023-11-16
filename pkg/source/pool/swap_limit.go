package pool

import "math/big"

// SwapLimit is the interface implement a separate limit from pool's State.
// i.e: SwapLimit must be something affect multiple pools once it is change.
type SwapLimit interface {
	// GetLimit return a limit for a certain key. Normally each dex will have different limit values
	// For example: PMM's key is token's string and its value is the inventory's balance of that token
	GetLimit(key string) *big.Int
	// UpdateLimit update both limits for a trade (assuming each trade is from 1 token to another token)
	// It returns the new limits for other purposes
	UpdateLimit(decreaseKey, increasedKey string, decreasedDelta, increasedDelta *big.Int) (increasedLimitAfter *big.Int, decreasedLimitAfter *big.Int, err error)
}

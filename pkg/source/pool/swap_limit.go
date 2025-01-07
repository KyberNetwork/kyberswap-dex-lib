package pool

import "math/big"

// SwapLimit is the interface implement a separate limit from pool's State.
// i.e: SwapLimit must be something affect multiple pools once it is change.
type SwapLimit interface {
	// Clone returns a clone of SwapLimit. Only guarantees that UpdateLimit of the original does not affect the clone.
	Clone() SwapLimit
	// GetExchange returns the exchange name
	GetExchange() string
	// GetLimit returns a limit for a certain key. Normally each dex will have different limit values
	// For example: PMM's key is token's string and its value is the inventory's balance of that token
	GetLimit(key string) *big.Int
	// GetSwapped returns the amount has been swapped through pools
	GetSwapped() map[string]*big.Int
	// GetAllowSenders returns a list of addresses that are allowed to swap through Limit Order.
	GetAllowSenders() string
	// UpdateLimit updates both limits for a trade (assuming each trade is from 1 token to another token)
	// It returns the new limits for other purposes
	UpdateLimit(decreaseKey, increasedKey string, decreasedDelta, increasedDelta *big.Int) (increasedLimitAfter *big.Int, decreasedLimitAfter *big.Int, err error)
}

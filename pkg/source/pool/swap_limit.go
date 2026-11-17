package pool

import (
	"math/big"

	"github.com/holiman/uint256"
)

// SwapLimit is the interface implement a separate limit from pool's State.
// i.e: SwapLimit must be something affect multiple pools once it is change.
type SwapLimit interface {
	// Clone returns a clone of SwapLimit. Only guarantees that UpdateLimit of the original does not affect the clone.
	Clone() SwapLimit
	// GetExchange returns the exchange name
	GetExchange() string
	// GetLimit returns a limit for a certain key. Normally each dex will have different limit values
	// For example: PMM's key is token's string and its value is the inventory's balance of that token
	GetLimit(key string) *uint256.Int
	// GetSwapped returns the amount has been swapped through pools
	GetSwapped() map[string]*uint256.Int
	// GetAllowedSenders returns a list of addresses that are allowed to swap through Limit Order.
	GetAllowedSenders() string
	// UpdateLimit updates both limits for a trade (assuming each trade is from 1 token to another token)
	// It returns the new limits for other purposes
	UpdateLimit(decreaseKey, increasedKey string, decreasedDelta, increasedDelta *uint256.Int) (increasedLimitAfter *uint256.Int, decreasedLimitAfter *uint256.Int, err error)
}

// SwapLimitMergePolicy determines how snapshots of the same shared limit are
// combined. The zero value preserves the existing maximum-value behavior.
type SwapLimitMergePolicy uint8

const (
	SwapLimitMergeMax SwapLimitMergePolicy = iota
	SwapLimitMergeMin
)

// ISwapLimitMergePolicy is an optional simulator capability. Simulators sharing
// a limit key must agree on its policy. Keys without a policy default to max.
type ISwapLimitMergePolicy interface {
	GetLimitMergePolicy(key string) SwapLimitMergePolicy
}

// SwapLimitBig is the interface implement a separate limit from pool's State.
// i.e: SwapLimitBig must be something affect multiple pools once it is change.
type SwapLimitBig interface {
	// Clone returns a clone of SwapLimitBig. Only guarantees that UpdateLimit of the original does not affect the clone.
	Clone() SwapLimitBig
	// GetExchange returns the exchange name
	GetExchange() string
	// GetLimit returns a limit for a certain key. Normally each dex will have different limit values
	// For example: PMM's key is token's string and its value is the inventory's balance of that token
	GetLimit(key string) *big.Int
	// GetSwapped returns the amount has been swapped through pools
	GetSwapped() map[string]*big.Int
	// GetAllowedSenders returns a list of addresses that are allowed to swap through Limit Order.
	GetAllowedSenders() string
	// UpdateLimit updates both limits for a trade (assuming each trade is from 1 token to another token)
	// It returns the new limits for other purposes
	UpdateLimit(decreaseKey, increasedKey string, decreasedDelta, increasedDelta *big.Int) (increasedLimitAfter *big.Int, decreasedLimitAfter *big.Int, err error)
}

// MergeSwapLimits combines a simulator's snapshot into shared limits without
// mutating either snapshot's integers. Values must be non-nil and remain read-only.
func MergeSwapLimits(limits map[string]*uint256.Int, simulator IPoolSim) {
	policy, hasPolicy := simulator.(ISwapLimitMergePolicy)
	for key, value := range simulator.CalcLimit() {
		old, exists := limits[key]
		useMin := hasPolicy && policy.GetLimitMergePolicy(key) == SwapLimitMergeMin
		if !exists || (useMin && old.Cmp(value) > 0) || (!useMin && old.Cmp(value) < 0) {
			limits[key] = value
		}
	}
}

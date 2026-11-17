package swaplimit

import (
	"maps"
	"sync"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Inventory implement inventory-based Swap Limit for kyber-pmm, limit order, etc...
// key is any string, and the limit is its balance.
// The balances are stored WITHOUT decimals.
// DO NOT directly modify it but use UpdateLimit instead.
type Inventory struct {
	exchange string
	lock     *sync.RWMutex
	balance  map[string]*uint256.Int
}

// NewInventory creates a new Inventory.
func NewInventory(exchange string, balance map[string]*uint256.Int) *Inventory {
	return &Inventory{
		exchange: exchange,
		lock:     &sync.RWMutex{},
		balance:  balance,
	}
}

// Clone clones Inventory. Only guarantees that UpdateLimit of the original does not affect the clone.
// The balances are copied on UpdateBalance, and so is shadow-cloned here.
func (i *Inventory) Clone() pool.SwapLimit {
	return &Inventory{
		exchange: i.exchange,
		lock:     &sync.RWMutex{},
		balance:  maps.Clone(i.balance),
	}
}

// GetExchange returns the exchange name.
func (i *Inventory) GetExchange() string {
	return i.exchange
}

// GetLimit returns the balance for the token in Inventory. Do not modify the result.
func (i *Inventory) GetLimit(tokenAddress string) *uint256.Int {
	i.lock.RLock()
	balance, ok := i.balance[tokenAddress]
	i.lock.RUnlock()
	if !ok {
		return nil
	}
	return balance
}

func (i *Inventory) GetSwapped() map[string]*uint256.Int {
	return nil
}

func (i *Inventory) GetAllowedSenders() string {
	return ""
}

// CheckLimit returns an error when the balance for the token is missing or below amount.
func (i *Inventory) CheckLimit(tokenAddress string, amount *uint256.Int) error {
	i.lock.RLock()
	balance, ok := i.balance[tokenAddress]
	i.lock.RUnlock()
	if !ok {
		return pool.ErrTokenNotAvailable
	}
	if balance.Lt(amount) {
		return pool.ErrNotEnoughInventory
	}
	return nil
}

// UpdateLimit updates the balances to reflect the inventory change of a swap. Creates a new *uint256.Int so as not
// to affect limits in a cloned Inventory. Note that the delta amounts are without decimals.
func (i *Inventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (i *Inventory) updateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if decreaseDelta == nil || increaseDelta == nil {
		return new(uint256.Int), new(uint256.Int), pool.ErrNotEnoughInventory
	}
	decreasedTokenBalance, ok := i.balance[decreaseTokenAddress]
	if !ok {
		return new(uint256.Int), new(uint256.Int), pool.ErrTokenNotAvailable
	} else if decreasedTokenBalance.Lt(decreaseDelta) {
		return new(uint256.Int), new(uint256.Int), pool.ErrNotEnoughInventory
	}

	decreasedTokenBalance = new(uint256.Int).Sub(decreasedTokenBalance, decreaseDelta)
	i.balance[decreaseTokenAddress] = decreasedTokenBalance

	increasedTokenBalance, ok := i.balance[increaseTokenAddress]
	if !ok {
		increasedTokenBalance = new(uint256.Int).Set(increaseDelta)
	} else {
		increasedTokenBalance = new(uint256.Int).Add(increasedTokenBalance, increaseDelta)
	}
	i.balance[increaseTokenAddress] = increasedTokenBalance

	return decreasedTokenBalance, increasedTokenBalance, nil
}

package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Inventory implement inventory-based Swap Limit for kyber-pmm, limit order, etc...
// key is any string, and the limit is its balance.
// The balances are stored WITHOUT decimals.
// DO NOT directly modify it but use UpdateLimit instead.
type Inventory struct {
	exchange string
	lock     *sync.RWMutex
	balance  map[string]*big.Int
}

// NewInventory creates a new Inventory.
func NewInventory(exchange string, balance map[string]*big.Int) *Inventory {
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
func (i *Inventory) GetLimit(tokenAddress string) *big.Int {
	i.lock.RLock()
	balance, ok := i.balance[tokenAddress]
	i.lock.RUnlock()
	if !ok {
		return bignumber.ZeroBI
	}
	return balance
}

// CheckLimit returns the balance for the token in Inventory. Do not modify the result.
func (i *Inventory) CheckLimit(tokenAddress string, amount *big.Int) error {
	i.lock.RLock()
	balance, ok := i.balance[tokenAddress]
	i.lock.RUnlock()
	if !ok {
		return pool.ErrTokenNotAvailable
	}
	if balance.Cmp(amount) < 0 {
		return pool.ErrNotEnoughInventory
	}
	return nil
}

// UpdateLimit updates the balances to reflect the inventory change of a swap. Creates a new *big.Int so as not to
// affect limits in a cloned Inventory. Note that the delta amounts are without decimals.
func (i *Inventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	decreasedTokenBalance, ok := i.balance[decreaseTokenAddress]
	if !ok {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrTokenNotAvailable
	}
	increasedTokenBalance, ok := i.balance[increaseTokenAddress]
	if !ok {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrTokenNotAvailable
	}
	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrNotEnoughInventory
	}

	decreasedTokenBalance = new(big.Int).Sub(decreasedTokenBalance, decreaseDelta)
	i.balance[decreaseTokenAddress] = decreasedTokenBalance
	increasedTokenBalance = new(big.Int).Add(increasedTokenBalance, increaseDelta)
	i.balance[increaseTokenAddress] = increasedTokenBalance

	return decreasedTokenBalance, increasedTokenBalance, nil
}

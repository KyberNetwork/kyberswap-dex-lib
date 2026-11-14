package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
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
		return nil
	}
	return balance
}

func (i *Inventory) GetSwapped() map[string]*big.Int {
	return nil
}

func (i *Inventory) GetAllowedSenders() string {
	return ""
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

	return i.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (i *Inventory) updateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	decreasedTokenBalance, ok := i.balance[decreaseTokenAddress]
	if !ok {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrTokenNotAvailable
	} else if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrNotEnoughInventory
	}

	decreasedTokenBalance = new(big.Int).Sub(decreasedTokenBalance, decreaseDelta)
	i.balance[decreaseTokenAddress] = decreasedTokenBalance

	increasedTokenBalance, ok := i.balance[increaseTokenAddress]
	if !ok {
		increasedTokenBalance = new(big.Int).Set(increaseDelta)
	} else {
		increasedTokenBalance = new(big.Int).Add(increasedTokenBalance, increaseDelta)
	}
	i.balance[increaseTokenAddress] = increasedTokenBalance

	return decreasedTokenBalance, increasedTokenBalance, nil
}

// GetLimitU is GetLimit with a uint256 result. It returns nil when the key is absent, or when the balance is
// negative or does not fit in 256 bits, so the caller never reads a wrapped value.
func (i *Inventory) GetLimitU(tokenAddress string) *uint256.Int {
	balance := i.GetLimit(tokenAddress)
	if balance == nil {
		return nil
	}
	u, ok := toU256(balance)
	if !ok {
		return nil
	}
	return u
}

// CheckLimitU is CheckLimit with uint256 amounts. The amount must fit the same way as GetLimitU.
func (i *Inventory) CheckLimitU(tokenAddress string, amount *uint256.Int) error {
	return i.CheckLimit(tokenAddress, amount.ToBig())
}

// UpdateLimitU is UpdateLimit with uint256 deltas and results. It runs the same big.Int logic, so the
// balances and errors match UpdateLimit exactly.
func (i *Inventory) UpdateLimitU(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	dec, inc, err := i.UpdateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta.ToBig(), increaseDelta.ToBig())
	if err != nil {
		return nil, nil, err
	}
	decU, ok1 := toU256(dec)
	incU, ok2 := toU256(inc)
	if !ok1 || !ok2 {
		return nil, nil, pool.ErrNotEnoughInventory
	}
	return decU, incU, nil
}

// toU256 converts a non-negative big.Int that fits 256 bits. It reports false for anything else.
func toU256(b *big.Int) (*uint256.Int, bool) {
	if b.Sign() < 0 || b.BitLen() > 256 {
		return nil, false
	}
	return uint256.MustFromBig(b), true
}

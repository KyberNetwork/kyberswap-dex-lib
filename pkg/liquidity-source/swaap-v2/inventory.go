package swaapv2

import (
	"errors"
	"math/big"
	"sync"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

var (
	ErrInventoryTokenNotFound         = errors.New("inventory: token not found")
	ErrInventoryInsufficientLiquidity = errors.New("inventory: insufficient liquidity")
)

type Inventory struct {
	lock    *sync.RWMutex
	Balance map[string]*big.Int
}

func NewInventory(balance map[string]*big.Int) *Inventory {
	return &Inventory{
		lock:    &sync.RWMutex{},
		Balance: balance,
	}
}

// GetLimit returns a copy of balance for the token in Inventory
func (i *Inventory) GetLimit(tokenAddress string) *big.Int {
	i.lock.RLock()
	defer i.lock.RUnlock()

	balance, ok := i.Balance[tokenAddress]
	if !ok {
		return integer.Zero()
	}

	return new(big.Int).Set(balance)
}

// CheckLimit returns a copy of balance for the token in Inventory
func (i *Inventory) CheckLimit(tokenAddress string, amount *big.Int) error {
	i.lock.RLock()
	defer i.lock.RUnlock()

	balance, ok := i.Balance[tokenAddress]
	if !ok {

		return ErrInventoryTokenNotFound
	}

	if balance.Cmp(amount) < 0 {
		return ErrInventoryInsufficientLiquidity
	}

	return nil
}

// UpdateLimit will reduce the limit to reflect the change in inventory
// note this delta is amount without Decimal
func (i *Inventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string, decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	decreasedTokenBalance, ok := i.Balance[decreaseTokenAddress]
	if !ok {
		return nil, nil, ErrInventoryTokenNotFound
	}

	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return nil, nil, ErrInventoryInsufficientLiquidity
	}

	i.Balance[decreaseTokenAddress] = new(big.Int).Sub(decreasedTokenBalance, decreaseDelta)

	increasedTokenBalance, ok := i.Balance[increaseTokenAddress]
	if !ok {
		return nil, nil, ErrInventoryTokenNotFound
	}

	i.Balance[increaseTokenAddress] = new(big.Int).Add(increasedTokenBalance, increaseDelta)

	return new(big.Int).Set(i.Balance[decreaseTokenAddress]), new(big.Int).Set(i.Balance[increaseTokenAddress]), nil
}

package bebop

import (
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Inventory struct {
	mu      sync.RWMutex
	balance map[string]*big.Int
}

func NewInventory(balance map[string]*big.Int) *Inventory {
	return &Inventory{
		mu:      sync.RWMutex{},
		balance: balance,
	}
}

// GetLimit returns a copy of balance for the token in Inventory
func (i *Inventory) GetLimit(tokenAddress string) *big.Int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	balance, avail := i.balance[tokenAddress]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// CheckLimit returns a copy of balance for the token in Inventory
func (i *Inventory) CheckLimit(tokenAddress string, amount *big.Int) error {
	i.mu.RLock()
	defer i.mu.RUnlock()
	balance, avail := i.balance[tokenAddress]
	if !avail {
		return ErrTokenNotFound
	}
	if balance.Cmp(amount) < 0 {
		return ErrInsufficientLiquidity
	}
	return nil
}

// UpdateLimit will reduce the limit to reflect the change in inventory
// note this delta is amount without Decimal
func (i *Inventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string, decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	decreasedTokenBalance, avail := i.balance[decreaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), pool.ErrNotEnoughInventory
	}
	i.balance[decreaseTokenAddress] = decreasedTokenBalance.Sub(decreasedTokenBalance, decreaseDelta)

	increasedTokenBalance, avail := i.balance[increaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	i.balance[increaseTokenAddress] = increasedTokenBalance.Add(increasedTokenBalance, increaseDelta)
	return big.NewInt(0).Set(i.balance[decreaseTokenAddress]), big.NewInt(0).Set(i.balance[increaseTokenAddress]), nil
}

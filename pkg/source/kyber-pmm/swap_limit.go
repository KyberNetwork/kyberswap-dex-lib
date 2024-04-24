//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Inventory
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package kyberpmm

import (
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Inventory implement Swap Limit for kyber-pmm
// key is tokenAddress, and the limit is its balance
// The balance is stored WITHOUT decimals
// DONOT directly modify it, use UpdateLimit if needed
type Inventory struct {
	lock    *sync.RWMutex `msg:"-"`
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
	balance, avail := i.Balance[tokenAddress]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// CheckLimit returns a copy of balance for the token in Inventory
func (i *Inventory) CheckLimit(tokenAddress string, amount *big.Int) error {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Balance[tokenAddress]
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
	i.lock.Lock()
	defer i.lock.Unlock()
	decreasedTokenBalance, avail := i.Balance[decreaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), pool.ErrNotEnoughInventory
	}
	i.Balance[decreaseTokenAddress] = decreasedTokenBalance.Sub(decreasedTokenBalance, decreaseDelta)

	increasedTokenBalance, avail := i.Balance[increaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	i.Balance[increaseTokenAddress] = increasedTokenBalance.Add(increasedTokenBalance, increaseDelta)
	return big.NewInt(0).Set(i.Balance[decreaseTokenAddress]), big.NewInt(0).Set(i.Balance[increaseTokenAddress]), nil
}

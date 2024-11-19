package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type InventoryWithSwapped struct {
	Inventory
	swapped map[string]*big.Int
}

func NewInventoryWithSwapped(exchange string, balance map[string]*big.Int) *InventoryWithSwapped {
	return &InventoryWithSwapped{
		Inventory: Inventory{
			exchange: exchange,
			lock:     &sync.RWMutex{},
			balance:  balance,
		},
		swapped: make(map[string]*big.Int),
	}
}

func (k *InventoryWithSwapped) GetSwapped() map[string]*big.Int {
	k.lock.RLock()
	defer k.lock.RUnlock()

	return maps.Clone(k.swapped)
}

func (k *InventoryWithSwapped) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	swapped1, ok := k.swapped[decreaseTokenAddress]
	if !ok {
		swapped1 = new(big.Int)
	}
	swapped1 = swapped1.Sub(swapped1, decreaseDelta)
	k.swapped[decreaseTokenAddress] = swapped1

	swapped2, ok := k.swapped[increaseTokenAddress]
	if !ok {
		swapped2 = new(big.Int)
	}
	swapped2 = swapped1.Add(swapped2, increaseDelta)
	k.swapped[increaseTokenAddress] = swapped2

	decreasedTokenBalance, ok := k.balance[decreaseTokenAddress]
	if !ok {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrTokenNotAvailable
	} else if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return bignumber.ZeroBI, bignumber.ZeroBI, pool.ErrNotEnoughInventory
	}

	decreasedTokenBalance = new(big.Int).Sub(decreasedTokenBalance, decreaseDelta)
	k.balance[decreaseTokenAddress] = decreasedTokenBalance

	increasedTokenBalance, ok := k.balance[increaseTokenAddress]
	if !ok {
		increasedTokenBalance = new(big.Int).Set(increaseDelta)
	} else {
		increasedTokenBalance = new(big.Int).Add(increasedTokenBalance, increaseDelta)
	}
	k.balance[increaseTokenAddress] = increasedTokenBalance

	return decreasedTokenBalance, increasedTokenBalance, nil
}

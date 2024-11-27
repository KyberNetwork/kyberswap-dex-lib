package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type SwappedInventory struct {
	Inventory
	swapped map[string]*big.Int
}

func NewSwappedInventory(exchange string, balance map[string]*big.Int) *SwappedInventory {
	return &SwappedInventory{
		Inventory: Inventory{
			exchange: exchange,
			lock:     &sync.RWMutex{},
			balance:  balance,
		},
		swapped: make(map[string]*big.Int),
	}
}

func (k *SwappedInventory) GetSwapped() map[string]*big.Int {
	k.lock.RLock()
	defer k.lock.RUnlock()

	return maps.Clone(k.swapped)
}

func (k *SwappedInventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	swappedOut, ok := k.swapped[decreaseTokenAddress]
	if !ok {
		swappedOut = new(big.Int)
	}
	swappedOut = swappedOut.Sub(swappedOut, decreaseDelta)
	k.swapped[decreaseTokenAddress] = swappedOut

	swappedIn, ok := k.swapped[increaseTokenAddress]
	if !ok {
		swappedIn = new(big.Int)
	}
	swappedIn = swappedIn.Add(swappedIn, increaseDelta)
	k.swapped[increaseTokenAddress] = swappedIn

	return k.Inventory.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (k *SwappedInventory) Clone() pool.SwapLimit {
	return &SwappedInventory{
		Inventory: Inventory{
			exchange: k.exchange,
			lock:     &sync.RWMutex{},
			balance:  maps.Clone(k.balance),
		},
		swapped: maps.Clone(k.swapped),
	}
}

package swaplimit

import (
	"maps"
	"sync"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type SwappedInventory struct {
	Inventory
	swapped map[string]*uint256.Int
}

func NewSwappedInventory(exchange string, balance map[string]*uint256.Int) *SwappedInventory {
	return &SwappedInventory{
		exchange: exchange,
		lock:     &sync.RWMutex{},
		balance:  balance,
		swapped:  make(map[string]*uint256.Int),
	}
}

func (k *SwappedInventory) GetSwapped() map[string]*uint256.Int {
	k.lock.RLock()
	defer k.lock.RUnlock()

	return maps.Clone(k.swapped)
}

func (k *SwappedInventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	// Comment out these, since we only keep track of swapped in.
	// Subtract swapped out may make the future swap in value
	// reduce, cause the swap limit to be lower than expected.

	// swappedOut, ok := k.swapped[decreaseTokenAddress]
	// if !ok {
	// 	swappedOut = new(uint256.Int)
	// }
	// swappedOut = swappedOut.Sub(swappedOut, decreaseDelta)
	// k.swapped[decreaseTokenAddress] = swappedOut

	swappedIn, ok := k.swapped[increaseTokenAddress]
	if !ok {
		swappedIn = new(uint256.Int)
	}
	swappedIn = swappedIn.Add(swappedIn, increaseDelta)
	k.swapped[increaseTokenAddress] = swappedIn

	return k.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (k *SwappedInventory) Clone() pool.SwapLimit {
	return &SwappedInventory{
		exchange: k.exchange,
		lock:     &sync.RWMutex{},
		balance:  maps.Clone(k.balance),
		swapped:  maps.Clone(k.swapped),
	}
}

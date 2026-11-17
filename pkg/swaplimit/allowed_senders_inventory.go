package swaplimit

import (
	"maps"
	"sync"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type InventoryWithAllowedSenders struct {
	Inventory
	AllowedSenders string
}

func NewInventoryWithAllowedSenders(
	exchange string,
	balance map[string]*uint256.Int,
	allowedSenders string,
) *InventoryWithAllowedSenders {
	return &InventoryWithAllowedSenders{
		exchange:       exchange,
		lock:           &sync.RWMutex{},
		balance:        balance,
		AllowedSenders: allowedSenders,
	}
}

func (i *InventoryWithAllowedSenders) GetAllowedSenders() string {
	return i.AllowedSenders
}

func (i *InventoryWithAllowedSenders) UpdateLimit(
	decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	return i.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (i *InventoryWithAllowedSenders) Clone() pool.SwapLimit {
	return &InventoryWithAllowedSenders{
		exchange:       i.exchange,
		lock:           &sync.RWMutex{},
		balance:        maps.Clone(i.balance),
		AllowedSenders: i.AllowedSenders,
	}
}

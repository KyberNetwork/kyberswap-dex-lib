package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type InventoryWithAllowedSenders struct {
	Inventory
	AllowedSenders string
}

func NewInventoryWithAllowedSenders(
	exchange string,
	balance map[string]*big.Int,
	allowedSenders string,
) *InventoryWithAllowedSenders {
	return &InventoryWithAllowedSenders{
		Inventory: Inventory{
			exchange: exchange,
			lock:     &sync.RWMutex{},
			balance:  balance,
		},
		AllowedSenders: allowedSenders,
	}
}

func (i *InventoryWithAllowedSenders) GetAllowedSenders() string {
	return i.AllowedSenders
}

func (i *InventoryWithAllowedSenders) UpdateLimit(
	decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int,
) (*big.Int, *big.Int, error) {
	return i.Inventory.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (i *InventoryWithAllowedSenders) Clone() pool.SwapLimit {
	return &InventoryWithAllowedSenders{
		Inventory: Inventory{
			exchange: i.exchange,
			lock:     &sync.RWMutex{},
			balance:  maps.Clone(i.balance),
		},
		AllowedSenders: i.AllowedSenders,
	}
}

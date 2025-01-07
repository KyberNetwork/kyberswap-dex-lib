package swaplimit

import (
	"maps"
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type InventoryWithAllowSenders struct {
	Inventory
	AllowSenders string
}

func NewInventoryWithAllowSenders(
	exchange string,
	balance map[string]*big.Int,
	allowSenders string,
) *InventoryWithAllowSenders {
	return &InventoryWithAllowSenders{
		Inventory: Inventory{
			exchange: exchange,
			lock:     &sync.RWMutex{},
			balance:  balance,
		},
		AllowSenders: allowSenders,
	}
}

func (i *InventoryWithAllowSenders) GetAllowSenders() string {
	return i.AllowSenders
}

func (i *InventoryWithAllowSenders) UpdateLimit(
	decreaseTokenAddress, increaseTokenAddress string,
	decreaseDelta, increaseDelta *big.Int,
) (*big.Int, *big.Int, error) {
	return i.Inventory.updateLimit(decreaseTokenAddress, increaseTokenAddress, decreaseDelta, increaseDelta)
}

func (i *InventoryWithAllowSenders) Clone() pool.SwapLimit {
	return &InventoryWithAllowSenders{
		Inventory: Inventory{
			exchange: i.exchange,
			lock:     &sync.RWMutex{},
			balance:  maps.Clone(i.balance),
		},
		AllowSenders: i.AllowSenders,
	}
}

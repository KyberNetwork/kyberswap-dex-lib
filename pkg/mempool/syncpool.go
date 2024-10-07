package mempool

import (
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
)

var EntityPool = sync.Pool{
	// New optionally specifies a function to generate
	// a value when Get would otherwise return nil.
	New: func() interface{} { return new(entity.Pool) },
}

// Reserve memory for pool entities in heap memory, avoid mem allocation burden
// Any item stored in the Pool may be removed automatically at any time without notification.
// If the Pool holds the only reference when this happens, the item might be deallocated.
// So only put pool entities into sync.Pool after using these entities to avoid deallocated, in this case using defer is correct.
func ReserveMany(pools []*entity.Pool) {
	for index := range pools {
		EntityPool.Put(pools[index])
	}
}

var AddressListPool = sync.Pool{

	New: func() interface{} {
		return &types.AddressList{
			Arr:     make([]string, 100),
			TrueLen: 0,
		}
	},
}

func ReturnAddressList(l *types.AddressList) {
	if l == nil {
		return
	}
	l.TrueLen = 0
	AddressListPool.Put(l)
}

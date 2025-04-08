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

// Reserve reserves memory for pool entity in a sync.Pool, reducing memory allocation burden.
// Only put pool entities into sync.Pool after finishing using them.
func Reserve(pool *entity.Pool) {
	EntityPool.Put(pool)
}

// ReserveMany is the same as Reserve, but for multiple pool entities.
func ReserveMany(pools ...*entity.Pool) {
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

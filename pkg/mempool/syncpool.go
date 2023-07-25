package mempool

import (
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

var EntityPool = sync.Pool{
	// New optionally specifies a function to generate
	// a value when Get would otherwise return nil.
	New: func() interface{} { return new(entity.Pool) },
}

func ReserveMany(pools []*entity.Pool) {
	for index := range pools {
		EntityPool.Put(pools[index])
	}
}

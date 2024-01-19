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
	l.TrueLen = 0
	AddressListPool.Put(l)
}

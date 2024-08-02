package poolmanager

import (
	"context"
	"sync"
	"sync/atomic"

	cachePolicy "github.com/hashicorp/golang-lru/v2"
)

func NewPointerSwapPoolManagerInstance(
	states [3]*LockedState,
	poolFactory IPoolFactory,
	poolRepository IPoolRepository,
	poolRankRepository IPoolRankRepository,
	config Config,
	poolCache *cachePolicy.Cache[string, struct{}]) PointerSwapPoolManager {
	return PointerSwapPoolManager{
		states:             states,
		readFrom:           atomic.Int32{},
		config:             config,
		poolFactory:        poolFactory,
		poolRepository:     poolRepository,
		poolRankRepository: poolRankRepository,
		poolCache:          poolCache,
		faultyPoolsLock:    &sync.RWMutex{},
		blackListlock:      &sync.RWMutex{},
	}
}

func (m *PointerSwapPoolManager) FilterInvalidPoolAddresses(addresses []string) []string {
	return m.filterInvalidPoolAddresses(addresses)
}

func (m *PointerSwapPoolManager) UpdateFaultyPools(ctx context.Context) {
	m.updateFaultyPools(ctx)
}

func (m *PointerSwapPoolManager) UpdateBlackListPool(ctx context.Context) {
	m.updateBlackListPool(ctx)
}

func (m *PointerSwapPoolManager) ReadFrom() int32 {
	return m.readFrom.Load()
}

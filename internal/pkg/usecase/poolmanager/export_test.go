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
	poolCache *cachePolicy.Cache[string, struct{}],
	lock *sync.RWMutex) PointerSwapPoolManager {
	return PointerSwapPoolManager{
		states:             states,
		readFrom:           atomic.Int32{},
		config:             config,
		poolFactory:        poolFactory,
		poolRepository:     poolRepository,
		poolRankRepository: poolRankRepository,
		poolCache:          poolCache,
		lock:               &sync.RWMutex{},
	}
}

func (m *PointerSwapPoolManager) ExcludeFaultyPools(ctx context.Context, addresses []string) []string {
	return m.excludeFaultyPools(ctx, addresses)
}

package poolmanager

import (
	"context"
	"sync/atomic"

	mapset "github.com/deckarep/golang-set/v2"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
)

func NewPointerSwapPoolManagerInstance(
	states [3]*LockedState,
	poolFactory IPoolFactory,
	poolRepository IPoolRepository,
	poolRankRepository IPoolRankRepository,
	config Config,
	poolCache *cachePolicy.Cache[string, struct{}],
	faultyPools mapset.Set[string],
	blacklist mapset.Set[string]) PointerSwapPoolManager {
	return PointerSwapPoolManager{
		states:             states,
		readFrom:           atomic.Int32{},
		config:             config,
		poolFactory:        poolFactory,
		poolRepository:     poolRepository,
		poolRankRepository: poolRankRepository,
		poolCache:          poolCache,
		faultyPools:        faultyPools,
		blackListPools:     blacklist,
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

func (m *PointerSwapPoolManager) BlackListPool() mapset.Set[string] {
	return m.blackListPools
}

func (m *PointerSwapPoolManager) FaultyPools() mapset.Set[string] {
	return m.faultyPools
}

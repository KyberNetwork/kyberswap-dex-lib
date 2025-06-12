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

func (p *PointerSwapPoolManager) FilterInvalidPoolAddresses(addresses []string) []string {
	return p.filterInvalidPoolAddresses(addresses)
}

func (p *PointerSwapPoolManager) UpdateFaultyPools(ctx context.Context) {
	p.updateFaultyPools(ctx)
}

func (p *PointerSwapPoolManager) UpdateBlackListPool(ctx context.Context) {
	p.updateBlackListPool(ctx)
}

func (p *PointerSwapPoolManager) ReadFrom() int32 {
	return p.readFrom.Load()
}

func (p *PointerSwapPoolManager) BlackListPool() mapset.Set[string] {
	return p.blackListPools
}

func (p *PointerSwapPoolManager) FaultyPools() mapset.Set[string] {
	return p.faultyPools
}

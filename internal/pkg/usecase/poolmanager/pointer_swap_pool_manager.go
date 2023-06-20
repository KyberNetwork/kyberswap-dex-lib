package poolmanager

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"k8s.io/apimachinery/pkg/util/sets"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type PointerSwapPoolManager struct {
	poolFactory        IPoolFactory
	poolRepository     IPoolRepository
	poolRankRepository IPoolRankRepository

	states   [2]*LockedState
	readFrom atomic.Int32
	config   Config

	// poolCache control which pool to maintain when there are too many pools
	// currently poolCache use LRU policy
	poolCache *cachePolicy.Cache[string, struct{}]

	lock *sync.RWMutex
}

type LockedState struct {
	poolByAddress map[string]poolPkg.IPool
	lock          *sync.RWMutex
}

func (s *LockedState) update(poolByAddress map[string]poolPkg.IPool) {
	s.lock.Lock()
	s.poolByAddress = poolByAddress
	s.lock.Unlock()
}

// NewPointerSwapPoolManager This will take a while to start since it will generate a copy of all Pool
func NewPointerSwapPoolManager(
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	config Config,
) (*PointerSwapPoolManager, error) {
	states := [2]*LockedState{}
	for i := 0; i < 2; i++ {
		states[i] = &LockedState{
			poolByAddress: make(map[string]poolPkg.IPool),
			lock:          &sync.RWMutex{},
		}
	}
	//TODO try policies other than LRU
	poolCache, err := cachePolicy.New[string, struct{}](config.Capacity)
	if err != nil {
		return nil, err
	}

	// initialize pools to read from DB
	poolAddresses := poolRankRepository.FindGlobalBestPools(context.Background(), int64(config.Capacity))
	// add in reverse order so that pools with most volume at top of LRU list
	for i := len(poolAddresses) - 1; i >= 0; i-- {
		poolCache.Add(poolAddresses[i], struct{}{})
	}

	p := PointerSwapPoolManager{
		states:             states,
		readFrom:           atomic.Int32{},
		config:             config,
		poolFactory:        poolFactory,
		poolRepository:     poolRepository,
		poolRankRepository: poolRankRepository,
		poolCache:          poolCache,
		lock:               &sync.RWMutex{},
	}
	p.readFrom.Store(0)

	if err = p.preparePoolsData(context.Background(), poolAddresses); err != nil {
		return nil, err
	}
	go p.maintain()
	return &p, nil
}

func (p *PointerSwapPoolManager) ApplyConfig(config Config) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.config = config
	p.poolCache.Resize(config.Capacity)
}

// GetPoolByAddress return a reference to pools maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned here, clone IPool before UpdateBalance
func (p *PointerSwapPoolManager) GetPoolByAddress(ctx context.Context, poolAddresses, dex []string) (map[string]poolPkg.IPool, error) {
	filteredPoolAddress := p.filterBlacklistedAddresses(poolAddresses)

	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	readFrom := p.readFrom.Load()
	return p.getPools(ctx, filteredPoolAddress, dex, readFrom), nil
}

func (p *PointerSwapPoolManager) getPools(ctx context.Context, poolAddresses, dex []string, readFrom int32) map[string]poolPkg.IPool {
	var (
		resultPoolByAddress = make(map[string]poolPkg.IPool, len(poolAddresses))
		poolsToFetchFromDB  []string
		dexSet              = sets.NewString(dex...)
	)

	p.states[readFrom].lock.RLock()
	for _, key := range poolAddresses {
		if pool, ok := p.states[readFrom].poolByAddress[key]; ok {
			if dexSet.Has(pool.GetExchange()) {
				resultPoolByAddress[key] = pool
			}
		} else {
			poolsToFetchFromDB = append(poolsToFetchFromDB, key)
		}
	}
	p.states[readFrom].lock.RUnlock()

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, poolsToFetchFromDB)
	if err != nil {
		return resultPoolByAddress
	}

	filteredPoolEntities := make([]*entity.Pool, 0, len(poolEntities))
	for i := range poolEntities {
		if dexSet.Has(poolEntities[i].Exchange) {
			filteredPoolEntities = append(filteredPoolEntities, poolEntities[i])
		}
	}

	//Note, when base pool is not fetched from DB, but curve-meta is fetched from DB -> cannot init curve-meta from factory
	//Skipping this case because it is unlikely and the logic to fix this is complicated
	//To fix: unmarshall curve-meta to get basePool address, fetch base pools from DB, pass both to factory
	poolInterfaces := p.poolFactory.NewPools(ctx, filteredPoolEntities)
	for i := range poolInterfaces {
		resultPoolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	return resultPoolByAddress
}

func (p *PointerSwapPoolManager) maintain() {
	for {
		time.Sleep(p.config.PoolRenewalInterval)
		// p.poolCache.Keys() return the list of pool address to maintain
		if err := p.preparePoolsData(context.Background(), p.poolCache.Keys()); err != nil {
			logger.Errorf("could not update pool's stateData, error:%s", err)
		}
	}
}

func (p *PointerSwapPoolManager) preparePoolsData(ctx context.Context, poolAddresses []string) error {
	writeTo := 1 - p.readFrom.Load()

	filteredPoolAddress := p.filterBlacklistedAddresses(poolAddresses)

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, filteredPoolAddress)
	if err != nil {
		return err
	}

	poolByAddress := p.poolFactory.NewPoolByAddress(ctx, poolEntities)
	p.states[writeTo].update(poolByAddress)
	//swapping here
	p.readFrom.Store(writeTo)
	logger.Debugf("PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(poolByAddress))
	return nil
}

func (p *PointerSwapPoolManager) filterBlacklistedAddresses(poolAddresses []string) []string {
	filtered := make([]string, 0, len(poolAddresses))

	for _, address := range poolAddresses {
		if p.config.BlacklistedPoolSet[address] {
			continue
		}

		filtered = append(filtered, address)
	}

	return filtered
}

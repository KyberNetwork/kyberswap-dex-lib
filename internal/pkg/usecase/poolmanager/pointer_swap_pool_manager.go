package poolmanager

import (
	"context"
	"fmt"
	"maps"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

const NState = 3

type PointerSwapPoolManager struct {
	poolFactory        IPoolFactory
	poolRepository     IPoolRepository
	poolRankRepository IPoolRankRepository

	aevmClient            aevmclient.Client
	poolsPublisher        IPoolsPublisher
	publishedStorageIDs   [NState]string
	balanceSlotsUsecase   erc20balanceslot.ICache
	balanceSlotsPreloaded atomic.Bool
	addressSetsPool       sync.Pool // Pool of mapset.Set[common.Address]

	GetPoolsIncludingBasePools IGetPoolsIncludingBasePools

	// We know that fastest state rotation happened every 3 seconds. most requests are done under 1 second.
	// To prevent data corruption without locking, we will use tri-state swapping
	// readFrom: return data requests to the request.
	// writeTo = readFrom+1 % Nstate: data to update to
	// dangling = writeTo+1 % Nstate: data is using by other requests.
	// assumption is that dangling state will soon be freed (all the requests calling into it has exited)
	states   [NState]*LockedState
	readFrom atomic.Int32

	config     Config
	configLock *sync.RWMutex

	// mapset.Set is thread-safety by itself, no need to maintain mutex
	blackListPools mapset.Set[string]

	// mapset.Set is thread-safety by itself, no need to maintain mutex
	faultyPools mapset.Set[string]

	// poolCache control which pool to maintain when there are too many pools, currently poolCache use LRU policy
	// PoolCache is thread-safety, it supports lock mechanism by itself
	poolCache *cachePolicy.Cache[string, struct{}]
}

// NewPointerSwapPoolManager This will take a while to start since it will generate a copy of all Pool
func NewPointerSwapPoolManager(
	ctx context.Context,
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	GetPoolsIncludingBasePools IGetPoolsIncludingBasePools,
	config Config,
	aevmClient aevmclient.Client,
	poolsPublisher IPoolsPublisher,
	balanceSlotsUsecase erc20balanceslot.ICache,
) (*PointerSwapPoolManager, error) {
	states := [NState]*LockedState{}
	for i := 0; i < NState; i++ {
		states[i] = NewLockedState()
	}
	// TODO try policies other than LRU, ex LFU
	poolCache, err := cachePolicy.New[string, struct{}](config.Capacity)
	if err != nil {
		return nil, err
	}

	p := PointerSwapPoolManager{
		states:                     states,
		readFrom:                   atomic.Int32{},
		config:                     config,
		configLock:                 &sync.RWMutex{},
		poolFactory:                poolFactory,
		poolRepository:             poolRepository,
		poolRankRepository:         poolRankRepository,
		GetPoolsIncludingBasePools: GetPoolsIncludingBasePools,
		poolCache:                  poolCache,
		blackListPools:             mapset.NewSet[string](), // we must use thread-safety map here
		faultyPools:                mapset.NewSet[string](), // we must use thread-safety map here
		aevmClient:                 aevmClient,
		poolsPublisher:             poolsPublisher,
		balanceSlotsUsecase:        balanceSlotsUsecase,
		addressSetsPool:            sync.Pool{New: func() any { return mapset.NewThreadUnsafeSet[common.Address]() }},
	}

	if err := p.start(ctx); err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *PointerSwapPoolManager) start(ctx context.Context) error {
	p.readFrom.Store(0)

	// initialize pools to read from DB
	poolAddresses, err := p.findGlobalBestPools(ctx, int64(p.config.Capacity))
	if err != nil {
		return err
	}

	// add in reverse order so that pools with most volume at top of LRU list
	for i := len(poolAddresses) - 1; i >= 0; i-- {
		p.poolCache.Add(poolAddresses[i], struct{}{})
	}

	// init pool manager first state
	p.updateBlackListPool(ctx)
	p.updateFaultyPools(ctx)
	if err := p.preparePoolsData(ctx, poolAddresses); err != nil {
		return err
	}

	// start lifecycle
	go p.reloadBlackListPool(ctx)
	go p.reloadFaultyPools(ctx)
	go p.reloadPoolStates(ctx)

	return nil

}

func (p *PointerSwapPoolManager) findGlobalBestPools(ctx context.Context, poolCount int64) ([]string, error) {
	// for simplified we will fetch only sorted set in composite index
	if p.config.FeatureFlags.IsLiquidityScoreIndexEnable {
		return p.poolRankRepository.FindGlobalBestPoolsByScores(ctx, poolCount, poolrank.SortByLiquidityScoreTvl)
	}

	return p.poolRankRepository.FindGlobalBestPools(ctx, poolCount)
}

// TODO will be refactor to remove this function from pool manager
func (p *PointerSwapPoolManager) GetAEVMClient() aevmclient.Client {
	if p.config.FeatureFlags.IsAEVMEnabled {
		return p.aevmClient
	}

	return nil
}

func (p *PointerSwapPoolManager) reloadBlackListPool(ctx context.Context) {
	for {
		time.Sleep(p.config.BlackListRenewalInterval)
		p.updateBlackListPool(ctx)
	}
}

func (p *PointerSwapPoolManager) reloadFaultyPools(ctx context.Context) {
	for {
		time.Sleep(p.config.FaultyPoolsRenewalInterval)
		p.updateFaultyPools(ctx)
	}
}

func (p *PointerSwapPoolManager) reloadPoolStates(ctx context.Context) {
	for {
		time.Sleep(p.config.PoolRenewalInterval)

		// p.poolCache.Keys() return the list of pool address to maintain
		if err := p.preparePoolsData(ctx, p.poolCache.Keys()); err != nil {
			logger.Errorf(ctx, "could not update pool's stateData, error:%s", err)
		}
	}
}

func (p *PointerSwapPoolManager) updateBlackListPool(ctx context.Context) {
	newBlackedList, err := p.poolRepository.GetPoolsInBlacklist(ctx)
	if err != nil {
		return
	}
	newBlackedListSet := mapset.NewSet(newBlackedList...)
	// The blacklist pool list is rarely updated
	// so implementing a simple equation check can help reduce the need for acquiring an exclusive lock to improve performance.
	if p.blackListPools.Equal(newBlackedListSet) {
		return
	}
	oldDiff := p.blackListPools.Difference(newBlackedListSet).ToSlice()
	newDiff := newBlackedListSet.Difference(p.blackListPools).ToSlice()
	p.blackListPools.RemoveAll(oldDiff...)
	p.blackListPools.Append(newDiff...)
}

func (p *PointerSwapPoolManager) updateFaultyPools(ctx context.Context) {
	newFaultyPools, err := p.poolRepository.GetFaultyPools(ctx)
	if err != nil {
		return
	}
	newFaultyPoolSet := mapset.NewSet(newFaultyPools...)
	// The faulty pool list is rarely updated
	// so implementing a simple equation check can help reduce the need for acquiring an exclusive lock to improve performance.
	if p.faultyPools.Equal(newFaultyPoolSet) {
		return
	}
	oldDiff := p.faultyPools.Difference(newFaultyPoolSet).ToSlice()
	newDiff := newFaultyPoolSet.Difference(p.faultyPools).ToSlice()
	p.faultyPools.RemoveAll(oldDiff...)
	p.faultyPools.Append(newDiff...)
}

func (p *PointerSwapPoolManager) preparePoolsData(ctx context.Context, poolAddresses []string) error {
	writeTo := (p.readFrom.Load() + 1) % NState

	filteredPoolAddress := p.filterInvalidPoolAddresses(poolAddresses)

	availableSources := mapset.NewThreadUnsafeSet(p.config.AvailableSources...)

	poolEntities, err := p.GetPoolsIncludingBasePools.Handle(ctx, filteredPoolAddress, func(pool *entity.Pool) bool {
		return availableSources.ContainsOne(pool.Exchange)
	})
	// reserve memory for pool entities in heap memory, avoid mem allocation burden
	// Any item stored in the Pool may be removed automatically at any time without notification.
	// If the Pool holds the only reference when this happens, the item might be deallocated.
	// So only put pool entities into sync.Pool after using these entities to avoid deallocated, in this case using defer is correct.
	defer mempool.ReserveMany(poolEntities...)
	if err != nil {
		return err
	}
	var stateRoot aevmcommon.Hash
	// if running with aevm
	if p.config.FeatureFlags.IsAEVMEnabled {
		stateRoot, err = p.aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}

	// preload ERC20 balance slots
	if p.config.FeatureFlags.IsAEVMEnabled && !p.balanceSlotsPreloaded.Load() {
		start := time.Now()
		tokens := p.addressSetsPool.Get().(mapset.Set[common.Address])
		tokens.Clear()
		for _, pool := range poolEntities {
			for _, token := range pool.Tokens {
				tokens.Add(common.HexToAddress(token.Address))
			}
		}
		logger.Debugf(ctx, "prepared tokens list from prepared pools took %s", time.Since(start))
		if err := p.balanceSlotsUsecase.PreloadMany(ctx, tokens.ToSlice()); err != nil {
			logger.Warnf(ctx, "could not PreloadMany: %s", err)
		}
		p.balanceSlotsPreloaded.Store(true)
		p.addressSetsPool.Put(tokens)
	}

	pools := p.poolFactory.NewPoolByAddress(ctx, poolEntities, common.Hash(stateRoot))
	if p.config.UseAEVMRemoteFinder && p.poolsPublisher != nil {
		start := time.Now()
		storageID, err := p.poolsPublisher.Publish(ctx, pools)
		if err != nil {
			return fmt.Errorf("could not publish pools: %w", err)
		}
		logger.Infof(ctx, "published pools took %s storageID=%s", time.Since(start).String(), storageID)
		p.publishedStorageIDs[writeTo] = storageID
	}
	p.states[writeTo].update(pools)

	// swapping pointer
	p.swapPointer(writeTo)

	logger.Debugf(ctx, "PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(pools))
	return nil
}

func (p *PointerSwapPoolManager) swapPointer(writeTo int32) {
	// TODO: zero out dangling ref but for now we dont need it.
	// release resources from dangling

	// from now on we read from the latest state.
	p.readFrom.Store(writeTo)
}

func (p *PointerSwapPoolManager) filterInvalidPoolAddresses(poolAddresses []string) []string {
	return lo.Filter(poolAddresses, func(poolAddress string, _ int) bool {
		return !p.blackListPools.ContainsOne(poolAddress) &&
			!p.faultyPools.ContainsOne(poolAddress) &&
			!p.config.BlacklistedPoolSet[poolAddress]
	})

}

// GetStateByPoolAddresses return a reference to pool simulators maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned from here, clone IPoolSimulator before UpdateBalance
func (p *PointerSwapPoolManager) GetStateByPoolAddresses(
	ctx context.Context,
	poolAddresses, dex []string,
	stateRoot common.Hash,
	extraData types.PoolManagerExtraData,
) (*types.FindRouteState, error) {
	filteredPoolAddress := p.filterInvalidPoolAddresses(poolAddresses)
	if len(filteredPoolAddress) == 0 {
		logger.Errorf(ctx,
			"filtered Pool addresses after filterBlacklistedAddresses now equal to 0. Blacklist config %v. PoolAddresses original len: %d",
			p.config.BlacklistedPoolSet, len(poolAddresses))
		return nil, getroute.ErrPoolSetFiltered
	}

	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	if len(dex) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	state, err := p.getPoolStates(ctx, filteredPoolAddress, dex, stateRoot, extraData)
	if err != nil {
		return nil, err
	}

	if len(state.Pools) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	return state, err
}

func (p *PointerSwapPoolManager) getPoolStates(
	ctx context.Context,
	poolAddresses, whitelistDexes []string,
	stateRoot common.Hash,
	extraData types.PoolManagerExtraData,
) (*types.FindRouteState, error) {
	var (
		pools              = make(map[string]poolpkg.IPoolSimulator, len(poolAddresses))
		resultLimits       map[string]map[string]*big.Int
		poolsToFetchFromDB []string
		whitelistDexSet    = sets.NewString(whitelistDexes...)
	)

	readFrom := p.readFrom.Load()
	state := p.states[readFrom]

	// 1. Read all pool entities that are available in read state
	state.lock.RLock()
	for _, key := range poolAddresses {
		if pool, ok := state.poolByAddress[key]; ok {
			if !whitelistDexSet.Has(pool.GetExchange()) {
				continue
			}

			if p.isPMMStalled(pool) {
				logger.Debugf(ctx, "stalling PMM pool %s", pool.GetAddress())
				// refetch again to get the latest data when pmm is stalled
				poolsToFetchFromDB = append(poolsToFetchFromDB, key)
				continue
			}
			pools[key] = pool
		} else {
			poolsToFetchFromDB = append(poolsToFetchFromDB, key)
		}
	}

	// shallow clone limits. limits are supposed to copy on write
	resultLimits = make(map[string]map[string]*big.Int, len(state.limits))
	for dexName, limits := range state.limits {
		resultLimits[dexName] = maps.Clone(limits)
	}
	state.lock.RUnlock()

	if len(pools) == 0 && len(poolsToFetchFromDB) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	// check to return immediately if we don't need to fetch pools from Redis
	if len(poolsToFetchFromDB) == 0 {
		return &types.FindRouteState{
			Pools:                   pools,
			SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits, extraData),
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	// 2. Fetch all pools that are not cached locally from Redis
	poolEntitiesFromDB, err := p.GetPoolsIncludingBasePools.Handle(ctx, poolsToFetchFromDB,
		func(pool *entity.Pool) bool { return whitelistDexSet.Has(pool.Exchange) })

	// reserve memory for pool entities, avoid mem allocation burden
	defer mempool.ReserveMany(poolEntitiesFromDB...)

	if err != nil {
		logger.Errorf(ctx, "poolRepository.FindByAddresses crashed into err : %v", err)
		return &types.FindRouteState{
			Pools:                   pools,
			SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits, extraData),
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	if len(pools) == 0 && len(poolEntitiesFromDB) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	// If there are no pools need to be initialized, return the result from mem state
	if len(poolEntitiesFromDB) == 0 {
		return &types.FindRouteState{
			Pools:                   pools,
			SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits, extraData),
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	// 3. Init pool simulators
	dbPools := p.poolFactory.NewPools(ctx, poolEntitiesFromDB, stateRoot)
	for _, pool := range dbPools {
		if p.isPMMStalled(pool) {
			logger.Debugf(ctx, "stalling PMM pool %s", pool.GetAddress())
			continue
		}
		if !whitelistDexSet.Has(pool.GetExchange()) { // some PoolSimulator might change Exchange
			continue
		}
		pools[pool.GetAddress()] = pool
	}

	if len(pools) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	UpdateLimits(resultLimits, pools)
	return &types.FindRouteState{
		Pools:                   pools,
		SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits, extraData),
		PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
	}, nil
}

func (p *PointerSwapPoolManager) isPMMStalled(pool poolpkg.IPoolSimulator) bool {
	// special case, non-configured stalling threshold is treat as non-enabling stalling threshold
	if p.config.StallingPMMThreshold == 0 {
		return false
	}
	if pool.GetType() == pooltypes.PoolTypes.KyberPMM {
		if pmmPoolMeta, ok := pool.GetMetaInfo("", "").(kyberpmm.RFQMeta); ok {
			createdTime := time.Unix(pmmPoolMeta.Timestamp, 0)
			if time.Since(createdTime) > p.config.StallingPMMThreshold {
				return true
			}
		}
	}
	return false
}

func (p *PointerSwapPoolManager) ApplyConfig(config Config) {
	p.configLock.Lock()
	p.config = config
	p.configLock.Unlock()

	// poolCache is guared by internal lock
	p.poolCache.Resize(config.Capacity)
}

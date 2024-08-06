package poolmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/util/sets"
)

const NState = 3

type PointerSwapPoolManager struct {
	poolFactory        IPoolFactory
	poolRepository     IPoolRepository
	poolRankRepository IPoolRankRepository

	aevmClient          aevmclient.Client
	poolsPublisher      IPoolsPublisher
	publishedStorageIDs [NState]string

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

	blackListPools []string
	blackListlock  *sync.RWMutex

	faultyPools     []string
	faultyPoolsLock *sync.RWMutex

	// poolCache control which pool to maintain when there are too many pools, currently poolCache use LRU policy
	// PoolCache is thread-safety, it supports lock mechanism by itself
	poolCache *cachePolicy.Cache[string, struct{}]
}

type LockedState struct {
	poolByAddress map[string]poolpkg.IPoolSimulator
	limits        map[string]map[string]*big.Int
	lock          *sync.RWMutex
}

func NewLockedState() *LockedState {

	var limits = make(map[string]map[string]*big.Int)
	limits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.NativeV1] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.LimitOrder] = make(map[string]*big.Int)

	return &LockedState{
		poolByAddress: make(map[string]poolpkg.IPoolSimulator),
		limits:        limits,
		lock:          &sync.RWMutex{},
	}
}

func (s *LockedState) update(poolByAddress map[string]poolpkg.IPoolSimulator) {
	s.lock.Lock()
	defer s.lock.Unlock()

	//update the inventory and tokenToPoolAddress list
	for poolAddress := range poolByAddress {
		//soft copy to save some lookupTime:
		pool := poolByAddress[poolAddress]

		dexLimit, avail := s.limits[pool.GetType()]
		if !avail {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
				dexLimit[k] = v
			}
		}
	}
	s.poolByAddress = poolByAddress
	// Optimize graph traversal by using tokenToPoolAddress list
}

// NewPointerSwapPoolManager This will take a while to start since it will generate a copy of all Pool
func NewPointerSwapPoolManager(
	ctx context.Context,
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	config Config,
	aevmClient aevmclient.Client,
	poolsPublisher IPoolsPublisher,
) (*PointerSwapPoolManager, error) {
	states := [NState]*LockedState{}
	for i := 0; i < NState; i++ {
		states[i] = NewLockedState()
	}
	//TODO try policies other than LRU
	poolCache, err := cachePolicy.New[string, struct{}](config.Capacity)
	if err != nil {
		return nil, err
	}

	p := PointerSwapPoolManager{
		states:             states,
		readFrom:           atomic.Int32{},
		config:             config,
		configLock:         &sync.RWMutex{},
		poolFactory:        poolFactory,
		poolRepository:     poolRepository,
		poolRankRepository: poolRankRepository,
		poolCache:          poolCache,
		blackListlock:      &sync.RWMutex{},
		faultyPoolsLock:    &sync.RWMutex{},
		aevmClient:         aevmClient,
		poolsPublisher:     poolsPublisher,
	}

	if err := p.start(ctx); err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *PointerSwapPoolManager) start(ctx context.Context) error {
	p.readFrom.Store(0)

	// initialize pools to read from DB
	poolAddresses, err := p.poolRankRepository.FindGlobalBestPools(ctx, int64(p.config.Capacity))
	if err != nil {
		return err
	}

	// add in reverse order so that pools with most volume at top of LRU list
	for i := len(poolAddresses) - 1; i >= 0; i-- {
		p.poolCache.Add(poolAddresses[i], struct{}{})
	}

	p.updateBlackListPool(ctx)
	p.updateFaultyPools(ctx)
	if err := p.preparePoolsData(ctx, poolAddresses); err != nil {
		return err
	}

	go p.reloadBlackListPool(ctx)
	go p.reloadFaultyPools(ctx)
	go p.reloadPoolStates(ctx)

	return nil

}

// TODO will be refactor to remove this function from pool manager
func (p *PointerSwapPoolManager) GetAEVMClient() aevmclient.Client {
	if p.config.UseAEVM {
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
	blackedList, err := p.poolRepository.GetPoolsInBlacklist(ctx)
	if err == nil {
		p.blackListlock.Lock()
		p.blackListPools = blackedList
		p.blackListlock.Unlock()
	}
}

func (p *PointerSwapPoolManager) updateFaultyPools(ctx context.Context) {
	faultyPools, err := p.poolRepository.GetFaultyPools(ctx)
	if err == nil {
		p.faultyPoolsLock.Lock()
		p.faultyPools = faultyPools
		p.faultyPoolsLock.Unlock()
	}
}

func (p *PointerSwapPoolManager) preparePoolsData(ctx context.Context, poolAddresses []string) error {
	writeTo := (p.readFrom.Load() + 1) % NState

	filteredPoolAddress := p.filterInvalidPoolAddresses(poolAddresses)

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, filteredPoolAddress)
	defer mempool.ReserveMany(poolEntities)
	if err != nil {
		return err
	}
	var stateRoot aevmcommon.Hash
	// if running with aevm
	if p.config.UseAEVM {
		stateRoot, err = p.aevmClient.LatestStateRoot(ctx)
		if err != nil {
			return fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
	poolByAddress := p.poolFactory.NewPoolByAddress(ctx, poolEntities, common.Hash(stateRoot))
	if p.poolsPublisher != nil {
		start := time.Now()
		storageID, err := p.poolsPublisher.Publish(ctx, poolByAddress)
		if err != nil {
			return fmt.Errorf("could not publish pools: %w", err)
		}
		logger.Infof(ctx, "published pools took %s storageID=%s", time.Since(start).String(), storageID)
		p.publishedStorageIDs[writeTo] = storageID
	}
	p.states[writeTo].update(poolByAddress)

	//swapping pointer
	p.swapPointer(writeTo)

	logger.Debugf(ctx, "PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(poolByAddress))
	return nil
}

func (p *PointerSwapPoolManager) swapPointer(writeTo int32) {
	// TODO: zero out dangling ref but for now we dont need it.
	// release resources from dangling

	//from now on we read from the latest state.
	p.readFrom.Store(writeTo)

}

func (p *PointerSwapPoolManager) getDynamicBlackListSet() mapset.Set[string] {
	p.blackListlock.RLock()
	defer p.blackListlock.RUnlock()

	// TODO: check to use thread-safe set to maintain blackListPools, remove mutex lock
	return mapset.NewThreadUnsafeSet(p.blackListPools...)
}

func (p *PointerSwapPoolManager) getFaultyPoolListSet() mapset.Set[string] {
	p.faultyPoolsLock.RLock()
	defer p.faultyPoolsLock.RUnlock()

	// TODO: check to use thread-safe set to maintain faultyPools, remove mutex lock
	return mapset.NewThreadUnsafeSet(p.faultyPools...)
}

func (p *PointerSwapPoolManager) filterInvalidPoolAddresses(poolAddresses []string) []string {
	dynamicBlackListSet := p.getDynamicBlackListSet()
	faultyPoolSet := p.getFaultyPoolListSet()
	filters := func(poolAddress string, _ int) bool {
		return !dynamicBlackListSet.ContainsOne(poolAddress) && !faultyPoolSet.ContainsOne(poolAddress) && !p.config.BlacklistedPoolSet[poolAddress]
	}

	return lo.Filter(poolAddresses, filters)

}

// GetStateByPoolAddresses return a reference to pools maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned here, clone IPool before UpdateBalance
func (p *PointerSwapPoolManager) GetStateByPoolAddresses(ctx context.Context, poolAddresses, dex []string, stateRoot common.Hash) (*types.FindRouteState, error) {
	filteredPoolAddress := p.filterInvalidPoolAddresses(poolAddresses)
	if len(filteredPoolAddress) == 0 {
		logger.Errorf(ctx, "filtered Pool addresses after filterBlacklistedAddresses now equal to 0. Blacklist config %v. PoolAddresses original len: %d", p.config.BlacklistedPoolSet, len(poolAddresses))
		return nil, getroute.ErrPoolSetFiltered
	}

	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	if len(dex) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	state, err := p.getPools(ctx, filteredPoolAddress, dex, stateRoot)
	if err != nil {
		return nil, err
	}

	if len(state.Pools) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	return state, err
}

func (p *PointerSwapPoolManager) getPools(ctx context.Context, poolAddresses, dex []string, stateRoot common.Hash) (*types.FindRouteState, error) {
	var (
		resultPoolByAddress = make(map[string]poolpkg.IPoolSimulator, len(poolAddresses))
		resultLimits        = make(map[string]map[string]*big.Int)
		poolsToFetchFromDB  []string
		dexSet              = sets.NewString(dex...)
	)

	readFrom := p.readFrom.Load()

	// 1. Read all pool entities that are available in read state
	p.states[readFrom].lock.RLock()
	for _, key := range poolAddresses {
		if pool, ok := p.states[readFrom].poolByAddress[key]; ok {
			if !dexSet.Has(pool.GetExchange()) {
				continue
			}

			if p.isPMMStalled(pool) {
				logger.Debugf(ctx, "stalling PMM pool %s", pool.GetAddress())
				// refetch again to get the latest data when pmm is stalled
				poolsToFetchFromDB = append(poolsToFetchFromDB, key)
				continue
			}
			resultPoolByAddress[key] = pool
		} else {
			poolsToFetchFromDB = append(poolsToFetchFromDB, key)
		}
	}
	//given a clone of limit
	for dexName, limits := range p.states[readFrom].limits {
		resLimit := make(map[string]*big.Int, len(limits))
		for key, l := range limits {
			resLimit[key] = big.NewInt(0).Set(l)
		}
		resultLimits[dexName] = resLimit
	}
	p.states[readFrom].lock.RUnlock()

	// return must be happened after unlock, check to return filter out all of pools here
	if len(resultPoolByAddress) == 0 && len(poolsToFetchFromDB) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	// check to return immediately if we don't need to fetch pools from Redis
	if len(poolsToFetchFromDB) == 0 {
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits),
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	// 2. Find all pool data which are not available in read state, fetch them from Redis
	poolEntities, err := p.poolRepository.FindByAddresses(ctx, poolsToFetchFromDB)
	if err != nil {
		logger.Errorf(ctx, "poolRepository.FindByAddresses crashed into err ", err)
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               nil,
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}
	// reserve memory for pool entities, avoid mem allocation burden
	defer mempool.ReserveMany(poolEntities)

	// After getting pool entities from Redis, filter out pool entities with dex non-included in dex set
	filteredPoolEntities := lo.Filter(poolEntities, func(pool *entity.Pool, i int) bool {
		return dexSet.Has(pool.Exchange)
	})

	//  check to return filter out all of pools again here
	if len(resultPoolByAddress) == 0 && len(filteredPoolEntities) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	// If there are no pools need to be initialized, return the result from mem state
	if len(filteredPoolEntities) == 0 {
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits),
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	// 3. Process specifically for curve meta pools, fetch base pools (if they have not been fetched) to init curve meta
	curveMetaBasePools, err := p.listCurveMetaBasePools(ctx, p.poolRepository, filteredPoolEntities)
	if err != nil {
		logger.Debugf(ctx, "failed to load curve-meta base pool %v", err)
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               nil,
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}
	filteredPoolEntities = append(filteredPoolEntities, curveMetaBasePools...)

	// 4. Init pool simulators
	poolInterfaces := p.poolFactory.NewPools(ctx, filteredPoolEntities, stateRoot)
	for i := range poolInterfaces {
		if p.isPMMStalled(poolInterfaces[i]) {
			logger.Debugf(ctx, "stalling PMM pool %s", poolInterfaces[i].GetAddress())
			continue
		}
		resultPoolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	if len(resultPoolByAddress) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	return &types.FindRouteState{
		Pools:                   resultPoolByAddress,
		SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits),
		PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
	}, nil
}

// listCurveMetaBasePools collects base pools of curveMeta pools
// - collects already fetched curveBase and curvePainOracle pools
// - for each curveMeta pool
//   - decode its staticExtra to get its basePool address
//   - if it hasn't been fetched, fetch the pool data
func (p *PointerSwapPoolManager) listCurveMetaBasePools(
	ctx context.Context,
	poolRepository IPoolRepository,
	pools []*entity.Pool,
) ([]*entity.Pool, error) {
	var (
		// alreadyFetchedSet contains fetched pool ids
		alreadyFetchedSet = map[string]bool{}
		// poolAddresses contains pool addresses to fetch
		poolAddresses = sets.NewString()
	)

	for _, pool := range pools {
		if pool.Type == pooltypes.PoolTypes.CurveBase ||
			pool.Type == pooltypes.PoolTypes.CurveStablePlain ||
			pool.Type == pooltypes.PoolTypes.CurvePlainOracle ||
			pool.Type == pooltypes.PoolTypes.CurveAave {
			alreadyFetchedSet[pool.Address] = true
		}
	}

	for _, pool := range pools {
		if pool.Type != pooltypes.PoolTypes.CurveMeta && pool.Type != pooltypes.PoolTypes.CurveStableMetaNg {
			continue
		}

		var staticExtra struct {
			BasePool string `json:"basePool"`
		}
		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			logger.WithFields(ctx, logger.Fields{
				"pool.Address": pool.Address,
				"pool.Type":    pool.Type,
				"error":        err,
			}).Warn("unable to unmarshal staticExtra")

			continue
		}

		if _, ok := alreadyFetchedSet[staticExtra.BasePool]; ok {
			continue
		}

		poolAddresses.Insert(strings.ToLower(staticExtra.BasePool))
	}

	return poolRepository.FindByAddresses(ctx, poolAddresses.List())
}

func (p *PointerSwapPoolManager) isPMMStalled(pool poolpkg.IPoolSimulator) bool {
	//special case, non-configured stalling threshold is treat as non-enabling stalling threshold
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

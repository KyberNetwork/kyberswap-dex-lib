package poolmanager

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"k8s.io/apimachinery/pkg/util/sets"

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

	// We know that fastest state rotation happened every 3 seconds. most requests are done under 1 second.
	// To prevent data corruption without locking, we will use tri-state swapping
	// readFrom: return data requests to the request.
	// writeTo = readFrom+1 % Nstate: data to update to
	// dangling = writeTo+1 % Nstate: data is using by other requests.
	// assumption is that dangling state will soon be freed (all the requests calling into it has exited)
	states   [NState]*LockedState
	readFrom atomic.Int32
	config   Config

	blackListPools []string
	// poolCache control which pool to maintain when there are too many pools
	// currently poolCache use LRU policy
	poolCache *cachePolicy.Cache[string, struct{}]

	aevmClient          aevmclient.Client
	poolsPublisher      IPoolsPublisher
	publishedStorageIDs [NState]string

	lock *sync.RWMutex
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

	s.lock.Unlock()
}

// NewNonMaintenancePointerSwapPoolManager return a Pool Manager with only pool addresses and not pool data
// any service using this implementation will have to call Reload() on its own
func NewNonMaintenancePointerSwapPoolManager(
	ctx context.Context,
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	config Config,
	aevmClient aevmclient.Client,
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
		aevmClient:         aevmClient,
	}
	p.readFrom.Store(0)

	var stateRoot aevmcommon.Hash
	// if running with aevm
	if p.config.UseAEVM {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
	p.updateBlackListPool(ctx)

	if err = p.preparePoolsData(context.Background(), poolAddresses, common.Hash(stateRoot)); err != nil {
		return nil, err
	}
	//go p.maintain()
	return &p, nil
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
		aevmClient:         aevmClient,
		poolsPublisher:     poolsPublisher,
	}
	p.readFrom.Store(0)

	var stateRoot aevmcommon.Hash
	// if running with aevm
	if p.config.UseAEVM {
		stateRoot, err = aevmClient.LatestStateRoot(ctx)
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
	if p.config.BlackListRenewalInterval == 0 {
		p.config.BlackListRenewalInterval = DefaultBlackListRenewalInterval
	}
	p.updateBlackListPool(ctx)
	if err = p.preparePoolsData(context.Background(), poolAddresses, common.Hash(stateRoot)); err != nil {
		return nil, err
	}
	go p.maintain(ctx)
	return &p, nil
}

func (p *PointerSwapPoolManager) GetAEVMClient() aevmclient.Client {
	if p.config.UseAEVM {
		return p.aevmClient
	}
	return nil
}

func (p *PointerSwapPoolManager) ApplyConfig(config Config) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.config = config
	p.poolCache.Resize(config.Capacity)
}

func (p *PointerSwapPoolManager) updateBlackListPool(ctx context.Context) {
	var (
		blackedList []string
		err         error
		counter     = 0
	)
	//since the wait time of updateBlackList can be quite long, we retry 3 times if it gets err
	for {
		blackedList, err = p.poolRepository.GetPoolsInBlacklist(ctx)
		if err != nil {
			logger.Errorf(ctx, "error checking pool blacklist. Err: %s", err)
			counter++
		} else {
			break
		}
		if counter > 3 {
			logger.Errorf(ctx, "failed to get blackList data 3 times")
			break
		}
		time.Sleep(time.Second)
	}
	if err == nil {
		p.lock.Lock()
		p.blackListPools = blackedList
		p.lock.Unlock()
	}
}

// GetStateByPoolAddresses return a reference to pools maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned here, clone IPool before UpdateBalance
func (p *PointerSwapPoolManager) GetStateByPoolAddresses(ctx context.Context, poolAddresses, dex []string, stateRoot common.Hash) (*types.FindRouteState, error) {
	filteredPoolAddress := p.filterBlacklistedAddresses(poolAddresses)
	if len(filteredPoolAddress) == 0 {
		logger.Errorf(ctx, "filtered Pool addresses after filterBlacklistedAddresses now equal to 0. Blacklist config %v. PoolAddresses original len: %d", p.config.BlacklistedPoolSet, len(poolAddresses))
		return nil, getroute.ErrPoolSetFiltered
	}
	filteredPoolAddress = p.excludeFaultyPools(ctx, filteredPoolAddress)
	if len(filteredPoolAddress) == 0 {
		logger.Errorf(ctx, "filtered Pool address after excludeFaultyPools now equal to 0. PoolAddresses original len: %d", len(poolAddresses))
		return nil, getroute.ErrPoolSetFiltered
	}
	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	readFrom := p.readFrom.Load()
	state, err := p.getPools(ctx, filteredPoolAddress, dex, readFrom, stateRoot)
	if err != nil {
		return nil, err
	}

	if len(filteredPoolAddress) > 0 && len(state.Pools) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	return state, err
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

func (p *PointerSwapPoolManager) getPools(ctx context.Context, poolAddresses, dex []string, readFrom int32, stateRoot common.Hash) (*types.FindRouteState, error) {
	var (
		resultPoolByAddress = make(map[string]poolpkg.IPoolSimulator, len(poolAddresses))
		resultLimits        = make(map[string]map[string]*big.Int)
		poolsToFetchFromDB  []string
		dexSet              = sets.NewString(dex...)
		isFiltered          = false
	)

	if len(dex) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	p.states[readFrom].lock.RLock()

	for _, key := range poolAddresses {
		if pool, ok := p.states[readFrom].poolByAddress[key]; ok {
			if dexSet.Has(pool.GetExchange()) {
				if p.isPMMStalled(pool) {
					logger.Debugf(ctx, "stalling PMM pool %s", pool.GetAddress())
					poolsToFetchFromDB = append(poolsToFetchFromDB, key) // refetch again to get the latest data
					continue
				}
				resultPoolByAddress[key] = pool
			} else {
				isFiltered = true
			}
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

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, poolsToFetchFromDB)
	if err != nil {
		logger.Errorf(ctx, "poolRepository.FindByAddresses crashed into err ", err)
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               nil, // just wonder why don't we return resultLimits here :think:
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}

	defer mempool.ReserveMany(poolEntities)

	filteredPoolEntities := make([]*entity.Pool, 0, len(poolEntities))
	for i := range poolEntities {
		if dexSet.Has(poolEntities[i].Exchange) {
			filteredPoolEntities = append(filteredPoolEntities, poolEntities[i])
		} else {
			isFiltered = true
		}
	}

	curveMetaBasePools, err := listCurveMetaBasePools(ctx, p.poolRepository, filteredPoolEntities)
	if err != nil {
		logger.Debugf(ctx, "failed to load curve-meta base pool %v", err)
		return &types.FindRouteState{
			Pools:                   resultPoolByAddress,
			SwapLimit:               nil, // just wonder why don't we return resultLimits here :think:
			PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
		}, nil
	}
	filteredPoolEntities = append(filteredPoolEntities, curveMetaBasePools...)

	poolInterfaces := p.poolFactory.NewPools(ctx, filteredPoolEntities, stateRoot)
	for i := range poolInterfaces {
		if p.isPMMStalled(poolInterfaces[i]) {
			logger.Debugf(ctx, "stalling PMM pool %s", poolInterfaces[i].GetAddress())
			continue
		}
		resultPoolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	if len(resultPoolByAddress) == 0 {
		if isFiltered {
			return nil, getroute.ErrPoolSetFiltered
		}
		return nil, getroute.ErrPoolSetEmpty
	}

	return &types.FindRouteState{
		Pools:                   resultPoolByAddress,
		SwapLimit:               p.poolFactory.NewSwapLimit(resultLimits),
		PublishedPoolsStorageID: p.publishedStorageIDs[readFrom],
	}, nil
}

func (p *PointerSwapPoolManager) Reload(ctx context.Context) error {
	var (
		stateRoot aevmcommon.Hash
		err       error
	)
	// if running with aevm
	if p.config.UseAEVM {
		stateRoot, err = p.aevmClient.LatestStateRoot(ctx)
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}

	return p.preparePoolsData(context.Background(), p.poolCache.Keys(), common.Hash(stateRoot))
}

func (p *PointerSwapPoolManager) reloadBlackListPool(ctx context.Context) {
	for {
		p.updateBlackListPool(ctx)
		time.Sleep(p.config.BlackListRenewalInterval)
	}
}

func (p *PointerSwapPoolManager) maintain(ctx context.Context) {
	go p.reloadBlackListPool(ctx)

	for {
		time.Sleep(p.config.PoolRenewalInterval)

		// p.poolCache.Keys() return the list of pool address to maintain
		if err := p.Reload(ctx); err != nil {
			logger.Errorf(ctx, "could not update pool's stateData, error:%s", err)
		}
	}
}

func (p *PointerSwapPoolManager) swapPointer(writeTo int32) {
	//TODO: zero out dangling ref but for now we dont need it.
	// release resources from dangling

	//from now on we read from the latest state.
	p.readFrom.Store(writeTo)

}

func (p *PointerSwapPoolManager) preparePoolsData(ctx context.Context, poolAddresses []string, stateRoot common.Hash) error {
	writeTo := (p.readFrom.Load() + 1) % NState

	filteredPoolAddress := p.filterBlacklistedAddresses(poolAddresses)

	filteredPoolAddress = p.excludeFaultyPools(ctx, filteredPoolAddress)

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, filteredPoolAddress)
	defer mempool.ReserveMany(poolEntities)
	if err != nil {
		return err
	}

	poolByAddress := p.poolFactory.NewPoolByAddress(ctx, poolEntities, stateRoot)
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
	//swapping here
	p.swapPointer(writeTo)
	logger.Debugf(ctx, "PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(poolByAddress))
	return nil
}

func (p *PointerSwapPoolManager) getBlackListSet() sets.String {
	p.lock.RLock()
	defer p.lock.RUnlock()
	//sets.NewString return a copy
	return sets.NewString(p.blackListPools...)
}

func (p *PointerSwapPoolManager) filterBlacklistedAddresses(poolAddresses []string) []string {
	filtered := make([]string, 0, len(poolAddresses))

	for _, address := range poolAddresses {
		if p.config.BlacklistedPoolSet[address] {
			continue
		}

		filtered = append(filtered, address)
	}

	blackListSet := p.getBlackListSet()
	validPools := make([]string, 0, len(filtered))
	for _, address := range filtered {
		if blackListSet.Has(address) {
			continue
		}

		validPools = append(validPools, address)
	}

	return validPools
}

func (m *PointerSwapPoolManager) excludeFaultyPools(ctx context.Context, addresses []string) []string {
	faultyPools, err := m.poolRepository.GetFaultyPools(ctx)
	if err != nil {
		logger.Errorf(ctx, "[PointerSwapPoolManager] excludeFaultyPools getFaultyPools error %v", err)
		return addresses
	}
	poolSet := mapset.NewSet[string](faultyPools...)

	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if !poolSet.ContainsOne(addr) {
			result = append(result, addr)
		}
	}

	return result
}

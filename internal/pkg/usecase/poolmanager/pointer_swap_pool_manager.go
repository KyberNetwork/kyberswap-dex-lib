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
	"github.com/ethereum/go-ethereum/common"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"k8s.io/apimachinery/pkg/util/sets"

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

	// poolCache control which pool to maintain when there are too many pools
	// currently poolCache use LRU policy
	poolCache *cachePolicy.Cache[string, struct{}]

	aevmClient aevmclient.Client

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
			dexLimit[k] = v
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
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
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
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return nil, fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}
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

// GetStateByPoolAddresses return a reference to pools maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned here, clone IPool before UpdateBalance
func (p *PointerSwapPoolManager) GetStateByPoolAddresses(ctx context.Context, poolAddresses, dex []string, stateRoot common.Hash) (*types.FindRouteState, error) {
	filteredPoolAddress := p.filterBlacklistedAddresses(ctx, poolAddresses)
	filteredPoolAddress = p.excludeFaultyPools(ctx, filteredPoolAddress, p.config)

	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	readFrom := p.readFrom.Load()
	state := p.getPools(ctx, filteredPoolAddress, dex, readFrom, stateRoot)

	return state, nil
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

func (p *PointerSwapPoolManager) getPools(ctx context.Context, poolAddresses, dex []string, readFrom int32, stateRoot common.Hash) *types.FindRouteState {
	var (
		resultPoolByAddress = make(map[string]poolpkg.IPoolSimulator, len(poolAddresses))
		resultLimits        = make(map[string]map[string]*big.Int)
		poolsToFetchFromDB  []string
		dexSet              = sets.NewString(dex...)
	)

	p.states[readFrom].lock.RLock()

	for _, key := range poolAddresses {
		if pool, ok := p.states[readFrom].poolByAddress[key]; ok {
			if dexSet.Has(pool.GetExchange()) {
				if p.isPMMStalled(pool) {
					logger.Debugf(ctx, "stalling PMM pool %s", pool.GetAddress())
					continue
				}
				resultPoolByAddress[key] = pool
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
		return &types.FindRouteState{
			Pools:     resultPoolByAddress,
			SwapLimit: nil,
		}
	}

	defer mempool.ReserveMany(poolEntities)

	filteredPoolEntities := make([]*entity.Pool, 0, len(poolEntities))
	for i := range poolEntities {
		if dexSet.Has(poolEntities[i].Exchange) {
			filteredPoolEntities = append(filteredPoolEntities, poolEntities[i])
		}
	}

	curveMetaBasePools, err := listCurveMetaBasePools(ctx, p.poolRepository, filteredPoolEntities)
	if err != nil {
		logger.Debugf(ctx, "failed to load curve-meta base pool %v", err)
		return &types.FindRouteState{
			Pools:     resultPoolByAddress,
			SwapLimit: nil,
		}
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

	return &types.FindRouteState{
		Pools:     resultPoolByAddress,
		SwapLimit: p.poolFactory.NewSwapLimit(resultLimits),
	}
}

func (p *PointerSwapPoolManager) Reload(ctx context.Context) error {
	var (
		stateRoot aevmcommon.Hash
		err       error
	)
	// if running with aevm
	if p.config.UseAEVM {
		stateRoot, err = p.aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf(ctx, "could not get latest state root for aevm %s", err)
			return fmt.Errorf("[AEVM] could not get latest state root for AEVM pools: %w", err)
		}
	}

	return p.preparePoolsData(context.Background(), p.poolCache.Keys(), common.Hash(stateRoot))
}

func (p *PointerSwapPoolManager) maintain(ctx context.Context) {
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

	filteredPoolAddress := p.filterBlacklistedAddresses(ctx, poolAddresses)

	filteredPoolAddress = p.excludeFaultyPools(ctx, filteredPoolAddress, p.config)

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, filteredPoolAddress)
	defer mempool.ReserveMany(poolEntities)
	if err != nil {
		return err
	}

	poolByAddress := p.poolFactory.NewPoolByAddress(ctx, poolEntities, stateRoot)
	p.states[writeTo].update(poolByAddress)
	//swapping here
	p.swapPointer(writeTo)
	logger.Debugf(ctx, "PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(poolByAddress))
	return nil
}

func (p *PointerSwapPoolManager) filterBlacklistedAddresses(ctx context.Context, poolAddresses []string) []string {
	filtered := make([]string, 0, len(poolAddresses))

	for _, address := range poolAddresses {
		if p.config.BlacklistedPoolSet[address] {
			continue
		}

		filtered = append(filtered, address)
	}

	// check again with Redis
	isInBlacklist, err := p.poolRepository.CheckPoolsInBlacklist(ctx, filtered)
	if err != nil {
		logger.Errorf(ctx, "error checking pool blacklist %v", err)
		return nil
	}
	validPools := make([]string, 0, len(filtered))
	for idx, address := range filtered {
		if isInBlacklist[idx] {
			continue
		}

		validPools = append(validPools, address)
	}

	return validPools
}

func (m *PointerSwapPoolManager) excludeFaultyPools(ctx context.Context, addresses []string, config Config) []string {
	// we need to add threshold for expire time, any faulty pools are not expired but have the expire time near threshold considered as expired
	start := time.Now().UnixMilli() + config.FaultyPoolsExpireThreshold.Milliseconds()
	offset := int64(0)

	poolSet := make(map[string]struct{})
	for {
		faultyPools, err := m.poolRepository.GetFaultyPools(ctx, start, offset, config.MaxFaultyPoolSize)
		if err != nil {
			return addresses
		}

		for _, pool := range faultyPools {
			poolSet[pool] = struct{}{}
		}

		// if faulty pool size is smaller than max config, then we already got the whole list
		// adding len(faultyPools) == 0 to easy unit test (because MaxFaultyPoolSize is usually configured to 0 when unit test)
		if len(faultyPools) == 0 || int64(len(faultyPools)) < config.MaxFaultyPoolSize {
			break
		}
		offset += config.MaxFaultyPoolSize
	}

	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if _, ok := poolSet[addr]; !ok {
			result = append(result, addr)
		}
	}

	return result
}

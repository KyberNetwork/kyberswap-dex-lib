package poolmanager

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
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

	aevmClient aevmclient.Client

	lock *sync.RWMutex
}

type LockedState struct {
	poolByAddress map[string]poolpkg.IPoolSimulator
	pmmInventory  map[string]*big.Int
	lock          *sync.RWMutex
}

func (s *LockedState) update(poolByAddress map[string]poolpkg.IPoolSimulator) {
	s.lock.Lock()
	s.poolByAddress = poolByAddress
	//update the inventory
	for key := range poolByAddress {
		if poolByAddress[key].GetType() == constant.PoolTypes.KyberPMM {
			tokens := s.poolByAddress[key].GetTokens()
			rsv := s.poolByAddress[key].GetReserves()
			for i, tok := range tokens {
				s.pmmInventory[tok] = big.NewInt(0).Set(rsv[i]) //clone here.
			}
		}
	}
	s.lock.Unlock()
}

// NewNonMaintenancePointerSwapPoolManager return a Pool Manager with only pool addresses and not pool data
// any service using this implementation will have to call Reload() on its own
func NewNonMaintenancePointerSwapPoolManager(
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	config Config,
	aevmClient aevmclient.Client,
) (*PointerSwapPoolManager, error) {
	states := [2]*LockedState{}
	for i := 0; i < 2; i++ {
		states[i] = &LockedState{
			poolByAddress: make(map[string]poolpkg.IPoolSimulator),
			pmmInventory:  make(map[string]*big.Int),
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
		aevmClient:         aevmClient,
	}
	p.readFrom.Store(0)

	var stateRoot aevmcommon.Hash
	// if running with aevm
	if aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf("could not get latest state root for aevm %s", err)
			return nil, err
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
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	poolRankRepository IPoolRankRepository,
	config Config,
	aevmClient aevmclient.Client,
) (*PointerSwapPoolManager, error) {
	states := [2]*LockedState{}
	for i := 0; i < 2; i++ {
		states[i] = &LockedState{
			poolByAddress: make(map[string]poolpkg.IPoolSimulator),
			pmmInventory:  make(map[string]*big.Int),
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
		aevmClient:         aevmClient,
	}
	p.readFrom.Store(0)

	var stateRoot aevmcommon.Hash
	// if running with aevm
	if aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf("could not get latest state root for aevm %s", err)
			return nil, err
		}
	}
	if err = p.preparePoolsData(context.Background(), poolAddresses, common.Hash(stateRoot)); err != nil {
		return nil, err
	}
	go p.maintain()
	return &p, nil
}

func (p *PointerSwapPoolManager) GetAEVMClient() aevmclient.Client {
	return p.aevmClient
}

func (p *PointerSwapPoolManager) ApplyConfig(config Config) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.config = config
	p.poolCache.Resize(config.Capacity)
}

// GetPoolByAddress return a reference to pools maintained by `PointerSwapPoolManager`
// Therefore, do not modify IPool returned here, clone IPool before UpdateBalance
func (p *PointerSwapPoolManager) GetPoolByAddress(ctx context.Context, poolAddresses, dex []string, stateRoot common.Hash) (map[string]poolpkg.IPoolSimulator, *poolpkg.Inventory, error) {
	filteredPoolAddress := p.filterBlacklistedAddresses(ctx, poolAddresses)

	// update cache policy
	for _, poolAddress := range filteredPoolAddress {
		p.poolCache.Add(poolAddress, struct{}{})
	}

	readFrom := p.readFrom.Load()
	pools, pmmInventory := p.getPools(ctx, filteredPoolAddress, dex, readFrom, stateRoot)
	return pools, pmmInventory, nil
}

func (p *PointerSwapPoolManager) getPools(ctx context.Context, poolAddresses, dex []string, readFrom int32, stateRoot common.Hash) (map[string]poolpkg.IPoolSimulator, *poolpkg.Inventory) {
	var (
		resultPoolByAddress = make(map[string]poolpkg.IPoolSimulator, len(poolAddresses))
		resultPMMInventory  = make(map[string]*big.Int)
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
	//given a clone of pmm balance
	for token, balance := range p.states[readFrom].pmmInventory {
		resultPMMInventory[token] = big.NewInt(0).Set(balance)
	}
	p.states[readFrom].lock.RUnlock()

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, poolsToFetchFromDB)
	if err != nil {
		return resultPoolByAddress, nil
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
		logger.Debugf("failed to load curve-meta base pool %v", err)
		return resultPoolByAddress, nil
	}
	filteredPoolEntities = append(filteredPoolEntities, curveMetaBasePools...)

	poolInterfaces := p.poolFactory.NewPools(ctx, filteredPoolEntities, stateRoot)
	for i := range poolInterfaces {
		resultPoolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	return resultPoolByAddress, poolpkg.NewInventory(resultPMMInventory)
}

func (p *PointerSwapPoolManager) Reload() error {
	var (
		stateRoot aevmcommon.Hash
		err       error
	)
	// if running with aevm
	if p.aevmClient != nil {
		stateRoot, err = p.aevmClient.LatestStateRoot()
		if err != nil {
			logger.Errorf("could not get latest state root for aevm %s", err)
			return err
		}
	}

	return p.preparePoolsData(context.Background(), p.poolCache.Keys(), common.Hash(stateRoot))
}

func (p *PointerSwapPoolManager) maintain() {
	for {
		time.Sleep(p.config.PoolRenewalInterval)

		// p.poolCache.Keys() return the list of pool address to maintain
		if err := p.Reload(); err != nil {
			logger.Errorf("could not update pool's stateData, error:%s", err)
		}
	}
}

func (p *PointerSwapPoolManager) preparePoolsData(ctx context.Context, poolAddresses []string, stateRoot common.Hash) error {
	writeTo := 1 - p.readFrom.Load()

	filteredPoolAddress := p.filterBlacklistedAddresses(ctx, poolAddresses)

	poolEntities, err := p.poolRepository.FindByAddresses(ctx, filteredPoolAddress)
	defer mempool.ReserveMany(poolEntities)
	if err != nil {
		return err
	}

	poolByAddress := p.poolFactory.NewPoolByAddress(ctx, poolEntities, stateRoot)
	p.states[writeTo].update(poolByAddress)
	//swapping here
	p.readFrom.Store(writeTo)
	logger.Debugf("PointerSwapPoolManager.preparePoolsData > Prepared %v pools", len(poolByAddress))
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
		logger.Errorf("error checking pool blacklist %v", err)
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

package poolmanager

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/common"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type PoolManager struct {
	poolRepository IPoolRepository
	poolFactory    IPoolFactory

	config Config

	mu sync.RWMutex
}

func NewPoolManager(
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	config Config,
) *PoolManager {
	return &PoolManager{
		poolRepository: poolRepository,
		poolFactory:    poolFactory,
		config:         config,
	}
}

func (m *PoolManager) GetStateByPoolAddresses(
	ctx context.Context,
	addresses, dex []string,
	stateRoot gethcommon.Hash,
) (*types.FindRouteState, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "poolManager.GetStateByPoolAddresses")
	defer span.End()

	pools, err := m.listPools(
		ctx,
		addresses,
		common.PoolFilterSources(dex),
		common.PoolFilterHasReserveOrAmplifiedTvl,
	)
	defer mempool.ReserveMany(pools)
	var (
		resultLimits = make(map[string]map[string]*big.Int)
		iPools       = m.poolFactory.NewPoolByAddress(ctx, pools, stateRoot)
	)
	resultLimits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
	resultLimits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)

	//given a clone of limit
	for _, pool := range iPools {
		dexLimit, avail := resultLimits[pool.GetType()]
		if !avail {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			dexLimit[k] = v
		}
	}
	if err != nil {
		return nil, err
	}

	return &types.FindRouteState{
		Pools:     iPools,
		SwapLimit: m.poolFactory.NewSwapLimit(resultLimits),
	}, nil
}

func (m *PoolManager) ApplyConfig(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
}

func (m *PoolManager) listPools(ctx context.Context, addresses []string, filters ...common.PoolFilter) ([]*entity.Pool, error) {
	filteredAddresses := m.filterBlacklistedAddresses(ctx, addresses)

	filteredAddresses = m.filterFaultyPools(ctx, filteredAddresses, m.config)

	pools, err := m.poolRepository.FindByAddresses(ctx, filteredAddresses)
	if err != nil {
		return nil, err
	}

	filteredPools := common.FilterPools(
		pools,
		filters...,
	)

	curveMetaBasePools, err := listCurveMetaBasePools(ctx, m.poolRepository, filteredPools)

	if err != nil {
		return nil, err
	}

	return append(filteredPools, curveMetaBasePools...), nil
}

// listCurveMetaBasePools collects base pools of curveMeta pools
// - collects already fetched curveBase and curvePainOracle pools
// - for each curveMeta pool
//   - decode its staticExtra to get its basePool address
//   - if it hasn't been fetched, fetch the pool data
func listCurveMetaBasePools(
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
		if pool.Type == pooltypes.PoolTypes.CurveBase {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == pooltypes.PoolTypes.CurveStablePlain {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == pooltypes.PoolTypes.CurvePlainOracle {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == pooltypes.PoolTypes.CurveAave {
			alreadyFetchedSet[pool.Address] = true
		}
	}

	for _, pool := range pools {
		if pool.Type != pooltypes.PoolTypes.CurveMeta {
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

func (m *PoolManager) filterBlacklistedAddresses(ctx context.Context, addresses []string) []string {
	filtered := make([]string, 0, len(addresses))

	for _, address := range addresses {
		if m.config.BlacklistedPoolSet[address] {
			continue
		}

		filtered = append(filtered, address)
	}

	// check again with Redis
	isInBlacklist, err := m.poolRepository.CheckPoolsInBlacklist(ctx, filtered)
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

func (m *PoolManager) filterFaultyPools(ctx context.Context, addresses []string, config Config) []string {
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

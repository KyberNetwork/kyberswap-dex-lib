package poolmanager

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
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

func (m *PoolManager) GetPoolByAddress(
	ctx context.Context,
	addresses, dex []string,
) (map[string]poolpkg.IPool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "poolManager.GetPoolByAddress")
	defer span.Finish()

	pools, err := m.listPools(
		ctx,
		addresses,
		common.PoolFilterSources(dex),
		common.PoolFilterHasReserveOrAmplifiedTvl,
	)
	defer mempool.ReserveMany(pools)
	if err != nil {
		return nil, err
	}

	return m.poolFactory.NewPoolByAddress(ctx, pools), nil
}

func (m *PoolManager) ApplyConfig(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
}

func (m *PoolManager) listPools(ctx context.Context, addresses []string, filters ...common.PoolFilter) ([]*entity.Pool, error) {
	filteredAddresses := m.filterBlacklistedAddresses(addresses)

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
		if pool.Type == constant.PoolTypes.CurveBase {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == constant.PoolTypes.CurvePlainOracle {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == constant.PoolTypes.CurveAave {
			alreadyFetchedSet[pool.Address] = true
		}
	}

	for _, pool := range pools {
		if pool.Type != constant.PoolTypes.CurveMeta {
			continue
		}

		var staticExtra struct {
			BasePool string `json:"basePool"`
		}

		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			logger.WithFields(logger.Fields{
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

func (m *PoolManager) filterBlacklistedAddresses(addresses []string) []string {
	filtered := make([]string, 0, len(addresses))

	for _, address := range addresses {
		if m.config.BlacklistedPoolSet[address] {
			continue
		}

		filtered = append(filtered, address)
	}

	return filtered
}

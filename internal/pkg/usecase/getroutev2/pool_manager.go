package getroutev2

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type poolManager struct {
	poolRepository IPoolRepository
	poolFactory    IPoolFactory

	config PoolManagerConfig
}

func NewPoolManager(
	poolRepository IPoolRepository,
	poolFactory IPoolFactory,
	config PoolManagerConfig,
) *poolManager {
	return &poolManager{
		poolRepository: poolRepository,
		poolFactory:    poolFactory,
		config:         config,
	}
}

func (m *poolManager) GetPoolByAddress(
	ctx context.Context,
	addresses []string,
	filters ...PoolFilter,
) (map[string]poolpkg.IPool, error) {
	pools, err := m.listPools(ctx, addresses, filters)
	if err != nil {
		return nil, err
	}

	return m.poolFactory.NewPoolByAddress(ctx, pools), nil
}

func (m *poolManager) listPools(ctx context.Context, addresses []string, filters []PoolFilter) ([]*entity.Pool, error) {
	filteredAddresses := m.filterBlacklistedAddresses(addresses)

	pools, err := m.poolRepository.FindByAddresses(ctx, filteredAddresses)
	if err != nil {
		return nil, err
	}

	filteredPools := filterPools(
		pools,
		filters...,
	)

	curveMetaBasePools, err := m.listCurveMetaBasePools(ctx, filteredPools)
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
func (m *poolManager) listCurveMetaBasePools(
	ctx context.Context,
	pools []*entity.Pool,
) ([]*entity.Pool, error) {
	var (
		// alreadyFetchedSet contains fetched pool ids
		alreadyFetchedSet = map[string]bool{}

		// poolAddresses contains pool addresses to fetch
		poolAddresses []string
	)

	for _, pool := range pools {
		if pool.Type == constant.PoolTypes.CurveBase {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == constant.PoolTypes.CurvePlainOracle {
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

		poolAddresses = append(poolAddresses, staticExtra.BasePool)
	}

	return m.poolRepository.FindByAddresses(ctx, poolAddresses)
}

func (m *poolManager) filterBlacklistedAddresses(addresses []string) []string {
	filtered := make([]string, 0, len(addresses))

	for _, address := range addresses {
		if _, contained := m.config.BlacklistedPoolSet[address]; !contained {
			filtered = append(filtered, address)
		}
	}

	return filtered
}

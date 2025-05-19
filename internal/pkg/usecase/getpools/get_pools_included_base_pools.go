package getpools

import (
	"context"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type GetPoolsIncludingBasePools struct {
	poolRepository IPoolRepository
}

type PoolFilter func(pool *entity.Pool) bool

type PoolAddressFilter func(address string, index int) bool

func NewGetPoolsIncludingBasePools(
	poolRepo IPoolRepository,
) *GetPoolsIncludingBasePools {
	return &GetPoolsIncludingBasePools{
		poolRepository: poolRepo,
	}
}

func NonFilter(pool *entity.Pool) bool {
	return true
}

func (u *GetPoolsIncludingBasePools) Handle(ctx context.Context, addresses []string, filter PoolFilter) ([]*entity.Pool, error) {
	if filter == nil {
		filter = NonFilter
	}
	poolEntities, err := u.poolRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	filteredPoolEntities := make([]*entity.Pool, 0, len(poolEntities))
	// listCurveMetaBasePools collects base pools of curveMeta pools
	// - collects already fetched curveBase and curvePainOracle pools
	// - for each curveMeta pool
	//   - decode its staticExtra to get its basePool address
	//   - if it hasn't been fetched, fetch the pool data
	alreadyFetchedSet := mapset.NewThreadUnsafeSet[string]()
	curveMetaBasePoolAddresses := mapset.NewThreadUnsafeSet[string]()
	for _, pool := range poolEntities {
		if !filter(pool) {
			mempool.EntityPool.Put(pool)
			continue
		}
		filteredPoolEntities = append(filteredPoolEntities, pool)

		if pool.Type == pooltypes.PoolTypes.CurveBase ||
			pool.Type == pooltypes.PoolTypes.CurveStablePlain ||
			pool.Type == pooltypes.PoolTypes.CurvePlainOracle ||
			pool.Type == pooltypes.PoolTypes.CurveAave {
			alreadyFetchedSet.Add(pool.Address)
		}

		if pool.Type == pooltypes.PoolTypes.CurveMeta ||
			pool.Type == pooltypes.PoolTypes.CurveStableMetaNg {
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

			if staticExtra.BasePool != valueobject.ZeroAddress {
				curveMetaBasePoolAddresses.Add(strings.ToLower(staticExtra.BasePool))
			}
		}
	}
	// only fetch pools which are not fetched
	curveMetaBasePoolAddresses = curveMetaBasePoolAddresses.Difference(alreadyFetchedSet)

	// fetch extra base pools if needed
	if curveMetaBasePoolAddresses.Cardinality() != 0 {
		curveMetaBasePools, err := u.poolRepository.FindByAddresses(ctx, curveMetaBasePoolAddresses.ToSlice())
		if err != nil {
			// need to log err here for debug
			// If we fetch curve meta base pools fail, then IPoolSimulator for meta curve will be failed and ignored later, so ignore errors here
			logger.WithFields(ctx, logger.Fields{
				"pool.Address": curveMetaBasePoolAddresses,
				"error":        err,
			}).Errorf("failed to fetch based pools in pool manager")
		} else {
			filteredPoolEntities = append(filteredPoolEntities, curveMetaBasePools...)
		}
	}

	return filteredPoolEntities, nil
}

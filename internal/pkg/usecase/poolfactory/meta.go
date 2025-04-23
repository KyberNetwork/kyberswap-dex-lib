package poolfactory

import (
	"context"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
)

type (
	poolEntityMatcher func(*entity.Pool) bool
)

func (m poolEntityMatcher) and(other poolEntityMatcher) poolEntityMatcher {
	return func(pool *entity.Pool) bool {
		return m(pool) && other(pool)
	}
}

func poolOfType(poolType string) poolEntityMatcher {
	return func(pool *entity.Pool) bool {
		return pool.Type == poolType
	}
}

func poolNotContainingStaticExtra(extraSubstring string) poolEntityMatcher {
	return func(pool *entity.Pool) bool {
		return !strings.Contains(pool.StaticExtra, extraSubstring)
	}
}

var (
	basePoolTypesSets = [][]poolEntityMatcher{ // ordered sets of base pools that must be created first
		{
			poolOfType(pooltypes.PoolTypes.CurveBase),
			poolOfType(pooltypes.PoolTypes.CurveStablePlain),
			poolOfType(pooltypes.PoolTypes.CurvePlainOracle),
			poolOfType(pooltypes.PoolTypes.CurveAave),
			poolOfType(pooltypes.PoolTypes.CurveStablePlain),
			poolOfType(pooltypes.PoolTypes.CurveStableNg),

			poolOfType(pooltypes.PoolTypes.BalancerV2ComposableStable),
			poolOfType(pooltypes.PoolTypes.BalancerV2Stable).
				and(poolNotContainingStaticExtra(`basePools`)),
			poolOfType(pooltypes.PoolTypes.BalancerV2Weighted).
				and(poolNotContainingStaticExtra(`basePools`)),
		},
	}
)

func (f *PoolFactory) CloneMetaPoolsWithBasePools(
	ctx context.Context,
	allPools map[string]poolpkg.IPoolSimulator,
	basePools map[string]poolpkg.IPoolSimulator,
) []poolpkg.IPoolSimulator {
	var cloned []poolpkg.IPoolSimulator

	for _, pool := range allPools {
		pool, ok := pool.(poolpkg.IMetaPoolSimulator)
		if !ok {
			continue
		}

		basePoolList := pool.GetBasePools()

		for i := range basePoolList {
			basePoolAddress := basePoolList[i].GetAddress()
			basePoolList[i], ok = basePools[basePoolAddress]
			if !ok {
				continue
			}
			newMetaPool := finderCommon.DefaultClonePool(ctx, pool).(poolpkg.IMetaPoolSimulator)
			newMetaPool.SetBasePool(basePoolList[i])
			cloned = append(cloned, newMetaPool)
		}
	}

	return cloned
}

func matchesAny(pool *entity.Pool, matchers []poolEntityMatcher) bool {
	for _, matcher := range matchers {
		if matcher(pool) {
			return true
		}
	}
	return false
}

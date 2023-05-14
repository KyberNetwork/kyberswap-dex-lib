package common

import (
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func FilterPools(pools []*entity.Pool, filters ...PoolFilter) []*entity.Pool {
	filteredPools := make([]*entity.Pool, 0, len(pools))

	for _, pool := range pools {
		valid := true

		for _, filter := range filters {
			if !filter(pool) {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		filteredPools = append(filteredPools, pool)
	}

	return filteredPools
}

type PoolFilter func(pool *entity.Pool) bool

func PoolFilterSources(sources []string) PoolFilter {
	sourceSet := make(map[string]struct{}, len(sources))

	for _, source := range sources {
		sourceSet[source] = struct{}{}
	}

	return func(pool *entity.Pool) bool {
		_, contained := sourceSet[pool.Exchange]

		return contained
	}
}

func PoolFilterHasReserveOrAmplifiedTvl(pool *entity.Pool) bool {
	return pool.HasReserves() || pool.HasAmplifiedTvl()
}

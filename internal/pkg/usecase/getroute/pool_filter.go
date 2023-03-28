package getroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func filterPools(pools []entity.Pool, filters ...PoolFilter) []entity.Pool {
	filteredPools := make([]entity.Pool, 0, len(pools))

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

type PoolFilter func(pool entity.Pool) bool

func PoolFilterSources(sources []string) PoolFilter {
	sourceSet := make(map[string]bool, len(sources))

	for _, source := range sources {
		sourceSet[source] = true
	}

	return func(pool entity.Pool) bool {
		return sourceSet[pool.Exchange]
	}
}

func PoolFilterHasReserveOrAmplifiedTvl(pool entity.Pool) bool {
	return pool.HasReserves() || pool.HasAmplifiedTvl()
}

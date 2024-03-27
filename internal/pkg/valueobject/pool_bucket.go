package valueobject

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/huandu/go-clone"
)

// PoolBucket contains data for finding route
// it is responsible for cloning pools
type PoolBucket struct {
	// PerRequestPoolsByAddress mapping from pool address to IPool
	PerRequestPoolsByAddress map[string]poolpkg.IPoolSimulator
	// ChangedPools Keep track of pools we updated balance
	ChangedPools map[string]poolpkg.IPoolSimulator
}

func NewPoolBucket(perRequestPoolsByAddress map[string]poolpkg.IPoolSimulator) *PoolBucket {
	return &PoolBucket{
		PerRequestPoolsByAddress: perRequestPoolsByAddress,
		ChangedPools:             nil,
	}
}

func (b *PoolBucket) RollBackPools(backUpPools []poolpkg.IPoolSimulator) {
	for i := 0; i < len(backUpPools); i++ {
		if backUpPools[i] == nil {
			continue
		}
		if _, avail := b.ChangedPools[backUpPools[i].GetAddress()]; avail {
			b.ChangedPools[backUpPools[i].GetAddress()] = backUpPools[i]
		}
	}
}
func (b *PoolBucket) ClearChangedPools() {
	b.ChangedPools = nil
}

// ClonePool clone the pool before updating, so that it doesn't modify the original data copied from route service
// do nothing if pool is already cloned, or if original data of that pool not found
// otherwise, clone pool from PerRequestPoolsByAddress to ChangedPools
func (b *PoolBucket) ClonePool(poolAddress string) poolpkg.IPoolSimulator {
	var (
		pool  poolpkg.IPoolSimulator
		avail bool
	)
	if b.ChangedPools != nil {
		// if pool is already cloned, do thing
		if pool, avail = b.ChangedPools[poolAddress]; avail {
			return pool
		}
	} else {
		b.ChangedPools = make(map[string]poolpkg.IPoolSimulator)
	}
	pool, avail = b.PerRequestPoolsByAddress[poolAddress]
	// if original data not found, do nothing
	if !avail {
		return nil
	}

	// clone the pool and add to ChangedPools
	v := clone.Slowly(pool)
	pool = v.(poolpkg.IPoolSimulator)
	b.ChangedPools[poolAddress] = pool

	// Note: When we need to clone a curve-meta pool, we should clone its base pool as well (as the code below)
	//We will omit that adhoc logic here for simplicity -> expect some miscalculation in algorithm

	//if pool.GetType() == pooltypes.PoolTypes.CurveMeta {
	//	curveMetaPool, ok := pool.(*curveMeta.Pool)
	//	if !ok {
	//		return pool
	//	}
	//	basePool := b.ClonePool(curveMetaPool.BasePool.GetInfo().Address)
	//	if basePool == nil {
	//		return pool
	//	}
	//	(*curveMetaPool).BasePool = basePool.(curveMeta.ICurveBasePool)
	//}
	return pool
}

// GetPool search for changed pool, then search for original pool
func (b *PoolBucket) GetPool(poolAddress string) (poolpkg.IPoolSimulator, bool) {
	var (
		pool  poolpkg.IPoolSimulator
		avail bool
	)
	if b.ChangedPools != nil {
		pool, avail = b.ChangedPools[poolAddress]
		if avail {
			return pool, avail
		}
	}
	pool, avail = b.PerRequestPoolsByAddress[poolAddress]
	return pool, avail
}

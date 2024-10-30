package poolmanager

import (
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type LockedState struct {
	poolByAddress map[string]poolpkg.IPoolSimulator
	limits        map[string]map[string]*big.Int
	lock          *sync.RWMutex
}

func NewLockedState() *LockedState {

	var limits = make(map[string]map[string]*big.Int)
	limits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.NativeV1] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.LimitOrder] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Bebop] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Dexalot] = make(map[string]*big.Int)

	return &LockedState{
		poolByAddress: make(map[string]poolpkg.IPoolSimulator),
		limits:        limits,
		lock:          &sync.RWMutex{},
	}
}

func (s *LockedState) update(poolByAddress map[string]poolpkg.IPoolSimulator) {
	s.lock.Lock()
	defer s.lock.Unlock()

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
			if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
				dexLimit[k] = v
			}
		}
	}
	s.poolByAddress = poolByAddress
	// Optimize graph traversal by using tokenToPoolAddress list
}

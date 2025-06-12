package poolmanager

import (
	"math/big"
	"sync"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

type LockedState struct {
	poolByAddress map[string]poolpkg.IPoolSimulator
	limits        map[string]map[string]*big.Int
	stateRoot     common.Hash
	*sync.RWMutex
}

func NewLockedState() *LockedState {
	var limits = make(map[string]map[string]*big.Int)
	for _, poolType := range constant.DexUseSwapLimit {
		limits[poolType] = make(map[string]*big.Int)
	}

	return &LockedState{
		poolByAddress: make(map[string]poolpkg.IPoolSimulator),
		limits:        limits,
		RWMutex:       &sync.RWMutex{},
	}
}

func (s *LockedState) update(poolByAddress map[string]poolpkg.IPoolSimulator, stateRoot common.Hash) {
	s.Lock()
	defer s.Unlock()
	s.poolByAddress = poolByAddress
	s.clearLimits()
	UpdateLimits(s.limits, poolByAddress)
	s.stateRoot = stateRoot
}

func (s *LockedState) clearLimits() {
	for _, dexLimit := range s.limits {
		for k := range dexLimit {
			delete(dexLimit, k)
		}
	}
}

func UpdateLimits(limits map[string]map[string]*big.Int, poolByAddress map[string]poolpkg.IPoolSimulator) {
	for _, pool := range poolByAddress {
		dexLimit, ok := limits[pool.GetType()]
		if !ok {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
				dexLimit[k] = v
			}
		}
	}
}

package pool

import (
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	FactoryParams struct {
		EntityPool  entity.Pool
		BasePoolMap map[string]IPoolSim
		ChainID     valueobject.ChainID
		EthClient   ethereum.ContractCaller
		Opts        FactoryOpts
	}
	FactoryOpts struct {
		StaleCheck bool // whether to reject stale pools or not - route finding sets this true while indexing false
	}
	// FactoryFn returns a pool that implements IPoolSimulator (legacy) or IPoolSim (uint256).
	// Use Factory for legacy and FactoryPoolSim for the uint256 interface.
	FactoryFn func(FactoryParams) (any, error)
)

var (
	factoryMap      = make(map[string]FactoryFn, 256) // map of pool types to factory functions
	CanCalcAmountIn = make(map[string]struct{}, 256)  // map of pool types that can calculate amount in. don't modify
	UseSwapLimit    = make(map[string]struct{}, 16)   // exchanges or pool types that use swap limit. don't modify
)

// RegisterFactory registers a factory function for a pool type with factoryParams. P must implement IPoolSimulator
// (legacy big.Int) or IPoolSim (uint256).
func RegisterFactory[P any](poolType string, factory func(FactoryParams) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool factory already registered")
	}
	var zero P
	if _, legacy := any(zero).(IPoolSimulator); !legacy {
		if _, native := any(zero).(IPoolSim); !native {
			panic(poolType + " pool factory must implement IPoolSimulator or IPoolSim")
		}
	}
	factoryMap[poolType] = func(factoryParams FactoryParams) (pool any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.Errorf("panic: %v", r)
			}
			err = errors.WithMessagef(err, "failed to init pool %s (%s/%s)",
				factoryParams.EntityPool.Address, factoryParams.EntityPool.Exchange, poolType)
		}()
		return factory(factoryParams)
	}

	if _, ok := any(zero).(IPoolExactOutSimulator); ok {
		CanCalcAmountIn[poolType] = struct{}{}
	}

	return true
}

func RegisterUseSwapLimit(exchange string) bool {
	UseSwapLimit[exchange] = struct{}{}
	return true
}

// UsesSwapLimit reports whether a pool draws on a shared inventory, i.e. its exchange or its
// pool type is registered via RegisterUseSwapLimit.
func UsesSwapLimit(exchange, poolType string) bool {
	if _, ok := UseSwapLimit[exchange]; ok {
		return true
	}
	_, ok := UseSwapLimit[poolType]
	return ok
}

// RegisterFactory0 registers a factory function for a pool type with no factoryParams.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory0[P any](poolType string, factory func(entity.Pool) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool)
	})
}

// RegisterFactory1 registers a factory function for a pool type with chainID.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory1[P any](poolType string,
	factory func(entity.Pool, valueobject.ChainID) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.ChainID)
	})
}

// RegisterFactory2 registers a factory function for a pool type with chainID and ethClient.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory2[P any](poolType string,
	factory func(entity.Pool, valueobject.ChainID, ethereum.ContractCaller) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.ChainID, factoryParams.EthClient)
	})
}

// RegisterFactoryMeta registers a factory function for a meta pool type with basePoolMap.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactoryMeta[P any](poolType string,
	factory func(entity.Pool, map[string]IPoolSim) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.BasePoolMap)
	})
}

// Factory returns the factory for a pool type as a legacy IPoolSimulator. It returns nil for an unknown type, and
// an error from the returned function for a uint256-only pool. Use FactoryPoolSim for uint256.
func Factory(poolType string) func(FactoryParams) (IPoolSimulator, error) {
	factory := factoryMap[poolType]
	if factory == nil {
		return nil
	}
	return func(params FactoryParams) (IPoolSimulator, error) {
		sim, err := factory(params)
		if err != nil {
			return nil, err
		}
		legacy, ok := sim.(IPoolSimulator)
		if !ok {
			return nil, errors.Errorf("pool type %s is uint256-only; use FactoryPoolSim", poolType)
		}
		return legacy, nil
	}
}

// QuoteTime is the snapshot time to price a quote at, or the zero time when there is none. Only
// indexing (staleCheck false) uses it: a pool whose DB row is older than the tracked state still
// quotes as of the snapshot. Route finding keeps the wall clock, so it gets the zero time and the
// simulator's own clock applies.
func QuoteTime(staleCheck bool, timestamp int64) time.Time {
	if !staleCheck && timestamp > 0 {
		return time.Unix(timestamp, 0)
	}
	return time.Time{}
}

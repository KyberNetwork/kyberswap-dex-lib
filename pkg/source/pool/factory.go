package pool

import (
	"github.com/ethereum/go-ethereum"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	FactoryParams struct {
		EntityPool  entity.Pool
		BasePoolMap map[string]IPoolSimulator
		ChainID     valueobject.ChainID
		EthClient   ethereum.ContractCaller
	}
	FactoryFn func(FactoryParams) (IPoolSimulator, error)
)

var (
	factoryMap      = make(map[string]FactoryFn, 256) // map of pool types to factory functions
	CanCalcAmountIn = make(map[string]struct{}, 256)  // map of pool types that can calculate amount in. don't modify
)

// RegisterFactory registers a factory function for a pool type with factoryParams
func RegisterFactory[P IPoolSimulator](poolType string, factory func(FactoryParams) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool factory already registered")
	}
	factoryMap[poolType] = func(factoryParams FactoryParams) (IPoolSimulator, error) {
		pool, err := factory(factoryParams)
		return pool, errors.WithMessagef(err, "failed to init pool %s (%s/%s)",
			factoryParams.EntityPool.Address, factoryParams.EntityPool.Exchange, poolType)
	}
	var p P
	if _, ok := any(p).(IPoolExactOutSimulator); ok {
		CanCalcAmountIn[poolType] = struct{}{}
	}
	return true
}

// RegisterFactory0 registers a factory function for a pool type with no factoryParams.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory0[P IPoolSimulator](poolType string, factory func(entity.Pool) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool)
	})
}

// RegisterFactory1 registers a factory function for a pool type with chainID.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory1[P IPoolSimulator](poolType string,
	factory func(entity.Pool, valueobject.ChainID) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.ChainID)
	})
}

// RegisterFactory2 registers a factory function for a pool type with chainID and ethClient.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactory2[P IPoolSimulator](poolType string,
	factory func(entity.Pool, valueobject.ChainID, ethereum.ContractCaller) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.ChainID, factoryParams.EthClient)
	})
}

// RegisterFactoryMeta registers a factory function for a meta pool type with basePoolMap.
// TODO: deprecate this in favor of RegisterFactory
func RegisterFactoryMeta[P IPoolSimulator](poolType string,
	factory func(entity.Pool, map[string]IPoolSimulator) (P, error)) bool {
	return RegisterFactory(poolType, func(factoryParams FactoryParams) (P, error) {
		return factory(factoryParams.EntityPool, factoryParams.BasePoolMap)
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

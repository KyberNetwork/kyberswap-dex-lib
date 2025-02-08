package pool

import (
	"github.com/ethereum/go-ethereum"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	FactoryParams struct {
		BasePoolMap map[string]IPoolSimulator
		ChainID     valueobject.ChainID
		EthClient   ethereum.ContractCaller
	}
	FactoryFn func(entity.Pool, FactoryParams) (IPoolSimulator, error)
)

var (
	factoryMap = make(map[string]FactoryFn, 256) // map of pool types to factory functions
)

// RegisterFactory registers a factory function for a pool type with entityParams
func RegisterFactory[P IPoolSimulator](poolType string, factory func(entity.Pool, FactoryParams) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " factory already registered")
	}
	factoryMap[poolType] = func(entityPool entity.Pool, entityParams FactoryParams) (IPoolSimulator, error) {
		pool, err := factory(entityPool, entityParams)
		return pool, errors.WithMessagef(err, "failed to init pool %s (%s/%s)",
			entityPool.Address, entityPool.Exchange, poolType)
	}
	return true
}

// RegisterFactory0 registers a factory function for a pool type with no entityParams
func RegisterFactory0[P IPoolSimulator](poolType string, factory func(entity.Pool) (P, error)) bool {
	return RegisterFactory(poolType, func(entityPool entity.Pool, entityParams FactoryParams) (IPoolSimulator, error) {
		return factory(entityPool)
	})
}

// RegisterFactory1 registers a factory function for a pool type with chainID
func RegisterFactory1[P IPoolSimulator](poolType string,
	factory func(entity.Pool, valueobject.ChainID) (P, error)) bool {
	return RegisterFactory(poolType, func(entityPool entity.Pool, entityParams FactoryParams) (IPoolSimulator, error) {
		return factory(entityPool, entityParams.ChainID)
	})
}

// RegisterFactory2 registers a factory function for a pool type with chainID and ethClient
func RegisterFactory2[P IPoolSimulator](poolType string,
	factory func(entity.Pool, valueobject.ChainID, ethereum.ContractCaller) (P, error)) bool {
	return RegisterFactory(poolType, func(entityPool entity.Pool, entityParams FactoryParams) (IPoolSimulator, error) {
		return factory(entityPool, entityParams.ChainID, entityParams.EthClient)
	})
}

// RegisterFactoryMeta registers a factory function for a meta pool type with basePoolMap
func RegisterFactoryMeta[P IPoolSimulator](poolType string,
	factory func(entity.Pool, map[string]IPoolSimulator) (P, error)) bool {
	return RegisterFactory(poolType, func(entityPool entity.Pool, entityParams FactoryParams) (IPoolSimulator, error) {
		return factory(entityPool, entityParams.BasePoolMap)
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

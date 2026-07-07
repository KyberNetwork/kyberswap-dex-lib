package poollist

import (
	"context"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	IPoolsLister    = pool.IPoolsListUpdater
	IPoolRepository interface {
		Get(ctx context.Context, address string) (entity.Pool, error)
	}
	PoolsListerParams[C any] struct {
		Cfg *C
		Dependencies
	}
	FactoryParams struct {
		Exchange string
		Properties
		Dependencies
	}
	Dependencies struct {
		EthrpcClient   *ethrpc.Client
		PoolRepository IPoolRepository
		GraphqlClient  *graphqlpkg.Client
	}
	Properties map[string]any
	FactoryFn  func(string, FactoryParams) (IPoolsLister, error)
)

var (
	factoryMap = make(map[string]FactoryFn, 256) // map of pool types to pool lister factory functions
)

// RegisterFactory registers a factory function for a pool lister with config and factoryParams
func RegisterFactory[C any, P IPoolsLister](poolType string, factory func(PoolsListerParams[C]) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool lister factory already registered")
	}

	factoryMap[poolType] = func(exchange string, factoryParams FactoryParams) (IPoolsLister, error) {
		var cfg C
		properties := factoryParams.Properties
		if properties == nil {
			properties = make(map[string]any, 1)
		}
		properties["DexID"] = exchange
		if err := pool.PropertiesToStruct(properties, &cfg); err != nil {
			return nil, err
		}
		return factory(PoolsListerParams[C]{
			Cfg:          &cfg,
			Dependencies: factoryParams.Dependencies,
		})
	}
	return true
}

// RegisterFactoryC registers a factory function for a pool lister with config
func RegisterFactoryC[C any, P IPoolsLister](poolType string, factory func(*C) P) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg), nil
	})
}

// RegisterFactoryCE registers a factory function for a pool lister with config and ethrpcClient
func RegisterFactoryCE[C any, P IPoolsLister](poolType string, factory func(*C, *ethrpc.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg, params.EthrpcClient), nil
	})
}

// RegisterFactoryCE1 registers a factory function for a pool lister with config and ethrpcClient
func RegisterFactoryCE1[C any, P IPoolsLister](poolType string, factory func(*C, *ethrpc.Client) (P, error)) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg, params.EthrpcClient)
	})
}

// RegisterFactoryCG registers a factory function for a pool lister with config and graphqlClient
func RegisterFactoryCG[C any, P IPoolsLister](poolType string, factory func(*C, *graphqlpkg.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg, params.GraphqlClient), nil
	})
}

// RegisterFactoryCEG registers a factory function for a pool lister with config, ethrpcClient and graphqlClient
func RegisterFactoryCEG[C any, P IPoolsLister](poolType string, factory func(*C, *ethrpc.Client, *graphqlpkg.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg, params.EthrpcClient, params.GraphqlClient), nil
	})
}

// RegisterFactoryE registers a factory function for a pool lister with ethrpcClient
func RegisterFactoryE[P IPoolsLister](poolType string, factory func(*ethrpc.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[struct{}]) (IPoolsLister, error) {
		return factory(params.EthrpcClient), nil
	})
}

// RegisterFactoryCPG registers a factory function for a pool lister with poolRepository and graphqlClient
func RegisterFactoryCPG[C any, P IPoolsLister](poolType string,
	factory func(*C, IPoolRepository, *graphqlpkg.Client) (P, error)) bool {
	return RegisterFactory(poolType, func(params PoolsListerParams[C]) (IPoolsLister, error) {
		return factory(params.Cfg, params.PoolRepository, params.GraphqlClient)
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

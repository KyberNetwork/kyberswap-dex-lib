package pooltrack

import (
	"context"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	IPoolsTracker   = pool.IPoolTracker
	IPoolRepository interface {
		Get(ctx context.Context, address string) (entity.Pool, error)
	}
	PoolsTrackerParams[C any] struct {
		Cfg *C
		Dependencies
	}
	FactoryParams struct {
		Exchange string
		Properties
		Dependencies
	}
	Dependencies struct {
		EthrpcClient  *ethrpc.Client
		GraphqlClient *graphqlpkg.Client
	}
	Properties map[string]any
	FactoryFn  func(string, FactoryParams) (IPoolsTracker, error)
)

var (
	factoryMap = make(map[string]FactoryFn, 256) // map of pool types to pool tracker factory functions
)

// RegisterFactory registers a factory function for a pool tracker with config and factoryParams
func RegisterFactory[C any, P IPoolsTracker](poolType string, factory func(PoolsTrackerParams[C]) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool tracker factory already registered")
	}

	factoryMap[poolType] = func(exchange string, factoryParams FactoryParams) (IPoolsTracker, error) {
		var cfg C
		properties := factoryParams.Properties
		if properties == nil {
			properties = make(map[string]any, 1)
		}
		properties["DexID"] = exchange
		if err := pool.PropertiesToStruct(properties, &cfg); err != nil {
			return nil, err
		}
		return factory(PoolsTrackerParams[C]{
			Cfg:          &cfg,
			Dependencies: factoryParams.Dependencies,
		})
	}
	return true
}

// RegisterFactory0 registers a factory function for a pool tracker with no argument
func RegisterFactory0[P IPoolsTracker](poolType string, factory func() (P, error)) bool {
	return RegisterFactory(poolType, func(PoolsTrackerParams[struct{}]) (IPoolsTracker, error) {
		return factory()
	})
}

// RegisterFactoryC registers a factory function for a pool tracker with config
func RegisterFactoryC[C any, P IPoolsTracker](poolType string, factory func(*C) P) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg), nil
	})
}

// RegisterFactoryCE registers a factory function for a pool tracker with config and ethrpcClient
func RegisterFactoryCE[C any, P IPoolsTracker](poolType string, factory func(*C, *ethrpc.Client) (P, error)) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient)
	})
}

// RegisterFactoryCE0 registers a factory function for a pool tracker with config and ethrpcClient
func RegisterFactoryCE0[C any, P IPoolsTracker](poolType string, factory func(*C, *ethrpc.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient), nil
	})
}

// RegisterFactoryCEG registers a factory function for a pool tracker with config, ethrpcClient and graphqlClient
func RegisterFactoryCEG[C any, P IPoolsTracker](poolType string,
	factory func(*C, *ethrpc.Client, *graphqlpkg.Client) (P, error)) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient, params.GraphqlClient)
	})
}

// RegisterFactoryCEG0 registers a factory function for a pool tracker with config, ethrpcClient and graphqlClient
func RegisterFactoryCEG0[C any, P IPoolsTracker](poolType string,
	factory func(*C, *ethrpc.Client, *graphqlpkg.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient, params.GraphqlClient), nil
	})
}

// RegisterFactoryE registers a factory function for a pool tracker with ethrpcClient
func RegisterFactoryE[P IPoolsTracker](poolType string, factory func(*ethrpc.Client) (P, error)) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[struct{}]) (IPoolsTracker, error) {
		return factory(params.EthrpcClient)
	})
}

// RegisterFactoryE0 registers a factory function for a pool tracker with ethrpcClient
func RegisterFactoryE0[P IPoolsTracker](poolType string, factory func(*ethrpc.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsTrackerParams[struct{}]) (IPoolsTracker, error) {
		return factory(params.EthrpcClient), nil
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

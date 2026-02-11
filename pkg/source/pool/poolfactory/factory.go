package poolfactory

import (
	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type (
	IPoolFactoryDecoder       = pool.IPoolFactoryDecoder
	PoolsDecoderParams[C any] struct {
		Cfg *C
		Dependencies
	}
	Dependencies struct {
		EthrpcClient *ethrpc.Client
	}
	FactoryParams struct {
		Properties
		Dependencies
	}
	Properties map[string]any
	FactoryFn  func(string, FactoryParams) (IPoolFactoryDecoder, error)
)

var (
	factoryMap = make(map[string]FactoryFn, 256) // map of pool types to pool factory decoder factory functions
)

func RegisterFactory[C any, P IPoolFactoryDecoder](poolType string, factory func(PoolsDecoderParams[C]) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool factory decoder factory already registered")
	}
	factoryMap[poolType] = func(exchange string, factoryParams FactoryParams) (IPoolFactoryDecoder, error) {
		var cfg C
		properties := factoryParams.Properties
		if properties == nil {
			properties = make(map[string]any, 1)
		}
		properties["DexID"] = exchange
		if err := pool.PropertiesToStruct(properties, &cfg); err != nil {
			return nil, err
		}
		return factory(PoolsDecoderParams[C]{
			Cfg:          &cfg,
			Dependencies: factoryParams.Dependencies,
		})
	}
	return true
}

func RegisterFactoryC[C any, P IPoolFactoryDecoder](poolType string, factory func(*C) P) bool {
	return RegisterFactory(poolType, func(params PoolsDecoderParams[C]) (IPoolFactoryDecoder, error) {
		return factory(params.Cfg), nil
	})
}

func RegisterFactoryCE[C any, P IPoolFactoryDecoder](poolType string, factory func(*C, *ethrpc.Client) P) bool {
	return RegisterFactory(poolType, func(params PoolsDecoderParams[C]) (IPoolFactoryDecoder, error) {
		return factory(params.Cfg, params.EthrpcClient), nil
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

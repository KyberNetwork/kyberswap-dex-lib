package decode

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

type (
	IPoolDecoder              = pool.IPoolDecoder
	PoolsDecoderParams[C any] struct {
		Cfg *C
	}
	FactoryParams struct {
		Properties
	}
	Properties map[string]any
	FactoryFn  func(string, FactoryParams) (IPoolDecoder, error)
)

var (
	factoryMap = make(map[string]FactoryFn, 256) // map of pool types to pool lister factory functions
)

func RegisterFactory[C any, P IPoolDecoder](poolType string, factory func(PoolsDecoderParams[C]) (P, error)) bool {
	if factoryMap[poolType] != nil {
		panic(poolType + " pool lister factory already registered")
	}
	factoryMap[poolType] = func(exchange string, factoryParams FactoryParams) (IPoolDecoder, error) {
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
			Cfg: &cfg,
		})
	}
	return true
}

func RegisterFactoryC[C any, P IPoolDecoder](poolType string, factory func(*C) P) bool {
	return RegisterFactory(poolType, func(params PoolsDecoderParams[C]) (IPoolDecoder, error) {
		return factory(params.Cfg), nil
	})
}

// Factory returns the factory function for a pool type
func Factory(poolType string) FactoryFn {
	return factoryMap[poolType]
}

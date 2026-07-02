package pooltrack

import (
	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	backupFactoryMap = make(map[string]FactoryFn, 256) // map of pool types to pool tracker factory functions
)

func RegisterBackupFactoryCE0[C any, P IPoolsTracker](poolType string, factory func(*C, *ethrpc.Client) P) bool {
	return RegisterBackupFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient), nil
	})
}

func RegisterBackupFactoryCE[C any, P IPoolsTracker](poolType string, factory func(*C, *ethrpc.Client) (P, error)) bool {
	return RegisterBackupFactory(poolType, func(params PoolsTrackerParams[C]) (IPoolsTracker, error) {
		return factory(params.Cfg, params.EthrpcClient)
	})
}

// RegisterFactory registers a factory function for a pool tracker with config and factoryParams
func RegisterBackupFactory[C any, P IPoolsTracker](poolType string, factory func(PoolsTrackerParams[C]) (P, error)) bool {
	if backupFactoryMap[poolType] != nil {
		panic(poolType + " pool tracker backup factory already registered")
	}

	backupFactoryMap[poolType] = func(exchange string, factoryParams FactoryParams) (IPoolsTracker, error) {
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

// Factory returns the factory function for a pool type
func BackupFactory(poolType string) FactoryFn {
	return backupFactoryMap[poolType]
}

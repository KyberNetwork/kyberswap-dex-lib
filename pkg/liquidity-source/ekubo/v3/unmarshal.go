package ekubov3

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

func unmarshalExtra[T any](extraBytes []byte) (*T, error) {
	var state T
	err := json.Unmarshal(extraBytes, &state)

	return &state, err
}

func unmarshalPool(extraBytes []byte, staticExtra *StaticExtra) (Pool, error) {
	var pool Pool

	switch poolTypeConfig := staticExtra.PoolKey.Config.TypeConfig.(type) {
	case pools.FullRangePoolTypeConfig:
		poolKey := staticExtra.PoolKey.ToFullRange()

		switch staticExtra.ExtensionType {
		case ExtensionTypeBase:
			fullRangeState, err := unmarshalExtra[pools.FullRangePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing full range pool state: %w", err)
			}

			pool = pools.NewFullRangePool(
				poolKey,
				fullRangeState,
			)
		case ExtensionTypeOracle:
			oracleState, err := unmarshalExtra[pools.OraclePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing oracle pool state: %w", err)
			}

			pool = pools.NewOraclePool(
				poolKey,
				oracleState,
			)
		case ExtensionTypeTwamm:
			twammState, err := unmarshalExtra[pools.TwammPoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing oracle pool state: %w", err)
			}

			pool = pools.NewTwammPool(
				poolKey,
				twammState,
			)
		default:
			return nil, fmt.Errorf("unknown extension type %v for base pool", staticExtra.ExtensionType)
		}
	case pools.StableswapPoolTypeConfig:
		if staticExtra.ExtensionType != ExtensionTypeBase {
			return nil, fmt.Errorf("unknown extension type %v for stableswap pool", staticExtra.ExtensionType)
		}

		stableswapState, err := unmarshalExtra[pools.StableswapPoolState](extraBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing stableswap pool state: %w", err)
		}

		pool = pools.NewStableswapPool(
			staticExtra.PoolKey.ToStableswap(poolTypeConfig),
			stableswapState,
		)
	case pools.ConcentratedPoolTypeConfig:
		baseState, err := unmarshalExtra[pools.BasePoolState](extraBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing base pool state: %w", err)
		}

		poolKey := staticExtra.PoolKey.ToConcentrated(poolTypeConfig)

		switch staticExtra.ExtensionType {
		case ExtensionTypeBase:
			pool = pools.NewBasePool(
				poolKey,
				baseState,
			)
		case ExtensionTypeMevCapture:
			pool = pools.NewMevCapturePool(
				poolKey,
				baseState,
			)
		default:
			return nil, fmt.Errorf("unknown extension type %v for concentrated pool", staticExtra.ExtensionType)
		}
	default:
		return nil, fmt.Errorf("unknown pool type config %v", staticExtra.PoolKey.Config.TypeConfig)
	}

	return pool, nil
}

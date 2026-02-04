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

func unmarshalPool(extraBytes []byte, staticExtra *StaticExtra) (pools.Pool, error) {
	var pool pools.Pool

	switch config := staticExtra.PoolKey.Config.TypeConfig.(type) {
	case pools.FullRangePoolTypeConfig:
		poolKey := staticExtra.PoolKey.ToFullRange(config)

		switch staticExtra.ExtensionType {
		case ExtensionTypeNoSwapCallPoints:
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
				return nil, fmt.Errorf("parsing TWAMM pool state: %w", err)
			}

			pool = pools.NewTwammPool(
				poolKey,
				twammState,
			)
		default:
			return nil, fmt.Errorf("unknown extension type %v for base pool", staticExtra.ExtensionType)
		}
	case pools.StableswapPoolTypeConfig:
		if staticExtra.ExtensionType != ExtensionTypeNoSwapCallPoints {
			return nil, fmt.Errorf("unknown extension type %v for stableswap pool", staticExtra.ExtensionType)
		}

		stableswapState, err := unmarshalExtra[pools.StableswapPoolState](extraBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing stableswap pool state: %w", err)
		}

		pool = pools.NewStableswapPool(
			staticExtra.PoolKey.ToStableswap(config),
			stableswapState,
		)
	case pools.ConcentratedPoolTypeConfig:
		poolKey := staticExtra.PoolKey.ToConcentrated(config)

		switch staticExtra.ExtensionType {
		case ExtensionTypeNoSwapCallPoints:
			state, err := unmarshalExtra[pools.BasePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing base pool state: %w", err)
			}

			pool = pools.NewBasePool(
				poolKey,
				state,
			)
		case ExtensionTypeMevCapture:
			state, err := unmarshalExtra[pools.MevCapturePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing MEVCapture pool state: %w", err)
			}

			pool = pools.NewMevCapturePool(
				poolKey,
				state,
			)
		case ExtensionTypeBoostedFeesConcentrated:
			state, err := unmarshalExtra[pools.BoostedFeesPoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing BoostedFees pool state: %w", err)
			}

			pool = pools.NewBoostedFeesPool(poolKey, state)
		default:
			return nil, fmt.Errorf("unknown extension type %v for concentrated pool", staticExtra.ExtensionType)
		}
	default:
		return nil, fmt.Errorf("unknown pool type config %T", staticExtra.PoolKey.Config.TypeConfig)
	}

	return pool, nil
}

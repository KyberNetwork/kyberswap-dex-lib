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
		case ExtensionTypeVe33:
			state, err := unmarshalExtra[pools.Ve33PoolState[*pools.FullRangePoolState]](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing Ve33 full range pool state: %w", err)
			}

			pool = pools.NewVe33Pool(
				pools.NewFullRangePool(poolKey, state.UnderlyingPoolState),
				state.SwapFee,
			)
		default:
			return nil, fmt.Errorf("unknown extension type %v for full range pool", staticExtra.ExtensionType)
		}
	case pools.StableswapPoolTypeConfig:
		poolKey := staticExtra.PoolKey.ToStableswap(config)
		switch staticExtra.ExtensionType {
		case ExtensionTypeNoSwapCallPoints:
			stableswapState, err := unmarshalExtra[pools.StableswapPoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing stableswap pool state: %w", err)
			}
			pool = pools.NewStableswapPool(poolKey, stableswapState)
		case ExtensionTypeVe33:
			state, err := unmarshalExtra[pools.Ve33PoolState[*pools.StableswapPoolState]](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing Ve33 stableswap pool state: %w", err)
			}
			pool = pools.NewVe33Pool(pools.NewStableswapPool(poolKey, state.UnderlyingPoolState), state.SwapFee)
		default:
			return nil, fmt.Errorf("unknown extension type %v for stableswap pool", staticExtra.ExtensionType)
		}
	case pools.ConcentratedPoolTypeConfig:
		poolKey := staticExtra.PoolKey.ToConcentrated(config)

		switch staticExtra.ExtensionType {
		case ExtensionTypeNoSwapCallPoints:
			state, err := unmarshalExtra[pools.ConcentratedPoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing concentrated pool state: %w", err)
			}

			pool = pools.NewConcentratedPool(
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
		case ExtensionTypeVe33:
			state, err := unmarshalExtra[pools.Ve33PoolState[*pools.ConcentratedPoolState]](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing Ve33 concentrated pool state: %w", err)
			}

			pool = pools.NewVe33Pool(
				pools.NewConcentratedPool(poolKey, state.UnderlyingPoolState),
				state.SwapFee,
			)
		default:
			return nil, fmt.Errorf("unknown extension type %v for concentrated pool", staticExtra.ExtensionType)
		}
	default:
		return nil, fmt.Errorf("unknown pool type config %T", staticExtra.PoolKey.Config.TypeConfig)
	}

	return pool, nil
}

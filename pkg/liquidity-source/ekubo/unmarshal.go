package ekubo

import (
	"encoding/json"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
)

func unmarshalExtra[T any](extraBytes []byte) (*T, error) {
	var state T
	err := json.Unmarshal(extraBytes, &state)

	return &state, err
}

func unmarshalPool(extraBytes []byte, staticExtra *StaticExtra) (Pool, error) {
	var pool Pool
	switch staticExtra.ExtensionType {
	case ExtensionTypeBase:
		if staticExtra.PoolKey.Config.TickSpacing == 0 {
			fullRangeState, err := unmarshalExtra[pools.FullRangePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing full range pool state: %w", err)
			}

			pool = pools.NewFullRangePool(
				staticExtra.PoolKey,
				fullRangeState,
			)
		} else {
			baseState, err := unmarshalExtra[pools.BasePoolState](extraBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing base pool state: %w", err)
			}

			pool = pools.NewBasePool(
				staticExtra.PoolKey,
				baseState,
			)
		}
	case ExtensionTypeOracle:
		oracleState, err := unmarshalExtra[pools.OraclePoolState](extraBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing oracle pool state: %w", err)
		}

		pool = pools.NewOraclePool(
			staticExtra.PoolKey,
			oracleState,
		)
	case ExtensionTypeTwamm:
		twammState, err := unmarshalExtra[pools.TwammPoolState](extraBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing oracle pool state: %w", err)
		}

		pool = pools.NewTwammPool(
			staticExtra.PoolKey,
			twammState,
		)
	default:
		return nil, fmt.Errorf("unknown pool extension %v, %v", staticExtra.ExtensionType, staticExtra.PoolKey.Config.Extension)
	}

	return pool, nil
}

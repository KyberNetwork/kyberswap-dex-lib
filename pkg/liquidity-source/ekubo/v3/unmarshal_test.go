package ekubov3

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

func TestUnmarshalVe33Pools(t *testing.T) {
	t.Parallel()

	const swapFee = uint64(123)
	ve33 := common.HexToAddress("0xd100000000000000000000000000000000000000")

	tests := []struct {
		name       string
		typeConfig pools.PoolTypeConfig
		state      any
	}{
		{
			name:       "full range",
			typeConfig: pools.NewFullRangePoolTypeConfig(),
			state: pools.NewFullRangePoolState(
				pools.NewFullRangePoolSwapState(new(uint256.Int).Lsh(uint256.NewInt(1), 128)),
				uint256.NewInt(1_000_000),
			),
		},
		{
			name:       "stableswap",
			typeConfig: pools.NewStableswapPoolTypeConfig(0, 1),
			state: pools.NewStableswapPoolState(
				pools.NewStableswapPoolSwapState(new(uint256.Int).Lsh(uint256.NewInt(1), 128)),
				uint256.NewInt(1_000_000),
			),
		},
		{
			name:       "concentrated",
			typeConfig: pools.NewConcentratedPoolTypeConfig(1),
			state: pools.NewConcentratedPoolState(
				pools.NewConcentratedPoolSwapState(
					new(uint256.Int).Lsh(uint256.NewInt(1), 128),
					uint256.NewInt(1_000_000),
					0,
				),
				[]pools.Tick{},
				[2]int32{-10, 10},
				0,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
				common.HexToAddress("0x1"),
				common.HexToAddress("0x2"),
				pools.NewPoolConfig(ve33, 0, tt.typeConfig),
			)}
			extra, err := json.Marshal(pools.NewVe33PoolState(tt.state, swapFee))
			require.NoError(t, err)

			pool, err := unmarshalPool(extra, &StaticExtra{
				ExtensionType: ExtensionTypeVe33,
				PoolKey:       key,
			})
			require.NoError(t, err)

			roundTrip, err := json.Marshal(pool.GetState())
			require.NoError(t, err)
			require.JSONEq(t, string(extra), string(roundTrip))
		})
	}
}

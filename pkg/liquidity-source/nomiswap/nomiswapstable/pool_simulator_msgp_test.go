package nomiswapstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpMarshalUnmarshal(t *testing.T) {
	poolEntities := []entity.Pool{
		{
			Address:  "0x1e40450F8E21BB68490D7D91Ab422888Fb3D60f1",
			Exchange: "nomiswap",
			Type:     "nomiswap-stable",
			Reserves: []string{
				"53332989360391363843011",
				"74994257625190868514451",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x55d398326f99059fF775485246999027B3197955",
					Swappable: true,
				},
				{
					Address:   "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee\":6,\"token0PrecisionMultiplier\":1,\"token1PrecisionMultiplier\":1,\"a\":200000}",
		},
		{
			Address:  "0x1e40450F8E21BB68490D7D91Ab422888Fb3D60f1",
			Exchange: "nomiswap",
			Type:     "nomiswap-stable",
			Reserves: []string{
				"53332989360391363843011",
				"74994257625190868514451",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x55d398326f99059fF775485246999027B3197955",
					Swappable: true,
				},
				{
					Address:   "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee\":6,\"token0PrecisionMultiplier\":1,\"token1PrecisionMultiplier\":1,\"a\":200000}",
		},
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(poolEntity)
		require.NoError(t, err)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}

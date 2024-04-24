package syncswapstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	poolEntities := []entity.Pool{
		{
			Address:  "0x92eae0b3a75f3ef6c50369ce8ca96b285d2139b8",
			Exchange: "syncswap",
			Type:     "syncswap-stable",
			Reserves: []string{
				"276926762767",
				"284081796016",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
					Swappable: true,
				},
				{
					Address:   "0xfc7e56298657b002b3e656400e746b7212912757",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee0To1\":40,\"swapFee1To0\":40,\"token0PrecisionMultiplier\":1000000000000,\"token1PrecisionMultiplier\":1000000000000}",
		},
		{
			Address:  "0x92eae0b3a75f3ef6c50369ce8ca96b285d2139b8",
			Exchange: "syncswap",
			Type:     "syncswap-stable",
			Reserves: []string{
				"276838614939",
				"284170002373",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
					Swappable: true,
				},
				{
					Address:   "0xfc7e56298657b002b3e656400e746b7212912757",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee0To1\":40,\"swapFee1To0\":40,\"token0PrecisionMultiplier\":1000000000000,\"token1PrecisionMultiplier\":1000000000000}",
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

package velodrome

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			SwapFee:     0.0005, // from factory https://optimistic.etherscan.io/address/0x25cbddb98b35ab1ff77413456b31ec81a6b6b746#readContract
			Reserves:    entity.PoolReserves{"2082415614000308399878", "3631620514949"},
			Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}},
			StaticExtra: "{\"stable\": false}",
		})
		require.NoError(t, err)
		pools = append(pools, p)
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

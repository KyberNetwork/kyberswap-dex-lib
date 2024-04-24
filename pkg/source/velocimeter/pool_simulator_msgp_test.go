package velocimeter

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmar(t *testing.T) {
	var pools []*Pool
	{
		p, err := NewPool(entity.Pool{
			Exchange:    "",
			Type:        "",
			SwapFee:     0.003, // from factory getFee https://ftmscan.com/address/0x472f3c3c9608fe0ae8d702f3f8a2d12c410c881a#readContract#F6
			Reserves:    entity.PoolReserves{"257894248517799332584152", "629103671583531892529021"},
			Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
			StaticExtra: "{\"stable\": false}",
		})
		require.NoError(t, err)
		pools = append(pools, p)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(Pool)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(Pool{})...))
	}
}

package iziswap

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
			Address:  "0xee45cffbfafe97691b8ef068c8d55163086a3431",
			Exchange: "iziswap",
			Type:     "iziswap",
			SwapFee:  400,
			Reserves: entity.PoolReserves{"1167087113545385273", "18037620383221447465"},
			Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
			Extra:    "{\"CurrentPoint\":28912,\"PointDelta\":8,\"LeftMostPt\":-800000,\"RightMostPt\":800000,\"Fee\":400,\"Liquidity\":23123688144702854,\"LiquidityX\":8210612878032008,\"Liquidities\":[{\"LiqudityDelta\":23123688144702854,\"Point\":28728},{\"LiqudityDelta\":-23123688144702854,\"Point\":29128}],\"LimitOrders\":[]}",
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

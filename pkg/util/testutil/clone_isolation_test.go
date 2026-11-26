package testutil_test

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	beetsss "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/beets-ss"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	tokenA = "0x00000000000000000000000000000000000000a1"
	tokenB = "0x00000000000000000000000000000000000000b2"
)

// newPool is a beets-ss pool, whose GetReserves follows UpdateBalance (uniswap-v2's keeps its own state).
func newPool(t *testing.T) pool.IPoolSimulator {
	t.Helper()
	extra, err := json.Marshal(beetsss.Extra{TotalSupply: bignumber.NewBig10("1000000000000000000000"),
		TotalAssets: bignumber.NewBig10("1050000000000000000000")})
	require.NoError(t, err)
	p, err := beetsss.NewPoolSimulator(entity.Pool{Address: "0xbeets", Exchange: "beets-ss", Type: "beets-ss",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1050000000000000000000"},
		Tokens: []*entity.PoolToken{{Address: tokenA, Decimals: 18, Swappable: true},
			{Address: tokenB, Decimals: 18, Swappable: true}},
		Extra: string(extra)})
	require.NoError(t, err)
	return p
}

func swapAB() pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenA, Amount: bignumber.TenPowInt(18)},
		TokenOut: tokenB}
}

// SwapOp applies each swap in turn: two swaps of A move the reserves twice as far as one.
func TestSwapOp(t *testing.T) {
	one, two := newPool(t), newPool(t)
	testutil.SwapOp(swapAB())(t, one)
	testutil.SwapOp(swapAB(), swapAB())(t, two)

	require.Equal(t, "1001000000000000000000", one.GetReserves()[0].String())
	require.Equal(t, "1002000000000000000000", two.GetReserves()[0].String())
	require.True(t, two.GetReserves()[1].Cmp(one.GetReserves()[1]) < 0, "the second swap must take more of B")

	none := newPool(t)
	testutil.SwapOp()(t, none)
	require.Equal(t, "1000000000000000000000", none.GetReserves()[0].String(), "no swaps leave the pool")
}

// A simulator whose clones are isolated passes, with and without a watched simulator.
func TestCloneIsolationPassesAnIsolatedSimulator(t *testing.T) {
	ops := map[string]testutil.CloneOp{"swap": testutil.SwapOp(swapAB())}
	testutil.TestCloneIsolation(t, newPool(t), ops)
	testutil.TestCloneIsolation(t, newPool(t), ops, newPool(t))
}

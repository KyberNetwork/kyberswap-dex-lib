package ramsesv2

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
		token0 := "0x912ce59144191c1204e64559fe8253a0e49e6548"
		token1 := "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"
		p, err := NewPoolSimulator(
			entity.Pool{
				Exchange: "ramses-v2",
				Type:     "ramses-v2",
				SwapFee:  500,
				Reserves: entity.PoolReserves{"269329183753846211200", "526169379"},
				Tokens:   []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 6}},
				Extra:    "{\"liquidity\":4360306776077439,\"sqrtPriceX96\":85811322860530180084948,\"feeTier\":500,\"tickSpacing\":10,\"tick\":-274728,\"ticks\":[{\"index\":-283380,\"liquidityGross\":17166285019404,\"liquidityNet\":17166285019404},{\"index\":-279780,\"liquidityGross\":977381896105089,\"liquidityNet\":977381896105089},{\"index\":-278630,\"liquidityGross\":157248791282830,\"liquidityNet\":157248791282830},{\"index\":-278550,\"liquidityGross\":7351763429974,\"liquidityNet\":7351763429974},{\"index\":-276800,\"liquidityGross\":380989062434636,\"liquidityNet\":380989062434636},{\"index\":-276680,\"liquidityGross\":1196219220219038,\"liquidityNet\":1196219220219038},{\"index\":-276330,\"liquidityGross\":7351763429974,\"liquidityNet\":-7351763429974},{\"index\":-276170,\"liquidityGross\":294632869974088,\"liquidityNet\":294632869974088},{\"index\":-276070,\"liquidityGross\":826497613613152,\"liquidityNet\":826497613613152},{\"index\":-275100,\"liquidityGross\":510171037429202,\"liquidityNet\":510171037429202},{\"index\":-274550,\"liquidityGross\":157248791282830,\"liquidityNet\":-157248791282830},{\"index\":-274500,\"liquidityGross\":510171037429202,\"liquidityNet\":-510171037429202},{\"index\":-274170,\"liquidityGross\":1196219220219038,\"liquidityNet\":-1196219220219038},{\"index\":-274030,\"liquidityGross\":294632869974088,\"liquidityNet\":-294632869974088},{\"index\":-273320,\"liquidityGross\":826497613613152,\"liquidityNet\":-826497613613152},{\"index\":-272280,\"liquidityGross\":380989062434636,\"liquidityNet\":-380989062434636},{\"index\":-271750,\"liquidityGross\":977381896105089,\"liquidityNet\":-977381896105089},{\"index\":-269510,\"liquidityGross\":17166285019404,\"liquidityNet\":-17166285019404}]}",
			}, 1)
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

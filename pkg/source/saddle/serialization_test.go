package saddle_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
)

// A decoded simulator, after a swap and two msgpack round trips, quotes swaps, deposits and
// withdrawals like the original.
func TestMsgpackRoundTrip(t *testing.T) {
	p := entity.Pool{
		Address: "0xa6018520eaacc06c30ff2e1b3ee2c7c22e64196a",
		Type:    saddle.DexTypeSaddle,
		Reserves: entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907",
			"553429429583268691085"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"initialA":"48000","futureA":"92000","initialATime":1652287436,"futureATime":1653655053,"swapFee":"4000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LP","precisionMultipliers":["1","1","1"]}`,
	}
	sim, err := saddle.NewPoolSimulator(p)
	require.NoError(t, err)
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(1e18)}, TokenOut: "B"}
	res, err := sim.CalcAmountOut(in)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee})

	sims := map[string]pool.IPoolSimulator{p.Address: sim}
	for range 2 {
		encoded, err := msgpack.EncodePoolSimulatorsMap(sims)
		require.NoError(t, err)
		sims, err = msgpack.DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
	}
	for _, q := range [][2]string{{"A", "B"}, {"C", "LP"}, {"LP", "B"}} {
		in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: q[0], Amount: big.NewInt(1e17)}, TokenOut: q[1]}
		want, err := sim.CalcAmountOut(in)
		require.NoError(t, err)
		got, err := sims[p.Address].CalcAmountOut(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	require.Equal(t, sim.Info.Reserves, sims[p.Address].GetReserves())
}

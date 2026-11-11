package velodromev1_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The router receives simulators through pkg/msgpack (unexported fields), so the pair balances must survive the trip:
// a decoded simulator has to quote, reject and update like the original, with and without balances.
func TestMsgpackRoundTrip(t *testing.T) {
	const a, b = "0x00000000000000000000000000000000000000a1", "0x00000000000000000000000000000000000000b2"
	newSim := func(t *testing.T, balance0, balance1 *big.Int) *velodromev1.PoolSimulator {
		extra, err := json.Marshal(velodromev1.PoolExtra{Fee: 5, Balance0: balance0, Balance1: balance1})
		require.NoError(t, err)
		sim, err := velodromev1.NewPoolSimulator(entity.Pool{
			Address: "0xpair", Exchange: "velodrome", Type: velodromev1.DexType,
			Tokens: []*entity.PoolToken{
				{Address: a, Swappable: true, Decimals: 6}, {Address: b, Swappable: true, Decimals: 6},
			},
			Reserves:    entity.PoolReserves{"1000000000000", "3000000000000"},
			Extra:       string(extra),
			StaticExtra: `{"feePrecision":10000}`,
		})
		require.NoError(t, err)
		return sim
	}
	quote := func(sim pool.IPoolSimulator, in, out string, amount int64) (*pool.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(amount)}, TokenOut: out,
		})
	}
	roundTrip := func(t *testing.T, sim pool.IPoolSimulator) pool.IPoolSimulator {
		enc, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"p": sim})
		require.NoError(t, err)
		dec, err := msgpack.DecodePoolSimulatorsMap(enc)
		require.NoError(t, err)
		return dec["p"]
	}

	t.Run("with balances", func(t *testing.T) {
		// balanceOf(pair) is above the reserves on token a (a donation) and the swap is applied before the trip
		sim := newSim(t, big.NewInt(1_000_000_500_000), big.NewInt(3_000_000_000_000))
		res, err := quote(sim, a, b, 1_000_000_000)
		require.NoError(t, err)
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: a, Amount: big.NewInt(1_000_000_000)},
			TokenAmountOut: *res.TokenAmountOut,
			Fee:            *res.Fee,
		})

		decoded := roundTrip(t, sim)
		for _, amount := range []int64{1_000_000, 1_000_000_000} {
			want, err := quote(sim, b, a, amount)
			require.NoError(t, err)
			got, err := quote(decoded, b, a, amount)
			require.NoError(t, err)
			require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
			require.Equal(t, want.Fee, got.Fee)
		}
	})

	t.Run("balance below amount out is still rejected", func(t *testing.T) {
		// the pair holds 1 of token b: no quote can be paid out of it
		sim := newSim(t, big.NewInt(1_000_000_000_000), big.NewInt(1))
		_, err := quote(sim, a, b, 1_000_000_000)
		require.ErrorIs(t, err, velodromev1.ErrInsufficientBalance)

		_, err = quote(roundTrip(t, sim), a, b, 1_000_000_000)
		require.ErrorIs(t, err, velodromev1.ErrInsufficientBalance)
	})

	t.Run("without balances", func(t *testing.T) {
		sim := newSim(t, nil, nil)
		want, err := quote(sim, a, b, 1_000_000_000)
		require.NoError(t, err)

		got, err := quote(roundTrip(t, sim), a, b, 1_000_000_000)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
	})
}

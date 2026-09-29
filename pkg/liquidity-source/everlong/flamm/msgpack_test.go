package everlongflamm

import (
	"bytes"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestMsgpackRoundTrip guards the pool-service -> router-service hop: the whole state lives in unexported fields, so
// it survives only with pkg/msgpack's IncludeUnexported(true) (mirrored here; that package cannot be imported, its
// generated registry imports this one). The decoded simulator quotes identically, and since decoding bypasses
// NewPoolSimulator every quote re-checks the listing and the refresh verdict.
func TestMsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"live", "armed", "capped"} {
		sim := simFor(t, gridReads(t, "51313000", tag), Policy{LeverRouting: true, PriceBandMarginBps: 1})
		// The refresh's end-of-window market oracle answers travel with the state (Extra.OracleAhead): here the
		// snapshot's own, so the gate they drive is inert and the quotes below are the state's.
		for i := range sim.state.Router.Venues {
			m := &sim.state.Router.Venues[i].Morpho
			sim.OracleAhead = append(sim.OracleAhead, OracleAnswer{Ok: m.OracleOk, Price: m.OraclePrice})
		}

		var buf bytes.Buffer
		enc := msgpack.NewEncoder(&buf)
		enc.IncludeUnexported(true)
		enc.SetForceAsArray(true)
		require.NoError(t, enc.Encode(sim))
		dec := msgpack.NewDecoder(&buf)
		dec.IncludeUnexported(true)
		dec.SetForceAsArray(true)
		var decoded PoolSimulator
		require.NoError(t, dec.Decode(&decoded))
		require.Nil(t, decoded.nowFn)
		ts := sim.state.Timestamp
		decoded.nowFn = func() uint64 { return ts }
		require.Empty(t, e2eDiff(sim.state, decoded.state))
		require.Equal(t, sim.state.Timestamp, decoded.state.Timestamp)
		require.Equal(t, sim.Policy, decoded.Policy)
		require.Equal(t, sim.StaticExtra, decoded.StaticExtra)
		require.Equal(t, sim.Venues, decoded.Venues) // the venue set the decoded simulator re-checks on every quote
		require.Equal(t, sim.OracleAhead, decoded.OracleAhead)
		require.NotEmpty(t, decoded.OracleAhead)
		require.Equal(t, []uint64{sim.snapshotTs, sim.scheduledAt, sim.lineage},
			[]uint64{decoded.snapshotTs, decoded.scheduledAt, decoded.lineage})

		for _, c := range []struct {
			sell bool
			a    uint64
		}{{true, 15000}, {false, 20_000_000}, {true, 50_000}, {false, 3_000_000}, {true, 1_000_000}} {
			want, errWant := sim.CalcAmountOut(amountIn(sim, c.sell, c.a))
			got, errGot := decoded.CalcAmountOut(amountIn(&decoded, c.sell, c.a))
			require.Equal(t, errWant, errGot, "%s %+v", tag, c)
			if errWant != nil {
				continue
			}
			require.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
			require.Equal(t, want.RemainingTokenAmountIn.Amount.String(), got.RemainingTokenAmountIn.Amount.String())
			require.Equal(t, want.Gas, got.Gas)
			require.Equal(t, want.SwapInfo.(SwapInfo).Venue, got.SwapInfo.(SwapInfo).Venue)
			// The decoded simulator adopts its own quote.
			clone := decoded.CloneState().(*PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: got.SwapInfo})
			require.Empty(t, e2eDiff(clone.state, want.SwapInfo.(SwapInfo).next))
		}

		params := amountIn(&decoded, true, 15000)
		tampered := decoded
		tampered.Attested = false
		_, err := tampered.CalcAmountOut(params)
		require.ErrorIs(t, err, ErrNotAttested)
		tampered = decoded
		tampered.StaticExtra.Implementation = tampered.StaticExtra.Router
		_, err = tampered.CalcAmountOut(params)
		require.ErrorIs(t, err, ErrInvalidProfile)
	}
}

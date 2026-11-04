package integral

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/nami"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func namiSnapshotPool(t *testing.T, timestamp uint64) entity.Pool {
	t.Helper()
	candidate := p
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(candidate.Extra), &extra))
	extra.NamiFee = &nami.Config{SnapshotTimestamp: timestamp}
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	candidate.Extra = string(raw)
	candidate.Exchange = "nami-v4"
	return candidate
}

func TestNamiSnapshotStaleCheck(t *testing.T) {
	const now = uint32(1793779223)
	old := blockTimestamp
	blockTimestamp = func() uint32 { return now }
	t.Cleanup(func() { blockTimestamp = old })
	for _, tc := range []struct {
		name      string
		timestamp uint64
		stale     bool
	}{
		{"fresh despite old pool activity", uint64(now), false},
		{"expiry boundary", uint64(now) - nami.MaxSnapshotAge, false},
		{"expired refresh", uint64(now) - nami.MaxSnapshotAge - 1, true},
		{"old snapshot without timestamp", 0, true},
		{"future snapshot", uint64(now) + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := namiSnapshotPool(t, tc.timestamp)
			_, err := newPoolSimulatorWithParams(pool.FactoryParams{EntityPool: candidate, Opts: pool.FactoryOpts{StaleCheck: true}})
			if tc.stale {
				require.ErrorIs(t, err, ErrNamiStaleSnapshot)
			} else {
				require.NoError(t, err)
			}
			// Indexing must remain able to construct the same snapshot without an age gate.
			_, err = newPoolSimulatorWithParams(pool.FactoryParams{EntityPool: candidate})
			require.NoError(t, err)
		})
	}
	_, err := newPoolSimulatorWithParams(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err, "other Algebra venues retain existing behavior")
	candidate := p
	candidate.Exchange = "nami-v4"
	_, err = newPoolSimulatorWithParams(pool.FactoryParams{EntityPool: candidate, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.ErrorIs(t, err, ErrNamiStaleSnapshot, "pre-integration Nami snapshots must refresh")
}

func TestNamiCachedQuoteExpiresWithoutRefresh(t *testing.T) {
	now := uint32(1793779223)
	old := blockTimestamp
	blockTimestamp = func() uint32 { return now }
	t.Cleanup(func() { blockTimestamp = old })
	sim, err := newPoolSimulatorWithParams(pool.FactoryParams{EntityPool: namiSnapshotPool(t, uint64(now)), Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	var encoded bytes.Buffer
	encoder := msgpack.NewEncoder(&encoded)
	encoder.IncludeUnexported(true)
	encoder.SetForceAsArray(true)
	require.NoError(t, encoder.Encode(sim))
	decoder := msgpack.NewDecoder(&encoded)
	decoder.IncludeUnexported(true)
	decoder.SetForceAsArray(true)
	var restored PoolSimulator
	require.NoError(t, decoder.Decode(&restored))
	now += uint32(nami.MaxSnapshotAge) + 1
	tokens := sim.GetTokens()
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(1_000_000)}, TokenOut: tokens[1]})
	require.ErrorIs(t, err, ErrNamiStaleSnapshot)
	clone := sim.CloneState().(*PoolSimulator)
	require.ErrorIs(t, clone.checkNamiSnapshot(), ErrNamiStaleSnapshot)
	require.ErrorIs(t, restored.checkNamiSnapshot(), ErrNamiStaleSnapshot)
}

func TestNamiGasBudgetIsNotConsumedGas(t *testing.T) {
	baseline := ps.CloneState().(*PoolSimulator)
	baseline.globalState.PluginConfig = AFTER_SWAP_FLAG
	tokens := baseline.GetTokens()
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(1_000_000)}, TokenOut: tokens[1]}
	q, err := baseline.CalcAmountOut(params)
	require.NoError(t, err)
	for _, tc := range []struct {
		name        string
		storedGas   uint64
		hookEnabled bool
		extraGas    int64
	}{
		{"old five-million snapshot", 5_000_000, true, int64(nami.MEVXGasEstimate)},
		{"new estimate", nami.MEVXGasEstimate, true, int64(nami.MEVXGasEstimate)},
		{"router disabled", 0, true, 0},
		{"afterSwap disabled", 5_000_000, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim := baseline.CloneState().(*PoolSimulator)
			sim.namiFee = &nami.Config{MEVXGas: tc.storedGas, CallGasBudget: 5_000_000}
			if !tc.hookEnabled {
				sim.globalState.PluginConfig = 0
			}
			actual, err := sim.CalcAmountOut(params)
			require.NoError(t, err)
			require.Equal(t, q.Gas+tc.extraGas, actual.Gas)
			require.Equal(t, q.TokenAmountOut.Amount, actual.TokenAmountOut.Amount)
		})
	}
}

package thogprop

import (
	"encoding/json"
	"maps"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func TestSnapshotRejectsIncompleteOrIncompatibleImages(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	for _, mutate := range []func(*poolData){
		func(p *poolData) { p.SchemaVersion = 7 }, func(p *poolData) { p.Words = p.Words[:125] },
		func(p *poolData) { p.PortfolioBalances = p.PortfolioBalances[:7] }, func(p *poolData) { p.Words[1].Slot = 2 },
		func(p *poolData) { p.Tokens[0].VolumeRate = 0 }, func(p *poolData) { p.Categories[0].AnchorTokenIndex = 4 },
		func(p *poolData) { p.BaseFeePerGas = nil }, func(p *poolData) { p.Balances[0] = nil },
		func(p *poolData) { p.Tokens[1].Token = p.Tokens[0].Token },
		func(p *poolData) { p.Words[6].Value.Sub(p.Words[6].Value, new(big.Int).Lsh(big.NewInt(1), 176)) },
	} {
		p := fixturePage(t, sample)
		mutate(&p)
		_, err := snapshotState(p, sample.ShareBalances)
		require.Error(t, err)
	}
	p := fixturePage(t, sample)
	_, err := snapshotState(p, nil)
	require.Error(t, err)
	shares := maps.Clone(sample.ShareBalances)
	shares[f.Proxy] = ShareBalance{}
	_, err = snapshotState(p, shares)
	require.Error(t, err)
}
func TestTrackerSchedulesUpdatesWithoutMakerEvents(t *testing.T) {
	tracker := &PoolTracker{}
	require.Equal(t, int64(1001), tracker.NextUpdateAt(entity.Pool{Timestamp: 1000}))
	f := fixture(t, "execution-model3.json.gz")
	s := stateFrom(t, f.Snapshots[0])
	posted, maxAge := s.small(2, 36, 32), s.small(2, 138, 8)
	s.BlockNumber = posted + maxAge
	// The router's explicit one-block inclusion policy crosses the actual
	// contract cutoff; it does not silently clamp the age or use old quotes.
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	_, err = sim.quoteAtCapture(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[0].Address, Amount: big.NewInt(1000000)}, TokenOut: s.Tokens[1].Address})
	require.ErrorIs(t, err, ErrStaleSnapshot)
}

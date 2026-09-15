package lunarbase

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestSingleCallSnapshotReplacesMetadataAtLatest(t *testing.T) {
	f := newVerifiedFixture()
	p := verifiedEntity()
	p.Extra = `{"bh":"previous-fork","paused":true,"mp":999}`
	before := publicJSON(t, p)
	tracker := publicTracker(f.client(t))
	tracker.config.SingleCallSnapshot = true
	got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, before, publicJSON(t, p))
	require.Equal(t, uint64(900), got.BlockNumber)
	require.Len(t, f.requests, 1)
	require.Equal(t, "eth_call", f.requests[0].Method)
	require.Equal(t, `"latest"`, string(f.requests[0].Params[1]))
	extra := publicExtra(t, got)
	require.True(t, extra.SnapshotComplete)
	require.Empty(t, extra.BlockHash)
	require.False(t, extra.Paused)
	require.Equal(t, uint32(125000), extra.MaxPunishmentX24)
	require.Equal(t, "1230000000", got.Reserves[0])
	require.Equal(t, "4560000000", got.Reserves[1])

	// Overrides retain a separately verified snapshot even when the option
	// is enabled. The override path must not publish no-hash metadata.
	f.requests = nil
	got, err = tracker.GetNewPoolStateWithOverrides(context.Background(), got, pool.GetNewPoolStateWithOverridesParams{})
	require.NoError(t, err)
	require.Len(t, f.requests, 3)
	require.Equal(t, f.header.Hash().Hex(), publicExtra(t, got).BlockHash)
	require.False(t, publicExtra(t, got).SnapshotComplete)
}

func TestSingleCallSnapshotFailuresPreserveInput(t *testing.T) {
	for _, name := range []string{"zero_block", "behind", "required_revert", "malformed", "missing_result", "both_models", "neither_model"} {
		t.Run(name, func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			switch name {
			case "zero_block":
				f.aggregateBlock = 0
			case "behind":
				p.BlockNumber = 901
			case "required_revert":
				f.items[2].Success = false
			case "malformed":
				f.items[2].ReturnData = []byte{1}
			case "missing_result":
				f.items = f.items[:9]
			case "both_models":
				f.items[3].Success = true
			case "neither_model":
				f.items[4].Success = false
			}
			before := publicJSON(t, p)
			tracker := publicTracker(f.client(t))
			tracker.config.SingleCallSnapshot = true
			got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			require.Error(t, err)
			require.Equal(t, before, publicJSON(t, p))
			require.Equal(t, before, publicJSON(t, got))
		})
	}
}

func TestSingleCallSnapshotKnownZeroMetadata(t *testing.T) {
	for _, name := range []string{"zero_max", "zero_concentration", "zero_update"} {
		t.Run(name, func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			switch name {
			case "zero_max":
				f.items[4].ReturnData = verifiedWord(big.NewInt(0))
			case "zero_concentration":
				f.items[3].Success = true
				f.items[4].Success = false
			case "zero_update":
				f.items[8].ReturnData = verifiedWords(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(15), big.NewInt(25), big.NewInt(0))
			}
			tracker := publicTracker(f.client(t))
			tracker.config.SingleCallSnapshot = true
			got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, ChainID: valueobject.ChainIDBSC})
			require.NoError(t, err)
			_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1000)}, TokenOut: p.Tokens[1].Address})
			if name == "zero_update" {
				require.ErrorIs(t, err, ErrStalePool)
				require.False(t, sim.IsStale(24))
				require.True(t, sim.IsStale(25))
			} else {
				require.NoError(t, err)
				require.Equal(t, name == "zero_concentration", sim.ConcentrationModel)
			}
		})
	}
}

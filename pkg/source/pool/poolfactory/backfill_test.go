package poolfactory

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestNewFilterLogsBackfiller_RequiresMaxBlockRangePerScan(t *testing.T) {
	_, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{MaxBlockRangePerScan: 0})
	require.Error(t, err)

	b, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{MaxBlockRangePerScan: 1})
	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestFilterLogsBackfiller_ParseMetadata(t *testing.T) {
	b, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{StartBlock: 100, MaxBlockRangePerScan: 5000})
	require.NoError(t, err)

	// Cold start: empty metadata resumes from cfg.StartBlock.
	meta, err := b.parseMetadata(nil)
	require.NoError(t, err)
	require.EqualValues(t, 100, meta.LastScannedBlock)
	require.False(t, meta.Done)

	// Warm start: persisted cursor is used regardless of StartBlock.
	persisted, err := json.Marshal(backfillMetadata{LastScannedBlock: 999})
	require.NoError(t, err)
	meta, err = b.parseMetadata(persisted)
	require.NoError(t, err)
	require.EqualValues(t, 999, meta.LastScannedBlock)
	require.False(t, meta.Done)
}

// TestFilterLogsBackfiller_Backfill_DoneShortCircuitsWithoutRPC verifies that
// once a prior run's metadata records Done: true, Backfill returns
// immediately without touching ethrpcClient at all -- proven here by giving
// it a nil client: if Backfill tried to call GetBlockNumber/FilterLogs on it,
// this would panic instead of returning cleanly.
func TestFilterLogsBackfiller_Backfill_DoneShortCircuitsWithoutRPC(t *testing.T) {
	b, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{MaxBlockRangePerScan: 5000})
	require.NoError(t, err)

	doneMetadata, err := json.Marshal(backfillMetadata{LastScannedBlock: 12345, Done: true})
	require.NoError(t, err)

	pools, newMetadata, done, err := b.Backfill(t.Context(), doneMetadata)
	require.NoError(t, err)
	require.True(t, done)
	require.Empty(t, pools)
	require.Equal(t, doneMetadata, newMetadata)
}

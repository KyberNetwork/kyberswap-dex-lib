package poolfactory

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// fakeDecoder implements only the base IPoolFactoryDecoder.
type fakeDecoder struct{}

func (fakeDecoder) IsEventSupported(common.Hash) bool                 { return false }
func (fakeDecoder) DecodePoolCreated(types.Log) (*entity.Pool, error) { return nil, nil }

// fakeDecoderWithTopics additionally implements IPoolFactoryDecoderWithTopics.
type fakeDecoderWithTopics struct {
	fakeDecoder
	topics []common.Hash
}

func (f fakeDecoderWithTopics) SupportedEventTopics() []common.Hash { return f.topics }

func TestNewFilterLogsBackfiller_RequiresMaxBlockRangePerScan(t *testing.T) {
	_, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{MaxBlockRangePerScan: 0})
	require.Error(t, err)

	b, err := NewFilterLogsBackfiller(nil, nil, BackfillConfig{MaxBlockRangePerScan: 1})
	require.NoError(t, err)
	require.NotNil(t, b)
}

// TestNewFilterLogsBackfiller_TopicsDerivedFromDecoder verifies the FilterLogs
// Topics filter is populated from a decoder implementing
// IPoolFactoryDecoderWithTopics, and left nil (fetch-everything, filter
// client-side) for a decoder that doesn't.
func TestNewFilterLogsBackfiller_TopicsDerivedFromDecoder(t *testing.T) {
	hash := common.HexToHash("0x1234")

	withTopics, err := NewFilterLogsBackfiller(
		fakeDecoderWithTopics{topics: []common.Hash{hash}}, nil, BackfillConfig{MaxBlockRangePerScan: 1})
	require.NoError(t, err)
	require.Equal(t, [][]common.Hash{{hash}}, withTopics.topics)

	without, err := NewFilterLogsBackfiller(fakeDecoder{}, nil, BackfillConfig{MaxBlockRangePerScan: 1})
	require.NoError(t, err)
	require.Nil(t, without.topics)
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

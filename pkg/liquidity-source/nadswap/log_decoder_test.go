package nadswap

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to assemble Sync event log data: two uint112 (left-padded to 32 bytes each).
func encodeSyncData(r0, r1 uint64) []byte {
	buf := make([]byte, 64)
	new(big.Int).SetUint64(r0).FillBytes(buf[:32])
	new(big.Int).SetUint64(r1).FillBytes(buf[32:])
	return buf
}

func TestLogDecoder_DecodesLatestSync(t *testing.T) {
	t.Parallel()
	dec := NewLogDecoder()

	syncTopic := pairABI.Events["Sync"].ID

	oldLog := types.Log{
		Address:     common.HexToAddress("0xpair"),
		Topics:      []common.Hash{syncTopic},
		Data:        encodeSyncData(100, 200),
		BlockNumber: 10,
		TxIndex:     0,
		Index:       0,
	}
	newLog := types.Log{
		Address:     common.HexToAddress("0xpair"),
		Topics:      []common.Hash{syncTopic},
		Data:        encodeSyncData(300, 400),
		BlockNumber: 11,
		TxIndex:     2,
		Index:       5,
	}

	data, blockNum, err := dec.Decode([]types.Log{oldLog, newLog}, nil)
	require.NoError(t, err)
	require.NotNil(t, data.Reserve0)
	require.NotNil(t, data.Reserve1)
	assert.Equal(t, "300", data.Reserve0.Dec())
	assert.Equal(t, "400", data.Reserve1.Dec())
	assert.Equal(t, uint64(11), blockNum.Uint64())
}

func TestLogDecoder_NoSyncEvents(t *testing.T) {
	t.Parallel()
	dec := NewLogDecoder()
	data, blockNum, err := dec.Decode([]types.Log{}, nil)
	require.NoError(t, err)
	assert.True(t, data.IsZero())
	assert.Nil(t, blockNum)
}

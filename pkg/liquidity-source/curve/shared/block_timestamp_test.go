package shared

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"
)

func TestAddBlockTimestampCall(t *testing.T) {
	var calls []*ethrpc.Call
	add := func(c *ethrpc.Call, _ []any) { calls = append(calls, c) }
	var ts *big.Int
	(&Config{}).AddBlockTimestampCall(add, &ts)
	require.Empty(t, calls, "no multicall, no call")
	(&Config{MulticallAddress: "0xca11bde05977b3631167028862be2a173976ca11"}).AddBlockTimestampCall(add, &ts)
	require.Len(t, calls, 1)
	require.Equal(t, ethrpc.MethodGetCurrentBlockTimestamp, calls[0].Method)
	require.Equal(t, "0xca11bde05977b3631167028862be2a173976ca11", calls[0].Target)
}

// A timestamp from the batch is used as is, with no read; without one and without a block there is none to read.
func TestBlockTimestamp(t *testing.T) {
	client := ethrpc.New("http://127.0.0.1:1") // any read fails
	got, err := BlockTimestamp(context.Background(), client, nil, big.NewInt(12), big.NewInt(1_795_000_000))
	require.NoError(t, err)
	require.EqualValues(t, 1_795_000_000, got)
	_, err = BlockTimestamp(context.Background(), client, nil, nil, nil)
	require.ErrorIs(t, err, ErrNoReadBlock)
	_, err = BlockTimestamp(context.Background(), client, nil, big.NewInt(12), nil)
	require.Error(t, err, "no timestamp from the batch: it is read at the block")
}

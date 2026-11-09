package dexv2

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestDropAppliedLogs(t *testing.T) {
	dex := common.HexToAddress("0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261")
	logAt := func(block uint64, removed bool) ethtypes.Log {
		return ethtypes.Log{Address: dex, BlockNumber: block, Removed: removed}
	}
	trigger := ethtypes.Log{} // empty log of an interval task
	logs := []ethtypes.Log{trigger, logAt(99, false), logAt(100, false), logAt(100, true), logAt(101, false)}

	// No bookkeeping yet (legacy state or event-created pool): nothing is dropped.
	require.Equal(t, logs, dropAppliedLogs(logs, 0))

	// State already includes blocks <= 100. Events there would be counted twice; the trigger and the
	// reverted-block log (resolved via RPC) are kept.
	got := dropAppliedLogs(logs, 100)
	require.Len(t, got, 3)
	require.True(t, got[0].Address == (common.Address{}), "trigger kept")
	require.True(t, got[1].Removed, "reverted log kept")
	require.Equal(t, uint64(101), got[2].BlockNumber)

	// Everything already applied: only the trigger remains, so the update still refreshes scalars.
	require.Equal(t, []ethtypes.Log{trigger}, dropAppliedLogs([]ethtypes.Log{trigger, logAt(5, false)}, 100))
}

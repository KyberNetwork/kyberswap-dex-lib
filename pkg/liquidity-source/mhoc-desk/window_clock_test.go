package mhocdesk

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestWindowBudgetFollowsTheChainNotTheWallClock: the snapshot was read at timestamp 999 (minute 16) with the
// minute's budget used up. The wall clock already reads 1,020 (minute 17), but the next block can land at 1,000,
// still in minute 16, where the pool reverts with ExceedsWindowCap. The simulator must not quote it.
func TestWindowBudgetFollowsTheChainNotTheWallClock(t *testing.T) {
	t.Parallel()
	in := readmeInput()
	in.Extra.StateWord = card(3000_000000*1e8, 100, 100, 250, 250, 500_000, 1, 2_000, 20)
	in.Extra.MaxPerWindow = 25_000
	in.Extra.Counters.WindowID = 999 / 60
	in.Extra.Counters.WindowFilled = uint256.NewInt(24_000_000000) // 24,000 of 25,000 USDC used
	in.Extra.ReadTimestamp = 999
	in.SellBase, in.AmountIn = true, pow10(18) // about 3,000 USDC
	in.Now = 1_020

	// the wall clock alone frees the budget: the quote the old code gave, which reverts on chain
	_, err := Quote(in)
	require.NoError(t, err)

	sim := &PoolSimulator{extra: *in.Extra}
	require.Equal(t, uint64(1_000), sim.windowNow(1_020))
	require.Equal(t, uint64(999), sim.windowNow(999))

	in.WindowNow = sim.windowNow(in.Now)
	_, err = Quote(in)
	require.ErrorIs(t, err, ErrExceedsWindowCap)

	// a snapshot read in the last second of a minute: the next block is in the next minute, the budget is free
	in.Extra.ReadTimestamp = 1_019
	in.Extra.Counters.WindowID = 1_019 / 60
	sim = &PoolSimulator{extra: *in.Extra}
	in.WindowNow = sim.windowNow(in.Now)
	require.Equal(t, uint64(1_020), in.WindowNow)
	_, err = Quote(in)
	require.NoError(t, err)

	// counters written by the swap are tagged with the same minute, so a second swap of the route sees the first
	in.Extra.ReadTimestamp = 999
	in.Extra.Counters.WindowID = 999 / 60
	in.Extra.Counters.WindowFilled = uint256.NewInt(20_000_000000)
	in.WindowNow = 1_000
	r, err := Quote(in)
	require.NoError(t, err)
	require.Equal(t, uint32(16), r.Counters.WindowID)
	require.True(t, r.Counters.WindowFilled.Gt(uint256.NewInt(20_000_000000)))
}

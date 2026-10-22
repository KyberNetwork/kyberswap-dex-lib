package integral

import (
	"context"
	"fmt"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// lens is AlgebraIntegralPluginLens over the chain: it makes the calls, then walks the timepoints
// by the contract's rules, reading at most perCall of them a call (0 for no limit), as the gas of a
// call would cut it short.
func (c *fakeChain) lens(perCall int) func(context.Context, *big.Int, common.Address, [][]byte,
	pluginWalk) (pluginSnapshot, error) {
	return func(_ context.Context, _ *big.Int, _ common.Address, calls [][]byte,
		walk pluginWalk) (pluginSnapshot, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.lensCalls++
		snap := pluginSnapshot{BlockNumber: big.NewInt(1), Walked: true, Done: true, First: walk.Start}
		for _, data := range calls {
			ok, out, err := c.reply(data)
			if err != nil {
				return snap, err
			}
			snap.Reads = append(snap.Reads, callResult{Success: ok, ReturnData: out})
		}
		if walk.FromHead {
			snap.Head, snap.First = c.index, c.index
		}
		next, count := snap.First, int64(0)
		for count < walk.MaxCount.Int64() {
			if perCall > 0 && count == int64(perCall) {
				snap.Done = false
				break
			}
			out := lo.Must(c.packTimepoint(next))
			snap.Timepoints = append(snap.Timepoints, out...)
			count++
			ts := c.slots[next]
			if ts == 0 || ts < walk.Cutoff || walk.HasKnown && next == walk.KnownIndex && ts == walk.KnownTimestamp {
				break
			}
			next--
		}
		snap.Next, snap.Count = next, big.NewInt(count)
		if walk.FromHead {
			for _, idx := range []uint16{0, c.index + 1, c.index + 2, c.index - 1} {
				snap.Extras = append(snap.Extras, lo.Must(c.packTimepoint(idx))...)
			}
		}
		return snap, nil
	}
}

// pluginData is what getPluginData returns.
type pluginData struct {
	timepoints map[uint16]Timepoint
	vo         *VolatilityOraclePlugin
	dynamicFee *DynamicFeeConfig
	slidingFee *SlidingFeeConfig
}

// readBoth reads the plugin of a pool holding cache by multicall, then through the lens, requires
// the two to agree, and returns what they read and the calls each took: multicalls, then lens calls
// and the multicalls the lens path fell back to.
func readBoth(t *testing.T, chain *fakeChain, cache map[uint16]Timepoint, perCall int) (pluginData, int, int,
	int) {
	t.Helper()
	p := &entity.Pool{Address: "0x7491c04dc4575e086a8ee31f7ce1c6d56fb7dcc1",
		Extra: string(lo.Must(json.Marshal(Extra{ExtraTimepoint: ExtraTimepoint{Timepoints: cache}})))}
	plugin := common.HexToAddress("0x3E6e46Ea6E7f5D659dD1b3aB889cb8Bc199A072F")
	read := func(tracker *PoolTracker) pluginData {
		var d pluginData
		var err error
		d.timepoints, d.vo, d.dynamicFee, d.slidingFee, err = tracker.getPluginData(context.Background(), p, plugin,
			big.NewInt(1), nil)
		require.NoError(t, err)
		return d
	}

	byRPC := newTrackerOnChain(t, chain)
	byRPC.config = &Config{}
	chain.multicalls, chain.lensCalls = 0, 0
	want := read(byRPC)
	multicalls := chain.multicalls

	byLens := newTrackerOnChain(t, chain)
	byLens.config = &Config{UseLens: true}
	byLens.pluginLensCall = chain.lens(perCall)
	chain.multicalls, chain.lensCalls = 0, 0
	got := read(byLens)
	require.Equal(t, want, got)
	return got, multicalls, chain.lensCalls, chain.multicalls
}

// TestPluginLens_MatchesMulticall: the lens path reads the same plugin data and timepoints as the
// multicalls from the same cache, in one call, whether the pool is read for the first time or
// refreshed after new timepoints, and when a walk takes several calls.
func TestPluginLens_MatchesMulticall(t *testing.T) {
	t.Parallel()
	const yesterday = tpNow - WINDOW

	tests := []struct {
		name    string
		index   uint16
		n       int
		full    bool
		spacing uint32
		// age moves every timestamp back, so the newest is age seconds old.
		age uint32
	}{
		{"wrapped, a day within the ring", 1000, 200, true, tpSpacing, 0},
		{"across the uint16 wrap", 65530, 200, true, tpSpacing, 0},
		{"not wrapped yet, younger than a day", 100, 101, false, tpSpacing, 0},
		{"one timepoint", 0, 1, false, tpSpacing, 0},
		{"more within a day than one read takes", 30000, 6000, true, 10, 0},
		{"the newest is older than a day", 1000, 10, false, tpSpacing, 2 * WINDOW},
		{"the newest just inside the day", 1000, 300, true, tpSpacing, WINDOW - 100},
	}
	for _, tc := range tests {
		for _, perCall := range []int{0, 7, 100} {
			t.Run(fmt.Sprintf("%s, %d a call", tc.name, perCall), func(t *testing.T) {
				t.Parallel()
				chain := newFakeChain(tc.index, 0, tc.full)
				for i := range tc.n {
					chain.slots[tc.index-uint16(i)] = tpNow - tc.age - uint32(i)*tc.spacing
				}

				first, multicalls, lensCalls, fallbacks := readBoth(t, chain, nil, perCall)
				require.NotNil(t, first.vo)
				require.NotNil(t, first.dynamicFee)
				require.Zero(t, fallbacks, "the lens path fell back to multicalls")
				if perCall == 0 {
					require.Equal(t, 1, lensCalls)
				}
				t.Logf("first read: %d multicalls, %d lens calls, %d timepoints", multicalls, lensCalls,
					len(first.timepoints))

				cache := first.timepoints
				for _, k := range []int{0, 1, 5, 400} {
					// New timepoints older than a day, which a tracker keeping up never sees, can take
					// the lens path back to multicalls (see TestPluginLens_FallsBackToMulticall).
					oldNew := k > 0 && chain.headTs()+tpSpacing < yesterday
					chain.write(k)
					next, multicalls, lensCalls, fallbacks := readBoth(t, chain, cache, perCall)
					if !oldNew {
						require.Zero(t, fallbacks, "the lens path fell back to multicalls after %d new", k)
					}
					if perCall == 0 {
						// A second call when the cache ends on an uninitialized slot, a pool younger
						// than a day, or short of a day, a read capped at MaxTimepointReads.
						require.LessOrEqual(t, lensCalls, min(2, multicalls), "after %d new", k)
					}
					if next.timepoints != nil {
						require.GreaterOrEqual(t, next.timepoints[chain.index].BlockTimestamp, min(chain.headTs(),
							uint32(yesterday)))
					}
					t.Logf("after %d new: %d multicalls, %d lens calls", k, multicalls, lensCalls)
					cache = next.timepoints
				}
			})
		}
	}
}

// TestPluginLens_FallsBackToMulticall: when the multicalls would keep a slot the lens did not read,
// a slot past the first one too old to keep, the lens path reads the plugin by multicall instead.
// That takes a cache whose newest timepoint is older than a day, and new ones older than a day too.
func TestPluginLens_FallsBackToMulticall(t *testing.T) {
	t.Parallel()
	chain := newFakeChain(1000, 0, false)
	for i := range 10 {
		chain.slots[1000-uint16(i)] = tpNow - 2*WINDOW - uint32(i)*tpSpacing
	}
	first, _, lensCalls, fallbacks := readBoth(t, chain, nil, 0)
	require.Equal(t, 1, lensCalls)
	require.Zero(t, fallbacks)

	chain.write(3)
	_, _, lensCalls, fallbacks = readBoth(t, chain, first.timepoints, 0)
	require.Equal(t, 1, lensCalls)
	require.Positive(t, fallbacks)
}

// TestPluginLens_MatchesMulticallRandomly is TestPluginLens_MatchesMulticall over random rings,
// ages and refreshes.
func TestPluginLens_MatchesMulticallRandomly(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 60 {
		index, n, full := uint16(rng.Uint32()), 1+rng.IntN(3000), rng.IntN(2) == 0
		spacing, age, perCall := 1+rng.Uint32N(1200), rng.Uint32N(2*WINDOW), []int{0, 0, 50, 1000}[rng.IntN(4)]
		writes := []int{rng.IntN(200), rng.IntN(200), rng.IntN(200)}
		t.Run(fmt.Sprintf("%d: index %d, %d by %ds, %ds old, full %v, %d a call", i, index, n, spacing, age, full,
			perCall), func(t *testing.T) {
			t.Parallel()
			chain := newFakeChain(index, 0, full)
			for j := range n {
				chain.slots[index-uint16(j)] = tpNow - age - uint32(j)*spacing
			}
			read, _, _, _ := readBoth(t, chain, nil, perCall)
			for _, k := range writes {
				chain.write(k)
				read, _, _, _ = readBoth(t, chain, read.timepoints, perCall)
			}
		})
	}
}

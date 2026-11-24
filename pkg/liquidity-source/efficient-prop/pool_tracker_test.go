package efficientprop

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// trackedChain is listing 1 of a WETH/USDC front at about 3,000 USDC, levels widening with size: selling up to
// 5 WETH, buying up to 10,000 USDC, with a minimum fill on each side.
func trackedChain() *fakeChain {
	chain := testChain()
	l := chain.listings[1]
	l.sides = [2]fakeSide{
		{rungs: []fakeRung{{1e18, 3, 1e9}, {2e18, 2997, 1e12}, {5e18, 2990, 1e12}}, minOut: 1e6},
		{rungs: []fakeRung{{3000e6, 333_000_333, 1}, {10_000e6, 332_000_000, 1}}, minOut: 1e14},
	}
	chain.balances[[2]common.Address{testWETH, testPoolA}] = big.NewInt(9e18)
	chain.balances[[2]common.Address{testUSDC, testPoolA}] = big.NewInt(40_000e6)
	return chain
}

// listed is the chain's listing 1 as the lister records it, with the decimals the pool service adds after listing.
func listed(t *testing.T, n *node) entity.Pool {
	t.Helper()
	u, err := NewPoolsListUpdater(testConfig(), n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	p := pools[0]
	p.Tokens[0].Decimals, p.Tokens[1].Decimals = 18, 6
	return p
}

func track(t *testing.T, n *node, p entity.Pool) entity.Pool {
	t.Helper()
	tr, err := NewPoolTracker(testConfig(), n.client())
	require.NoError(t, err)
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	return got
}

func extraOf(t *testing.T, p entity.Pool) Extra {
	t.Helper()
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	return e
}

// Both directions are sampled by the door's own quote at one block and at no gas price (tip 0), every point is
// exactly the door's answer, and the reserves are the listing's pool's custody.
func TestTrackerSamplesTheDoorAtOneBlockAndTipZero(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	before := len(n.calls())

	got := track(t, n, p)
	assert.Equal(t, uint64(1234), got.BlockNumber)
	assert.Equal(t, entity.PoolReserves{"9000000000000000000", "40000000000"}, got.Reserves)

	e := extraOf(t, got)
	assert.Equal(t, uint64(1e8), e.TipCapWei)
	// valid until the sampled window's end, the state's age bound plus the allowance, short of the word's end
	v := chain.listings[1].view
	assert.Equal(t, uint64(got.Timestamp)+uint64(maxAge/time.Second)+subSecondInclusionAllowance, e.ValidUntil)
	assert.Less(t, e.ValidUntil, validUntil(v, 30_000, 0, [2]bool{true, true}))
	assert.Equal(t, uint64(subSecondInclusionAllowance), e.Allowance, "Base's default allowance")
	assert.False(t, e.BaseToken1, "the pair word's bit 8: the base is token 0")
	for dir := range 2 {
		require.NotEmpty(t, e.Ladders[dir], "direction %d", dir)
		for _, pt := range e.Ladders[dir] {
			want, err := chain.listings[1].sides[dir].quote(pt[0].ToBig())
			require.NoError(t, err)
			assert.Equal(t, want, pt[1].ToBig())
		}
	}
	// the ladder ends at the last size the door fills: 5 WETH, 10,000 USDC at most
	top0 := e.Ladders[0][len(e.Ladders[0])-1][0]
	assert.LessOrEqual(t, top0.Uint64(), uint64(5e18))

	calls := n.calls()[before:]
	require.NotEmpty(t, calls)
	assert.Equal(t, "latest", calls[0].tag, "the first read takes the block")
	quotes := 0
	for _, c := range calls {
		assert.False(t, c.gasPrice, "a quote at a gas price is at a tip")
		if c.tag != "latest" {
			assert.Equal(t, hexutil.EncodeUint64(1234), c.tag)
		}
		if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == "0xb6466384" {
			quotes++
			assert.Equal(t, hexutil.EncodeUint64(1234), c.tag, "every quote is pinned")
		}
	}
	assert.Greater(t, quotes, 20)
}

// The simulator of a tracked listing never pays more than the door at the same state, at any size, either way.
func TestSimulatorOfATrackedListingNeverQuotesAboveTheDoor(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	// a second refresh sizes from the first ladder
	p = track(t, n, p)

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	tokens := [2]string{hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])}
	tops := [2]int64{6e18, 12_000e6}
	rng := rand.New(rand.NewSource(3))
	quoted := 0
	for dir := range 2 {
		for range 2000 {
			x := big.NewInt(rng.Int63n(tops[dir]) + 1)
			res, err := calc(sim, tokens[dir], tokens[1-dir], x, nil)
			if err != nil {
				continue
			}
			want, err := chain.listings[1].sides[dir].quote(x)
			require.NoError(t, err, "direction %d: the simulator quotes %s where the door refuses", dir, x)
			require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(want), 0, "direction %d at %s", dir, x)
			// and it gives up little: the chord of a concave walk sampled densely
			diff := new(big.Int).Sub(want, res.TokenAmountOut.Amount)
			assert.LessOrEqual(t, new(big.Int).Mul(diff, big.NewInt(10_000)).Cmp(new(big.Int).Mul(want, big.NewInt(50))), 0,
				"direction %d at %s: more than 50 bp under the door", dir, x)
			quoted++
		}
	}
	assert.Greater(t, quoted, 2000)
}

// The state is stamped with the refresh block's own time, read in the pinned probe, so its age runs from the state
// it holds, not from the end of the refresh's reads.
func TestTrackerStampsTheBlocksTime(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	n.time = uint64(time.Now().Unix()) - 4
	p := track(t, n, listed(t, n))
	assert.Equal(t, int64(n.time), p.Timestamp)

	// a listing not quoted is stamped the same way
	chain.listings[1].view.CreditMode = 1
	p = track(t, n, p)
	assert.Equal(t, int64(n.time), p.Timestamp)
}

// A later refresh probes past the last ladder's top, so a listing whose depth grew is followed.
func TestTrackerProbesPastTheLastTop(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	top := extraOf(t, p).Ladders[0]
	topIn := top[len(top)-1][0].ToBig()

	tr, err := NewPoolTracker(testConfig(), n.client())
	require.NoError(t, err)
	sizes := tr.probeSizes(p, 0, big.NewInt(40_000e6))
	assert.Contains(t, sizes, new(big.Int).Lsh(topIn, 1))
	assert.Contains(t, sizes, new(big.Int).Lsh(topIn, 2))
	for k := 1; k < len(sizes); k++ {
		assert.Equal(t, 1, sizes[k].Cmp(sizes[k-1]), "ascending, no repeats")
	}
}

// The state names the pair's base by the pair word's bit 8, for the gas of each direction.
func TestTrackerNamesThePairsBase(t *testing.T) {
	chain := trackedChain()
	chain.listings[1].view.PairWord = new(big.Int)
	n := chain.node(t, 1234)
	assert.True(t, extraOf(t, track(t, n, listed(t, n))).BaseToken1)
}

// A dark side has no ladder, and the token it would pay out is no liquidity.
func TestTrackerDarkSide(t *testing.T) {
	chain := trackedChain()
	chain.listings[1].sides[1].dark = true
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	assert.NotEmpty(t, e.Ladders[0])
	assert.Empty(t, e.Ladders[1])
	assert.Equal(t, entity.PoolReserves{"0", "40000000000"}, p.Reserves)
}

// A listing with a credit mode is not quoted on a front read at the head (Base), and not sampled: a WORD listing, and
// a PRINT listing on an approved source.
func TestTrackerSkipsACreditListing(t *testing.T) {
	for _, kind := range []string{"WORD", "PRINT"} {
		chain := trackedChain()
		n := chain.node(t, 1234)
		if kind == "PRINT" {
			chain, n = printListing(t, 0, 8_000)
		}
		p := listed(t, n)
		require.NotEmpty(t, extraOf(t, track(t, n, p)).Ladders[0], "%s: served with no credit mode", kind)
		chain.listings[1].view.CreditMode = 1
		before := len(n.calls())
		p = track(t, n, p)
		e := extraOf(t, p)
		assert.Empty(t, e.Ladders[0], kind)
		assert.Empty(t, e.Ladders[1], kind)
		assert.Equal(t, uint8(1), e.CreditMode, kind)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves, kind)
		for _, c := range n.calls()[before:] {
			assert.NotEqual(t, "0xb6466384", hexutil.Encode(c.data[:min(4, len(c.data))]), kind)
		}
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
	}
}

// A listing whose tip cap fell under the configured floor after it was listed is not quoted, and keeps its cap in
// the state.
func TestTrackerSkipsAListingUnderTheTipCapFloor(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	chain.listings[1].view.TipCap = 0x7005 // 5e7 wei
	cfg := testConfig()
	cfg.MinTipCapWei = 1e8
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	p, err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	e := extraOf(t, p)
	assert.Empty(t, e.Ladders[0])
	assert.Empty(t, e.Ladders[1])
	assert.Equal(t, uint64(5e7), e.TipCapWei)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
}

// The configured inclusion allowance goes into the state, for route finding.
func TestTrackerCarriesTheInclusionAllowance(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	cfg := testConfig()
	cfg.InclusionAllowance = 7
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	p, err := tr.GetNewPoolState(context.Background(), listed(t, n), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(7), extraOf(t, p).Allowance)
}

// ctlWord is a controls word: widening (ppm/sqrt s) at bits 136-155, premium (ppm) at 156-175, its window (ms) at
// 176-191 and the grace (ms) at 192-207.
func ctlWord(widen, premium, windowMs, graceMs int64) *big.Int {
	w := new(big.Int).Lsh(big.NewInt(widen), 136)
	w.Or(w, new(big.Int).Lsh(big.NewInt(premium), 156))
	w.Or(w, new(big.Int).Lsh(big.NewInt(windowMs), 176))
	return w.Or(w, new(big.Int).Lsh(big.NewInt(graceMs), 192))
}

// fill is a block a fill may land in: its second, its number and the sub-second BaseTime stores in it (-1: the
// head's, where the front's clock does not read BaseTime).
type fill struct {
	at, number uint64
	sub        int64
}

// fills are the blocks a fill quoted from the state may land in: on the fake chain's mode 2 clock every block from the
// state's on, 200 ms apart, by the count where the count places the state's block in its own second; where it does not,
// the front reads BaseTime, so by BaseTime's reading at the state's block, each block storing its own sub-second, and
// in each second also its last moment (999 ms), the latest any block of that second may store. On its mode 0 clock
// every second. None after until.
func (c *fakeChain) fills(p entity.Pool, until uint64) []fill {
	var out []fill
	if c.clockMode == 0 {
		for at := uint64(p.Timestamp); at <= until; at++ {
			out = append(out, fill{at, p.BlockNumber, -1})
		}
		return out
	}
	if count := c.clockMs0 + (p.BlockNumber-c.clockBlock0)*200; count/1000 == uint64(p.Timestamp) {
		for number := p.BlockNumber; ; number++ {
			at := (c.clockMs0 + (number-c.clockBlock0)*200) / 1000
			if at > until {
				return out
			}
			out = append(out, fill{at, number, -1})
		}
	}
	ms0 := uint64(p.Timestamp)*1000 + c.subMs
	for k := uint64(0); ; k++ {
		ms := ms0 + k*200
		if ms/1000 > until {
			return out
		}
		number := p.BlockNumber + k
		out = append(out, fill{ms / 1000, number, int64(ms % 1000)})
		if next := ms + 200; next/1000 != ms/1000 && ms%1000 != 999 {
			out = append(out, fill{ms / 1000, number, 999})
		}
	}
}

// checkWindow prices the state's simulator against the door in every block a fill quoted from it may land in, at
// sizes on and between the ladder's points: it never pays more.
func checkWindow(t *testing.T, chain *fakeChain, p entity.Pool) {
	t.Helper()
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	e := extraOf(t, p)
	require.LessOrEqual(t, e.ValidUntil, uint64(p.Timestamp)+uint64(maxAge/time.Second)+e.Allowance,
		"valid past the sampled window")
	l := chain.listings[1]
	tokens := [2]string{hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])}
	checked := 0
	for dir := range 2 {
		for _, x := range forkSizes(e.Ladders[dir]) {
			res, err := calc(sim, tokens[dir], tokens[1-dir], x, nil)
			if err != nil {
				continue
			}
			for _, b := range chain.fills(p, e.ValidUntil) {
				door, err := chain.quoteAtSub(l, dir, x, b.at, b.number, b.sub)
				require.NoError(t, err, "direction %d at %s: the door refuses in block %+v", dir, x, b)
				nowMs, _ := chain.clockAtSub(b.at, b.number, b.sub)
				require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(door), 0,
					"direction %d at %s: the state pays %s, the door at +%d ms %s", dir, x, res.TokenAmountOut.Amount,
					nowMs-uint64(p.Timestamp)*1000, door)
				checked++
			}
		}
	}
	assert.Greater(t, checked, 100)
}

// A listing whose levels drift or widen with age pays less as its word ages: the ladder holds for its whole window,
// sampled at the block's time and in the window's last block in the same pinned state, whatever the sub-second the
// state's block and the window's end fall on.
func TestTrackerLadderHoldsForItsWindowOnAnAgingListing(t *testing.T) {
	for _, sub := range []uint64{0, 200, 400, 600, 800} {
		chain := trackedChain()
		chain.subMs = sub
		chain.listings[1].decayPpmPerS = 100
		chain.listings[1].controls = ctlWord(200, 0, 0, 0)
		n := chain.node(t, 1234)
		p := track(t, n, listed(t, n))
		p = track(t, n, p)
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0])
		require.NotEmpty(t, e.Ladders[1])
		assert.Equal(t, uint64(p.Timestamp)+uint64(maxAge/time.Second)+e.Allowance, e.ValidUntil,
			"valid to the sampled window's end")
		checkWindow(t, chain, p)

		// the window's end is sampled in its second's last block, pinned to the same block's state
		fills := chain.fills(p, e.ValidUntil)
		last := fills[len(fills)-1]
		sampled := false
		for _, c := range n.calls() {
			if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == "0xb6466384" {
				assert.Equal(t, hexutil.EncodeUint64(1234), c.tag)
				sampled = sampled || c.at == last.at && c.number == last.number
			}
		}
		assert.True(t, sampled, "the window's last block %v is sampled, sub-second %d ms", last, sub)
	}
}

// On a clock that reads the block's second alone (mode 0), the window's end is sampled at its second, the age clock
// reading its second plus the age slack.
func TestTrackerLadderHoldsForItsWindowOnASecondsClock(t *testing.T) {
	chain := trackedChain()
	chain.clockMode, chain.ageSlackMs = 0, 999
	chain.listings[1].decayPpmPerS = 100
	chain.listings[1].controls = ctlWord(200, 0, 0, 0)
	n := chain.node(t, 1234)
	p := track(t, n, track(t, n, listed(t, n)))
	assert.Equal(t, uint64(p.Timestamp)+uint64(maxAge/time.Second)+subSecondInclusionAllowance, extraOf(t, p).ValidUntil)
	checkWindow(t, chain, p)
}

// A size the door fills at the block's time but refuses later in the window (a listing whose depth or source runs
// out within it) is no point: the ladder ends below it.
func TestTrackerDropsASizeRefusedLaterInTheWindow(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.shrinkAt, l.shrinkTo = uint64(time.Now().Unix())+5, 2e18
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	ladder0 := extraOf(t, p).Ladders[0]
	require.NotEmpty(t, ladder0)
	assert.LessOrEqual(t, ladder0[len(ladder0)-1][0].Uint64(), uint64(2e18))
	assert.NotEmpty(t, extraOf(t, p).Ladders[1], "the other side is not shrunk")
}

// A premium after the maker's post lowers early fills: on an aging listing whose premium outweighs the rest of the
// window's aging, the worst block is the premium window's last, which is sampled too, wherever in its second the
// premium ends and whatever the sub-second of the state's block.
func TestTrackerLadderHoldsThroughAPremiumWindow(t *testing.T) {
	for _, sub := range []uint64{0, 200, 400, 600, 800} {
		for _, windowMs := range []int64{1_800, 2_100, 2_500, 6_000} {
			chain := trackedChain()
			chain.subMs = sub
			l := chain.listings[1]
			l.decayPpmPerS = 100
			l.controls = ctlWord(200, 2_000, windowMs, 0)
			n := chain.node(t, 1234)
			// the word posted in the state's block
			l.view.Word = testWord(int64(n.time*1000+sub), 30)
			p := track(t, n, listed(t, n))
			t.Logf("sub-second %d ms, premium window %d ms", sub, windowMs)
			checkWindow(t, chain, p)
		}
	}
}

// A premium that ended earlier in the state's own second, before the state's block, is over in every block a fill
// can land in: it is not sampled (no quote in a block whose clock reads before the state's), and the ladder pays what
// the door pays at the state's block. One that ends later in that second is sampled, at the state's moment or after
// it. The block-count clock is anchored 50 blocks before the head, as on Base, where the anchor lies far in the past;
// or its anchor does not place the head in its own second, and the front reads the block's milliseconds from
// BaseTime.
func TestTrackerSamplesNoPremiumThatEndedBeforeTheBlock(t *testing.T) {
	quote := hexutil.Encode(doorABI.Methods["quote"].ID)
	for _, anchored := range []bool{true, false} {
		for _, c := range []struct {
			sinceMs uint64 // the word's post, before the head
			over    bool   // the premium is over at the head
		}{{2_100, true}, {2_550, true}, {2_000, true}, {1_900, false}, {1_500, false}} {
			chain := trackedChain()
			chain.subMs = 600
			l := chain.listings[1]
			l.controls = ctlWord(0, 2_000, 2_000, 0) // a premium of 2,000 ppm for 2 s, no aging
			n := chain.node(t, 1234)
			// the head reads T.600, by its anchor 50 blocks earlier, or else by BaseTime
			chain.clockBlock0, chain.clockMs0 = 1234-50, n.time*1000+600-50*200
			if !anchored {
				chain.clockMs0 -= 5_000
			}
			headMs, _ := chain.clockAt(n.time, 1234)
			require.Equal(t, n.time*1000+600, headMs)
			l.view.Word = testWord(int64(headMs-c.sinceMs), 30)
			p := track(t, n, listed(t, n))
			e := extraOf(t, p)
			require.NotEmpty(t, e.Ladders[0])
			for _, call := range n.calls() {
				if len(call.data) >= 4 && hexutil.Encode(call.data[:4]) == quote {
					ms, _ := chain.clockAtSub(call.at, call.number, call.sub)
					assert.GreaterOrEqual(t, ms, headMs, "anchored %v, posted %d ms before: a quote sampled at a "+
						"moment before the state's", anchored, c.sinceMs)
				}
			}
			for dir := range 2 {
				for _, pt := range e.Ladders[dir] {
					door, err := chain.quoteAt(l, dir, pt[0].ToBig(), n.time, 1234)
					require.NoError(t, err)
					if c.over {
						assert.Equal(t, door.String(), pt[1].ToBig().String(), "anchored %v, posted %d ms before, "+
							"direction %d at %s: the ladder against the door at the state's block, the premium over",
							anchored, c.sinceMs, dir, pt[0])
					} else {
						assert.LessOrEqual(t, pt[1].ToBig().Cmp(door), 0)
					}
				}
			}
			// its fills are the anchor's blocks, or off the anchor BaseTime's, each second's last moment included
			checkWindow(t, chain, p)
		}
	}
}

// On a clock that reads the block's second alone (mode 0), the premium is sampled in the last second its clock reads
// under the premium's end.
func TestTrackerLadderHoldsThroughAPremiumWindowOnASecondsClock(t *testing.T) {
	for _, windowMs := range []int64{1_800, 2_000, 2_500} {
		chain := trackedChain()
		chain.clockMode, chain.ageSlackMs = 0, 999
		l := chain.listings[1]
		l.decayPpmPerS = 100
		l.controls = ctlWord(200, 2_000, windowMs, 0)
		n := chain.node(t, 1234)
		l.view.Word = testWord(int64(n.time*1000), 30)
		checkWindow(t, chain, track(t, n, listed(t, n)))
	}
}

// ValidUntil is the sampled window's end: a simulator built from the state and kept past the state's age bound quotes
// no fill landing after the last sample, where an aging listing pays less.
func TestTrackerValidUntilIsTheSampledWindowsEnd(t *testing.T) {
	chain := trackedChain()
	chain.listings[1].decayPpmPerS = 100
	chain.listings[1].controls = ctlWord(200, 0, 0, 0)
	n := chain.node(t, 1234)
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	end := uint64(p.Timestamp) + uint64(maxAge/time.Second) + e.Allowance
	assert.Equal(t, end, e.ValidUntil)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	assert.True(t, sim.validAt(end-e.Allowance), "a fill landing at the window's end")
	assert.False(t, sim.validAt(end-e.Allowance+1), "a fill landing after the window's end")
}

// Where the node refuses a block override, a listing whose levels drift or widen is quoted no later than its grace:
// within it the age moves no level.
func TestTrackerWithoutOverridesStopsAtTheGrace(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.controls = ctlWord(200, 0, 0, 5_000)
	n := chain.node(t, 1234)
	n.refuseOverrides = true
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")

	// no widening and no drift: the sampled window's end stands
	l.controls = ctlWord(0, 0, 0, 5_000)
	p = track(t, n, p)
	assert.Equal(t, uint64(p.Timestamp)+uint64(maxAge/time.Second)+subSecondInclusionAllowance, extraOf(t, p).ValidUntil)
}

// A node that takes a block override but answers at the pinned block's own time and number is caught by the block's
// time and number read in each sample, and the listing quoted no later than its grace, as where the node refuses.
func TestTrackerWithANodeIgnoringOverridesStopsAtTheGrace(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.decayPpmPerS = 100
	l.controls = ctlWord(200, 0, 0, 5_000)
	n := chain.node(t, 1234)
	n.ignoreOverrides = true
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")
}

// A node that honours a block override's time but drops its number answers a sample in another block than the one
// the override named, where Base's front reads BaseTime and a younger sub-second: the number read in each sample
// catches it, and the listing is quoted no later than its grace.
func TestTrackerWithANodeDroppingTheNumberStopsAtTheGrace(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.decayPpmPerS = 100
	l.controls = ctlWord(200, 0, 0, 5_000)
	n := chain.node(t, 1234)
	n.ignoreNumber = true
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")
}

// A listing whose levels drift by its depth words' drift field alone (word 0, bits 92-111), with no widening in its
// controls, is quoted no later than its grace on a node that refuses block overrides, as one that widens is.
func TestTrackerWithoutOverridesStopsADepthDriftAloneAtTheGrace(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.decayPpmPerS = 100
	l.controls = ctlWord(0, 0, 0, 5_000)
	for _, k := range []int{0, 2} {
		d := new(big.Int)
		if l.view.Depth[k] != nil {
			d.Set(l.view.Depth[k])
		}
		l.view.Depth[k] = d.Or(d, new(big.Int).Lsh(big.NewInt(50), 92))
	}
	n := chain.node(t, 1234)
	n.refuseOverrides = true
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")
}

// Where the sides' depth expires at different seconds, the state is valid to the sooner: a fill of the side whose
// depth expired first would be refused.
func TestTrackerValidUntilIsTheSoonerSidesEnd(t *testing.T) {
	for _, bidFirst := range []bool{true, false} {
		chain := trackedChain()
		l := chain.listings[1]
		n := chain.node(t, 1234)
		soon := new(big.Int).Lsh(new(big.Int).SetUint64(n.time+6), 60)
		// the base, WETH, is token 0: the bid (depth[0]) prices WETH in, the ask (depth[2]) USDC in
		if bidFirst {
			l.view.Depth[0] = soon
		} else {
			l.view.Depth[2] = soon
		}
		p := track(t, n, listed(t, n))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0])
		require.NotEmpty(t, e.Ladders[1])
		assert.Equal(t, n.time+5, e.ValidUntil, "bid first %v: the sooner side's last second", bidFirst)
	}
}

// printListing is listing 1 of the tracked chain reading a price print (scopes, its fence kind PRINT 0 or SESSION 6),
// its levels aging, the print made 2 s before the head with a max age of maxAgeMs.
func printListing(t *testing.T, scopes uint8, maxAgeMs uint64) (*fakeChain, *node) {
	chain := trackedChain()
	chain.subMs = 400
	l := chain.listings[1]
	l.view.Scopes = scopes
	l.decayPpmPerS = 100
	l.controls = ctlWord(200, 0, 0, 0)
	n := chain.node(t, 1234)
	l.printMs, l.printMaxAgeMs = n.time*1000+400-2_000, maxAgeMs
	return chain, n
}

// printEndSecond is the last second whose last block finds alive a print whose age clock may reach endMs.
func printEndSecond(chain *fakeChain, endMs uint64) uint64 {
	lastBlock := func(at uint64) uint64 { return chain.clockMs0 + (at*1000+999-chain.clockMs0)/200*200 }
	want := endMs/1000 + 1
	for lastBlock(want) > endMs {
		want--
	}
	return want
}

// A SESSION listing reads a price print as a PRINT listing does: it is quoted only as far as the print lives.
func TestTrackerQuotesASessionListingAsFarAsItsPrintLives(t *testing.T) {
	chain, n := printListing(t, 6, 8_000)
	require.Equal(t, uint8(3), (chain.listings[1].view.Scopes>>1)&3, "the fence kind SESSION")
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	l := chain.listings[1]
	assert.Equal(t, printEndSecond(chain, l.printMs+l.printMaxAgeMs), e.ValidUntil)
	checkWindow(t, chain, p)
}

// sessionWord is a session word (record word 13): its closedAfter (bits 0-15, s) and offBase (bits 16-31, 0.1 bp).
func sessionWord(closedAfterS, offBase uint64) *big.Int {
	return new(big.Int).SetUint64(closedAfterS | offBase<<16)
}

// sessionReads is the tracker's reads of listing 1's session word (record word 13) from the front's storage, by
// block tag.
func sessionReads(chain *fakeChain, n *node) (tags []string) {
	slot := recordSlot(1, 13)
	for _, c := range n.calls() {
		if c.to == chain.front && common.BytesToHash(c.data) == slot && len(c.data) == common.HashLength {
			tags = append(tags, c.tag)
		}
	}
	return tags
}

// A SESSION listing with a closed phase (its session word's closedAfter nonzero) is not served. In that phase its band
// eases as the print ages while the word's levels tighten, so a quote can dip below every sample of the window, and
// the refused gap between the phases may be shorter than a second. It is left out with the print fresh (the open
// phase), stale under closedAfter (the gap) and past it (the closed phase), the session word read from the front's
// storage at the state's block; and so is one whose session word cannot be read. closedAfter is 16 bits, 60 to 21,600
// s: the delays 256 and 16,384, whose low 8 and 14 bits are zero, are left out too, so a read of fewer than 15 bits
// would serve them (no delay in that range tells 15 bits from 16).
func TestTrackerLeavesOutASessionListingWithAClosedPhase(t *testing.T) {
	type phase struct {
		name        string
		closedAfter uint64 // s
		ageMs       uint64 // the print's age at the state's block
		refuse      bool
	}
	cases := []phase{{"session word unread", 60, 2_000, true}}
	for _, closedAfter := range []uint64{60, 256, 16_384} {
		cases = append(cases,
			phase{fmt.Sprintf("closedAfter %d, open phase", closedAfter), closedAfter, 2_000, false},
			phase{fmt.Sprintf("closedAfter %d, gap", closedAfter), closedAfter, closedAfter * 500, false},
			phase{fmt.Sprintf("closedAfter %d, closed phase", closedAfter), closedAfter, closedAfter*1000 + 10_000,
				false})
	}
	for _, c := range cases {
		chain, n := printListing(t, 6, 8_000)
		l := chain.listings[1]
		l.printMs = n.time*1000 + chain.subMs - c.ageMs
		l.session = sessionWord(c.closedAfter, 2_000)
		if c.refuse {
			n.refuseSlot = recordSlot(1, 13)
		}
		p := track(t, n, track(t, n, listed(t, n)))
		e := extraOf(t, p)
		assert.Empty(t, e.Ladders[0], c.name)
		assert.Empty(t, e.Ladders[1], c.name)
		assert.Zero(t, e.ValidUntil, c.name)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves, c.name)
		tags := sessionReads(chain, n)
		assert.NotEmpty(t, tags, "%s: the session word is read", c.name)
		for _, tag := range tags {
			assert.Equal(t, hexutil.EncodeUint64(p.BlockNumber), tag, "%s: at the state's block", c.name)
		}
	}
}

// A SESSION listing with no closed phase (closedAfter 0, whatever its other closed-phase settings) is served as a
// PRINT listing is: quoted as far as its print lives.
func TestTrackerServesASessionListingWithNoClosedPhase(t *testing.T) {
	for _, w := range []*big.Int{nil, sessionWord(0, 2_000)} {
		chain, n := printListing(t, 6, 8_000)
		l := chain.listings[1]
		l.session = w
		p := track(t, n, track(t, n, listed(t, n)))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0], "session word %v", w)
		require.NotEmpty(t, e.Ladders[1], "session word %v", w)
		assert.Equal(t, printEndSecond(chain, l.printMs+l.printMaxAgeMs), e.ValidUntil, "session word %v", w)
		checkWindow(t, chain, p)
	}
}

// A PRINT listing is unchanged by a session word left in its record, which the front reads only for a SESSION
// listing: the word is not read, the listing is quoted as far as its print lives, and past it not at all.
func TestTrackerQuotesAPrintListingWhateverItsSessionWord(t *testing.T) {
	chain, n := printListing(t, 0, 8_000)
	l := chain.listings[1]
	l.session = sessionWord(60, 2_000)
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	require.NotEmpty(t, e.Ladders[1])
	assert.Equal(t, printEndSecond(chain, l.printMs+l.printMaxAgeMs), e.ValidUntil)
	checkWindow(t, chain, p)
	assert.Empty(t, sessionReads(chain, n), "no session word read for a PRINT listing")

	l.printMs = n.time*1000 + chain.subMs - 70_000
	p = track(t, n, p)
	e = extraOf(t, p)
	assert.Empty(t, e.Ladders[0], "a print past its life")
	assert.Empty(t, e.Ladders[1], "a print past its life")
}

// A print listing's run ends at its first refusal: a door that refuses in one second and fills again later is not
// quoted past the second before the refusal.
func TestTrackerEndsAPrintRunAtItsFirstRefusal(t *testing.T) {
	chain, n := printListing(t, 0, 30_000)
	l := chain.listings[1]
	l.gapFromMs, l.gapToMs = (n.time+3)*1000, (n.time+4)*1000
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, n.time+2, e.ValidUntil, "the last second before the gap")
}

// Where some of a print listing's probes cannot be read (the node refuses the later overrides), its print's life is
// not known, and it is quoted no later than the block's own second, though its earlier probes filled.
func TestTrackerQuotesAPrintListingWhoseLifeIsUnreadNoLaterThanItsBlock(t *testing.T) {
	_, n := printListing(t, 0, 8_000)
	n.refuseOverridesAfter = n.time + 3
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, uint64(p.Timestamp), e.ValidUntil)
}

// A premium that runs past the window is sampled only up to the window's end: the ladder pays what the door pays in
// the window's last block, which a fill quoted from the state may land in, not what it pays after it.
func TestTrackerSamplesNoPremiumPastTheWindow(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.decayPpmPerS = 100
	l.controls = ctlWord(200, 2_000, 20_000, 0)
	n := chain.node(t, 1234)
	l.view.Word = testWord(int64(n.time*1000+chain.subMs), 30)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	fills := chain.fills(p, e.ValidUntil)
	last := fills[len(fills)-1]
	for dir := range 2 {
		for _, pt := range e.Ladders[dir] {
			door, err := chain.quoteAtSub(l, dir, pt[0].ToBig(), last.at, last.number, last.sub)
			require.NoError(t, err)
			assert.Equal(t, door.String(), pt[1].ToBig().String(), "direction %d at %s: the ladder against the "+
				"door in the window's last block", dir, pt[0])
		}
	}
	checkWindow(t, chain, p)
}

// A listing that reads a price print is quoted only as far as that print lives on the sampled state: the window ends
// at the last second the door still fills in, its end sampled there, and the state is valid to it, no later. No
// later print is assumed.
func TestTrackerQuotesAPrintListingAsFarAsItsPrintLives(t *testing.T) {
	for _, sub := range []uint64{0, 400, 800} {
		chain := trackedChain()
		chain.subMs = sub
		l := chain.listings[1]
		l.view.Scopes = 0 // the fence kind PRINT
		l.decayPpmPerS = 100
		l.controls = ctlWord(200, 0, 0, 0)
		n := chain.node(t, 1234)
		head := n.time*1000 + sub
		// a print made 2 s before the state's block, its max age 8 s: it lives to 6 s after the block
		l.printMs, l.printMaxAgeMs = head-2_000, 8_000
		p := track(t, n, track(t, n, listed(t, n)))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0], "sub-second %d", sub)
		require.NotEmpty(t, e.Ladders[1], "sub-second %d", sub)
		// the last second whose last block still finds the print alive
		printEnd := l.printMs + l.printMaxAgeMs
		lastBlock := func(at uint64) uint64 { return chain.clockMs0 + (at*1000+999-chain.clockMs0)/200*200 }
		want := printEnd/1000 + 1
		for lastBlock(want) > printEnd {
			want--
		}
		assert.Equal(t, want, e.ValidUntil, "sub-second %d", sub)
		checkWindow(t, chain, p)

		// a print that dies in the block's own second, less than the inclusion allowance after it: not quoted
		l.printMs = head + 100 - l.printMaxAgeMs
		p = track(t, n, p)
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
		assert.ErrorIs(t, err, ladder.ErrStale, "sub-second %d", sub)
	}

	// where the print's life cannot be read (the node refuses overrides), no later than the block's own second
	chain := trackedChain()
	l := chain.listings[1]
	l.view.Scopes = 0
	n := chain.node(t, 1234)
	l.printMs, l.printMaxAgeMs = n.time*1000, 8_000
	n.refuseOverrides = true
	p := track(t, n, listed(t, n))
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	assert.Equal(t, uint64(p.Timestamp), extraOf(t, p).ValidUntil)
}

// On a clock whose sub-second no block override sets (BaseTime's, mode 1), a listing whose levels drift or widen is
// quoted no later than its grace, as where the node refuses overrides.
func TestTrackerOnABaseTimeClockStopsAtTheGrace(t *testing.T) {
	chain := trackedChain()
	chain.clockMode = 1
	l := chain.listings[1]
	l.controls = ctlWord(200, 0, 0, 5_000)
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")
	for _, c := range n.calls() {
		if c.at != 0 { // an eth_call's
			assert.Equal(t, [2]uint64{n.time, n.block}, [2]uint64{c.at, c.number}, "no block override")
		}
	}
}

// place finds the block in which the front's clock reads the latest moment no later than a given one: on Base's
// block-count clock the 200 ms block by its number, on a seconds clock the second; on any other clock none.
// moment reads a block's clock as the front does: the anchor's milliseconds where they fall in the block's second,
// else BaseTime's where they do, else the second's start; a seconds clock the second's start.
func TestClockMoment(t *testing.T) {
	u := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
	c := frontClock{ClockMode: u(clockBlocks), ClockBlock: u(1000), ClockMs: u(5_000_000)}
	assert.Equal(t, uint64(5_000_400), c.moment(5000, 1002, nil), "the anchor places the block")
	assert.Equal(t, uint64(5_000_400), c.moment(5000, 1002, u(5_000_900)), "the anchor before BaseTime")
	assert.Equal(t, uint64(5_001_700), c.moment(5001, 1002, u(5_001_700)), "BaseTime where the anchor does not")
	assert.Equal(t, uint64(5_001_000), c.moment(5001, 1002, u(5_002_100)), "BaseTime in another second")
	assert.Equal(t, uint64(5_001_000), c.moment(5001, 1002, nil), "no BaseTime")
	assert.Equal(t, uint64(4_500), c.moment(4, 999, u(4_500)), "BaseTime for a block before the anchor")
	c.ClockMode = u(clockSeconds)
	assert.Equal(t, uint64(5_000_000), c.moment(5000, 1002, u(5_000_400)), "a seconds clock")
}

func TestClockPlace(t *testing.T) {
	u := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
	blocks := frontClock{ClockMode: u(2), ClockBlock: u(1000), ClockMs: u(1_700_000_000_400)}
	o, ok := blocks.place(1_700_000_013_999)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_013, number: 1000 + 67}, o, "13.999 s on: the block at 13.800")
	o, ok = blocks.place(1_700_000_002_399)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_002, number: 1000 + 9}, o, "the block at 2.200, before 2.400")
	_, ok = blocks.place(1_700_000_000_399)
	assert.False(t, ok, "before the anchor")

	seconds := frontClock{ClockMode: u(0), ClockBlock: u(0), ClockMs: u(0)}
	o, ok = seconds.place(1_700_000_013_999)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_013}, o)

	for _, mode := range []uint64{1, 3} {
		_, ok = frontClock{ClockMode: u(mode), ClockBlock: u(1000), ClockMs: u(1_700_000_000_400)}.place(1_700_000_013_999)
		assert.False(t, ok, "mode %d", mode)
	}

	// Pinned on BaseTime with the count a second ahead of it: block 1100 counts 20.450, BaseTime reads 19.450. The
	// block place picks for 20.040, two 200 ms steps on (19.850), counts 20.850, in 20.040's own second, so the front
	// reads its count there and no block reads 20.040: place refuses it. 20.300's block counts 21.250, another second.
	ahead := frontClock{ClockMode: u(2), ClockBlock: u(1000), ClockMs: u(1_700_000_000_450)}
	require.False(t, ahead.pin(1_700_000_019, 1100, u(1_700_000_019_450)))
	_, ok = ahead.place(1_700_000_020_040)
	assert.False(t, ok, "the count falls in the moment's second")
	o, ok = ahead.place(1_700_000_020_300)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_020, number: 1104, baseTimeMs: 1_700_000_020_300}, o)
}

// pin reads the front's block-count clock at the state's block as PropAMMClock.read does, and tells how place puts
// later moments: on the count where the count falls in the block's second and BaseTime agrees with it or is not read
// there; on BaseTime where the count is a second or more off BaseTime's reading in the block's second, at the moment
// itself, in the chain's block by 200 ms from that reading, BaseTime's stored sub-second set to the moment's; nowhere
// where neither falls in the block's second. Where the count is off BaseTime by less than a second, the clock may step
// back.
func TestClockPin(t *testing.T) {
	u := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
	// block 1100 counts 1_700_000_020_400 from its anchor
	clock := func() frontClock {
		return frontClock{ClockMode: u(2), ClockBlock: u(1000), ClockMs: u(1_700_000_000_400)}
	}

	onBaseTime := clock()
	assert.False(t, onBaseTime.pin(1_700_000_025, 1100, u(1_700_000_025_600)), "5.2 s off")
	o, ok := onBaseTime.place(1_700_000_030_999)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_030, number: 1100 + 26, baseTimeMs: 1_700_000_030_999}, o,
		"the last moment of second 30, in the block 26 x 200 ms after BaseTime's reading")
	o, ok = onBaseTime.place(1_700_000_025_600)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_025, number: 1100, baseTimeMs: 1_700_000_025_600}, o)
	_, ok = onBaseTime.place(1_700_000_025_599)
	assert.False(t, ok, "before the state's moment")

	for _, c := range []struct {
		sec, baseTimeMs uint64
		steps           bool
	}{
		{1_700_000_020, 1_700_000_020_400, false}, // on the count
		{1_700_000_020, 1_700_000_020_600, true},  // 200 ms ahead of it
		{1_700_000_021, 1_700_000_021_399, true},  // 999 ms ahead, the count in another second
		{1_700_000_019, 1_700_000_019_401, true},  // 999 ms behind
		{1_700_000_021, 1_700_000_021_400, false}, // a second ahead
		{1_700_000_019, 1_700_000_019_400, false}, // a second behind
	} {
		k := clock()
		assert.Equal(t, c.steps, k.pin(c.sec, 1100, u(c.baseTimeMs)), "BaseTime %d", c.baseTimeMs)
	}

	counted := clock()
	assert.False(t, counted.pin(1_700_000_020, 1100, nil), "no BaseTime: the count")
	o, ok = counted.place(1_700_000_023_999)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_023, number: 1000 + 117}, o, "on the count, as before")

	neither := clock()
	assert.False(t, neither.pin(1_700_000_025, 1100, nil))
	_, ok = neither.place(1_700_000_030_999)
	assert.False(t, ok, "the clock reads the second's start: no override places a moment")

	seconds := frontClock{ClockMode: u(0), ClockBlock: u(0), ClockMs: u(0)}
	assert.False(t, seconds.pin(1_700_000_025, 1100, u(1_700_000_025_600)), "a seconds clock never steps back")
	o, ok = seconds.place(1_700_000_030_999)
	assert.True(t, ok)
	assert.Equal(t, override{time: 1_700_000_030}, o)
}

// The door's quoteFillable finds each side's top, so the depth above the last sampled size is quoted up to it.
func TestTrackerSamplesEachSidesTop(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.sides[0].rungs = append(l.sides[0].rungs, fakeRung{7e18, 2980, 1e12}) // top 7 WETH: no power of two of the grid
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	p = track(t, n, p)
	ladder0 := extraOf(t, p).Ladders[0]
	top := ladder0[len(ladder0)-1]
	assert.Equal(t, uint64(7e18), top[0].Uint64())
	want, err := l.sides[0].quote(big.NewInt(7e18))
	require.NoError(t, err)
	assert.Equal(t, want, top[1].ToBig())
}

// A listing whose doorView names another pool than the one listed, or another pair, is not quoted: its ladder would
// not be the listed custody's.
func TestTrackerSkipsAListingWhosePoolOrPairMoved(t *testing.T) {
	for name, move := range map[string]func(v *DoorView){
		"pool": func(v *DoorView) { v.Pool = testPoolB },
		"pair": func(v *DoorView) { v.Token1 = testPoolB },
	} {
		chain := trackedChain()
		n := chain.node(t, 1234)
		p := listed(t, n)
		move(&chain.listings[1].view)
		p = track(t, n, p)
		e := extraOf(t, p)
		assert.Empty(t, e.Ladders[0], name)
		assert.Empty(t, e.Ladders[1], name)
	}
}

func TestCollectRun(t *testing.T) {
	sizes := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)}
	outs := []*big.Int{big.NewInt(0), big.NewInt(20), big.NewInt(30), big.NewInt(0), big.NewInt(50), big.NewInt(60)}

	// under the minimum, then a run, then a refusal: the run ends there, though a larger size passes again
	run := collectRun(sizes, outs, []bool{false, true, true, false, true, true})
	assert.Equal(t, []Point{
		{uint256.NewInt(2), uint256.NewInt(20)}, {uint256.NewInt(3), uint256.NewInt(30)},
	}, run)

	// a falling output ends the run
	outs[3] = big.NewInt(25)
	run = collectRun(sizes, outs, []bool{false, true, true, true, true, true})
	assert.Len(t, run, 2)

	// a success that paid nothing is no point
	run = collectRun(sizes[:1], []*big.Int{big.NewInt(0)}, []bool{true})
	assert.Empty(t, run)
}

// A front that does not answer clock() gives no word's end: the refresh fails rather than quote without a bound.
func TestTrackerNeedsTheFrontsClock(t *testing.T) {
	chain := trackedChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	chain.noClock = true
	tr, err := NewPoolTracker(testConfig(), n.client())
	require.NoError(t, err)
	_, err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestNewPoolTrackerValidates(t *testing.T) {
	_, err := NewPoolTracker(&Config{}, nil)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

// trackWith is track under cfg.
func trackWith(t *testing.T, n *node, p entity.Pool, cfg *Config) entity.Pool {
	t.Helper()
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	return got
}

// approving is the test config approving the sources srcs on its front.
func approving(srcs ...common.Address) *Config {
	cfg := testConfig()
	cfg.Fronts[0].ApprovedSources = nil
	for _, s := range srcs {
		cfg.Fronts[0].ApprovedSources = append(cfg.Fronts[0].ApprovedSources, hexutil.Encode(s[:]))
	}
	return cfg
}

// storageReads is the tracker's reads of the front's storage slot, by block tag.
func storageReads(chain *fakeChain, n *node, slot common.Hash) (tags []string) {
	for _, c := range n.calls() {
		if c.to == chain.front && len(c.data) == common.HashLength && common.BytesToHash(c.data) == slot {
			tags = append(tags, c.tag)
		}
	}
	return tags
}

// unserved checks that a state quotes nothing: no ladder, no valid-until, no reserves.
func unserved(t *testing.T, p entity.Pool, msg string) {
	t.Helper()
	e := extraOf(t, p)
	assert.Empty(t, e.Ladders[0], msg)
	assert.Empty(t, e.Ladders[1], msg)
	assert.Zero(t, e.ValidUntil, msg)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves, msg)
}

// A listing that reads a price print (PRINT or SESSION; a tied listing is never served) is served only where every
// source it is bound to, its primary and its backup where it has one (record word 10), is a source the front's config
// approves, each read at the state's block: a source anyone may register answers the front's call with whatever it
// likes, so a quote need not be what a fill pays. The approved list is empty by default, and then no such listing is
// quoted nor its binding read. Source ids are 32 bits.
func TestTrackerServesAPrintListingOnlyOnApprovedSources(t *testing.T) {
	both := new(big.Int).SetUint64(1 | 3<<32)
	reversed := new(big.Int).SetUint64(3 | 1<<32)
	for _, c := range []struct {
		name     string
		scopes   uint8
		ids      *big.Int
		approved []common.Address
		served   bool
	}{
		{"PRINT, its source approved", 0, big.NewInt(1), []common.Address{testSource}, true},
		{"SESSION, its source approved", 6, big.NewInt(1), []common.Address{testSource}, true},
		{"PRINT, none approved", 0, big.NewInt(1), nil, false},
		{"SESSION, none approved", 6, big.NewInt(1), nil, false},
		{"tied, none approved", 16, big.NewInt(1), nil, false},
		{"its source another", 0, big.NewInt(2), []common.Address{testSource}, false},
		{"its backup not approved", 0, both, []common.Address{testSource}, false},
		{"its primary not approved", 0, reversed, []common.Address{testSource}, false},
		{"both approved", 0, both, []common.Address{testThird, testSource}, true},
		{"no source bound", 0, nil, []common.Address{testSource}, false},
		{"a source never registered", 0, big.NewInt(7), []common.Address{testSource}, false},
		// ids are 32 bits, and anyone may register sources until one passes 16: 65,537's low 16 bits name source 1
		{"an id past 16 bits, not approved", 0, big.NewInt(65_537), []common.Address{testSource}, false},
		{"an id past 16 bits, approved", 0, big.NewInt(65_538), []common.Address{testSource}, true},
		{"a backup id past 16 bits, not approved", 0, new(big.Int).SetUint64(1 | 65_537<<32),
			[]common.Address{testSource}, false},
	} {
		chain, n := printListing(t, c.scopes, 8_000)
		l := chain.listings[1]
		l.sourceIds = c.ids
		cfg := approving(c.approved...)
		p := listed(t, n)
		before := len(n.calls())
		p = trackWith(t, n, trackWith(t, n, p, cfg), cfg)
		if c.served {
			e := extraOf(t, p)
			assert.NotEmpty(t, e.Ladders[0], c.name)
			assert.NotEmpty(t, e.Ladders[1], c.name)
		} else {
			unserved(t, p, c.name)
		}
		reads := storageReads(chain, n, recordSlot(1, 10))
		if len(c.approved) == 0 {
			assert.Empty(t, reads, "%s: no binding read with no source approved", c.name)
			for _, call := range n.calls()[before:] {
				assert.NotEqual(t, "0xb6466384", hexutil.Encode(call.data[:min(4, len(call.data))]),
					"%s: no quote", c.name)
			}
			continue
		}
		assert.NotEmpty(t, reads, "%s: the binding is read", c.name)
		for _, tag := range reads {
			assert.Equal(t, hexutil.EncodeUint64(p.BlockNumber), tag, "%s: at the state's block", c.name)
		}
		if c.ids != nil {
			for _, tag := range storageReads(chain, n, sourceEntrySlot(uint32(c.ids.Uint64()&0xffffffff))) {
				assert.Equal(t, hexutil.EncodeUint64(p.BlockNumber), tag, "%s: the entry at the state's block", c.name)
			}
		}
	}
}

// An operator may rebind a listing's sources at any time: every refresh reads the binding again, so a listing rebound
// to a source not approved is no longer quoted, and one rebound to an approved source is quoted again.
func TestTrackerReadsTheBindingOnEveryRefresh(t *testing.T) {
	chain, n := printListing(t, 0, 8_000)
	l := chain.listings[1]
	p := track(t, n, listed(t, n))
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	l.sourceIds = big.NewInt(2)
	p = track(t, n, p)
	unserved(t, p, "rebound to a source not approved")
	l.sourceIds = big.NewInt(1)
	p = track(t, n, p)
	assert.NotEmpty(t, extraOf(t, p).Ladders[0], "rebound to the approved source")
}

// A binding the node does not answer is not known to be approved: the listing is not quoted.
func TestTrackerLeavesOutAPrintListingWhoseBindingIsUnread(t *testing.T) {
	for _, slot := range []common.Hash{recordSlot(1, 10), sourceEntrySlot(1)} {
		chain, n := printListing(t, 0, 8_000)
		_ = chain
		n.refuseSlot = slot
		unserved(t, track(t, n, listed(t, n)), "unread "+slot.Hex())
	}
}

// A listing that reads no print (WORD or NONE) is served whatever the approved list, its binding not read.
func TestTrackerServesAWordListingWithNoSourceApproved(t *testing.T) {
	for _, scopes := range []uint8{2, 4} {
		chain := trackedChain()
		chain.listings[1].view.Scopes = scopes
		chain.listings[1].sourceIds = big.NewInt(2)
		n := chain.node(t, 1234)
		p := trackWith(t, n, listed(t, n), approving())
		assert.NotEmpty(t, extraOf(t, p).Ladders[0], "scopes %d", scopes)
		assert.NotEmpty(t, extraOf(t, p).Ladders[1], "scopes %d", scopes)
		assert.Empty(t, storageReads(chain, n, recordSlot(1, 10)), "scopes %d: no binding read", scopes)
	}
}

// A listing bound to a guarded entry of an approved source is served, the guard read from the entry the listing is
// bound to, the primary's and the backup's, at the state's block, and marked for its gas where the source it reads now
// is the guarded one. A tied listing is not served, though every source it is bound to is approved: its age runs from
// its print.
func TestTrackerServesAGuardedListingAndLeavesOutATiedOne(t *testing.T) {
	for _, c := range []struct {
		name    string
		scopes  uint8
		ids     uint64
		served  bool
		guarded bool
	}{
		{"PRINT on a guarded entry", 0, 4, true, true},
		{"PRINT whose backup is a guarded entry", 0, 1 | 4<<32, true, false},
		{"PRINT whose primary is a guarded entry", 0, 4 | 1<<32, true, true},
		{"SESSION with no closed phase on a guarded entry", 6, 4, true, true},
		{"PRINT on a guarded entry of a source not approved", 0, 2, false, false},
		{"tied on a guarded entry", 16, 4, false, false},
		{"tied on an entry not guarded", 16, 1, false, false},
		{"PRINT on an entry not guarded", 0, 1, true, false},
		{"SESSION with no closed phase on an entry not guarded", 6, 1, true, false},
	} {
		chain, n := printListing(t, c.scopes, 8_000)
		chain.listings[1].sourceIds = new(big.Int).SetUint64(c.ids)
		chain.holdMs = map[common.Address]int64{testSource: 400}
		p := track(t, n, track(t, n, listed(t, n)))
		if !c.served {
			unserved(t, p, c.name)
			continue
		}
		e := extraOf(t, p)
		assert.NotEmpty(t, e.Ladders[0], c.name)
		assert.NotEmpty(t, e.Ladders[1], c.name)
		assert.True(t, e.Print, c.name)
		assert.Equal(t, c.guarded, e.Guarded, "%s: charged the guarded gas", c.name)
		checkWindow(t, chain, p)
	}
}

// The gas class follows the source the listing reads now, its record word 2's guard (bit 247), read at the state's
// block where either bound entry is guarded: a fill reads that word alone. The hold is sampled wherever either bound
// entry is guarded, since a darken or a promote moves the listing between them. Where the word is not read, either
// entry's guard sets the class.
func TestTrackerChargesTheLiveSourcesGasClass(t *testing.T) {
	for _, c := range []struct {
		name     string
		ids      uint64
		price    func(chain *fakeChain) *big.Int // nil: the primary's
		refused  bool                            // the node refuses the word's read
		guarded  bool
		holdRead bool
	}{
		{"a guarded primary", 4, nil, false, true, true},
		{"an unguarded primary, a guarded backup", 1 | 4<<32, nil, false, false, true},
		{"a guarded primary, an unguarded backup", 4 | 1<<32, nil, false, true, true},
		{"on its guarded backup", 1 | 4<<32, func(c *fakeChain) *big.Int { return liveOf(c.sources[4], true) },
			false, true, true},
		{"a guarded primary darkened onto its unguarded backup", 4 | 1<<32,
			func(c *fakeChain) *big.Int { return liveOf(c.sources[1], true) }, false, false, true},
		{"an unguarded primary, a guarded backup, the word unread", 1 | 4<<32, nil, true, true, true},
		{"an unguarded primary", 1, nil, false, false, false},
	} {
		chain, n := printListing(t, 0, 8_000)
		l := chain.listings[1]
		l.sourceIds = new(big.Int).SetUint64(c.ids)
		chain.holdMs = map[common.Address]int64{testSource: 400}
		if c.price != nil {
			l.price = c.price(chain)
		}
		if c.refused {
			n.refuseSlot = recordSlot(1, 2)
		}
		p := track(t, n, track(t, n, listed(t, n)))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0], c.name)
		require.NotEmpty(t, e.Ladders[1], c.name)
		assert.Equal(t, c.guarded, e.Guarded, "%s: charged the guarded gas", c.name)
		reads := storageReads(chain, n, recordSlot(1, 2))
		if c.holdRead {
			assert.NotEmpty(t, reads, "%s: the live word is read", c.name)
		} else {
			assert.Empty(t, reads, "%s: bound to no guarded entry, the live word is not read", c.name)
		}
		for _, tag := range reads {
			assert.Equal(t, hexutil.EncodeUint64(p.BlockNumber), tag, "%s: at the state's block", c.name)
		}
		sel := hexutil.Encode(laneABI.Methods["HOLD_MS"].ID)
		held := false
		for _, call := range n.calls() {
			held = held || call.to == testSource && len(call.data) >= 4 && hexutil.Encode(call.data[:4]) == sel
		}
		assert.Equal(t, c.holdRead, held, "%s: the hold sampled", c.name)
		checkWindow(t, chain, p)
	}
	// a listing that reads no source reads no live word
	chain := trackedChain()
	n := chain.node(t, 1234)
	chain.listings[1].view.Scopes = 2
	track(t, n, listed(t, n))
	assert.Empty(t, storageReads(chain, n, recordSlot(1, 2)), "a WORD listing")
}

// guardedListing is printListing bound to the approved source's guarded entry (source 4), whose lane answers HOLD_MS
// holdMs (0: the lane does not answer).
func guardedListing(t *testing.T, holdMs int64) (*fakeChain, *node) {
	chain, n := printListing(t, 0, 8_000)
	chain.listings[1].sourceIds = big.NewInt(4)
	if holdMs != 0 {
		chain.holdMs = map[common.Address]int64{testSource: holdMs}
	}
	return chain, n
}

// A guarded source prices a fill's side at the worst print its lane still holds, and once the hold ends the side is
// priced on the kept print alone, so it pays more. A side bound by its output pays its top's bound inside the hold and
// refuses the top once the hold ends, and fills it again only as the age's drift catches up. So every size is sampled
// at every block from the state's to the first at or after the hold's end (the lane's HOLD_MS, read from the lane at
// the state's block; 2,000 ms where it cannot be read), the least output kept and a size refused at any of them
// dropped: the ladder holds in every block of the window, either side held, whenever the post that started the hold.
func TestTrackerSamplesEveryBlockOfAHold(t *testing.T) {
	for _, c := range []struct {
		name            string
		holdMs, heldFor int64 // the lane's HOLD_MS (0: not read), the hold's end after the state's moment
		dir             int
		outCap          int64
	}{
		{"a post in the state's block, the sell side held", 400, 400, 0, 5_000e6},
		{"a post in the state's block, the buy side held", 400, 400, 1, 1e18},
		{"a post a block before the state's", 400, 200, 0, 5_000e6},
		{"a 1 s hold", 1_000, 1_000, 0, 5_000e6},
		{"a lane that does not answer: 2 s", 0, 1_800, 0, 5_000e6},
	} {
		chain, n := guardedListing(t, c.holdMs)
		l := chain.listings[1]
		head := n.time*1000 + chain.subMs
		l.heldUntilMs, l.heldCutPpm[c.dir] = head+uint64(c.heldFor), 300
		l.outCap[c.dir], l.fillableUnderCap = c.outCap, true
		p := track(t, n, listed(t, n))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[c.dir], c.name)
		require.True(t, e.Guarded, c.name)
		// the top the door fills at the state's block is refused once the hold ends
		tokens := [2]common.Address{testWETH, testUSDC}
		data, err := doorABI.Pack("quoteFillable", tokens[c.dir], tokens[1-c.dir], new(big.Int).Lsh(big.NewInt(1), 100))
		require.NoError(t, err)
		ok, ret := chain.respond(fakeDoorOf(chain.factory, 1), data, n.time, 1234, -1)
		require.True(t, ok, c.name)
		vals, err := doorABI.Methods["quoteFillable"].Outputs.Unpack(ret)
		require.NoError(t, err)
		fillable := vals[0].(*big.Int)
		after := head + uint64(c.heldFor)
		_, err = chain.quoteAt(l, c.dir, fillable, after/1000, 1234+uint64(c.heldFor)/200)
		require.Error(t, err, "%s: the top is refused once the hold ends", c.name)
		for _, pt := range e.Ladders[c.dir] {
			assert.Less(t, pt[0].ToBig().Cmp(fillable), 0, "%s: the top is no point", c.name)
		}
		checkWindow(t, chain, p)
		// the lane's hold is read at the state's block
		sel := hexutil.Encode(laneABI.Methods["HOLD_MS"].ID)
		var reads []string
		for _, call := range n.calls() {
			if call.to == testSource && len(call.data) >= 4 && hexutil.Encode(call.data[:4]) == sel {
				reads = append(reads, call.tag)
			}
		}
		assert.NotEmpty(t, reads, c.name)
		for _, tag := range reads {
			assert.Equal(t, hexutil.EncodeUint64(1234), tag, "%s: HOLD_MS at the state's block", c.name)
		}
	}
}

// Where the node refuses block overrides, or takes them and ignores them, no block of a hold is sampled and a guarded
// listing is not quoted, nor where it refuses the hold's first block alone. The same listing on an entry not guarded
// keeps its ladder, quoted no later than its bounds.
func TestTrackerWithoutOverridesLeavesOutAGuardedListing(t *testing.T) {
	for _, how := range []string{"refuses", "ignores", "refuses the hold's first block"} {
		chain, n := guardedListing(t, 400)
		n.refuseOverrides, n.ignoreOverrides = how == "refuses", how == "ignores"
		if how == "refuses the hold's first block" {
			n.refuseNumber = 1235 // 200 ms after the state's block, in its second: no print-life probe asks it
		}
		unserved(t, track(t, n, track(t, n, listed(t, n))), "a node that "+how+" overrides")
		chain.listings[1].sourceIds = big.NewInt(1)
		assert.NotEmpty(t, extraOf(t, track(t, n, listed(t, n))).Ladders[0], "unguarded, a node that %s overrides", how)
	}
}

// On a seconds clock every second from the state's to the first at or after the hold's end is sampled: a guarded
// listing whose lane holds for 2,000 ms, the hold ending one or two seconds after the state's, keeps its ladder, the
// top it fills inside the hold dropped where the hold's end refuses it, and the ladder holds in every second of its
// window.
func TestTrackerSamplesAHoldOnASecondsClock(t *testing.T) {
	for _, heldFor := range []uint64{1_000, 2_000} {
		chain := trackedChain()
		chain.clockMode = 0
		l := chain.listings[1]
		l.view.Scopes = 0
		l.sourceIds = big.NewInt(4)
		chain.holdMs = map[common.Address]int64{testSource: 2_000}
		l.decayPpmPerS = 20
		l.controls = ctlWord(200, 0, 0, 0)
		n := chain.node(t, 1234)
		l.view.Word = testWord(int64(n.time*1000), 30)
		l.printMs, l.printMaxAgeMs = n.time*1000-1_000, 20_000
		l.heldUntilMs, l.heldCutPpm[0] = n.time*1000+heldFor, 300
		l.outCap[0], l.fillableUnderCap = 5_000e6, true
		p := track(t, n, listed(t, n))
		e := extraOf(t, p)
		require.True(t, e.Guarded)
		require.NotEmpty(t, e.Ladders[0], "held for %d ms", heldFor)
		require.Greater(t, e.ValidUntil, uint64(p.Timestamp)+2, "held for %d ms: the window runs past the hold",
			heldFor)
		checkWindow(t, chain, p)
	}
}

// A hold that ends in the window's last second is sampled there too: the print lives to the second after the state's,
// the hold ends 400 ms after the state's moment, and the top the door fills inside the hold is refused only in the
// first block after it, the side's drift bringing it back under its bound later in that second.
func TestTrackerSamplesAHoldEndingInTheWindowsLastSecond(t *testing.T) {
	chain := trackedChain()
	chain.subMs = 800
	l := chain.listings[1]
	l.view.Scopes = 0
	l.sourceIds = big.NewInt(4)
	chain.holdMs = map[common.Address]int64{testSource: 400}
	l.decayPpmPerS = 600
	l.controls = ctlWord(200, 0, 0, 0)
	n := chain.node(t, 1234)
	head := n.time*1000 + chain.subMs
	l.view.Word = testWord(int64(head), 30)
	// the print dies after T+1.900: the last block of T+1 (T+1.800) fills, T+2's does not
	l.printMs, l.printMaxAgeMs = head-1_000, 1_000+1_100
	l.heldUntilMs, l.heldCutPpm[0] = head+400, 300
	l.outCap[0], l.fillableUnderCap = 5_000e6, true
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.True(t, e.Guarded)
	require.NotEmpty(t, e.Ladders[0])
	require.Equal(t, uint64(p.Timestamp)+1, e.ValidUntil, "the print lives to the next second")
	checkWindow(t, chain, p)
}

// A hold that ends one block after the state's (a post a block before it) refuses the side's top only in that first
// block after the hold, the drift bringing it under its bound again a block later: that block is sampled, and the top
// is no point.
func TestTrackerDropsATopRefusedOnlyInTheHoldsFirstBlock(t *testing.T) {
	chain, n := guardedListing(t, 400)
	l := chain.listings[1]
	l.decayPpmPerS = 1_000
	head := n.time*1000 + chain.subMs
	l.view.Word = testWord(int64(head), 30)
	l.heldUntilMs, l.heldCutPpm[0] = head+200, 300
	l.outCap[0], l.fillableUnderCap = 5_000e6, true
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	checkWindow(t, chain, p)
}

// A listing bound to guarded entries on two lanes whose holds differ samples the longer hold, whichever lane is the
// primary.
func TestTrackerSamplesTheLongerHoldOfTwoLanes(t *testing.T) {
	for _, ids := range []uint64{4 | 5<<32, 5 | 4<<32} {
		chain, n := guardedListing(t, 400)
		chain.sources[5] = testEntry(testThird, true)
		chain.holdMs[testThird] = 1_000
		l := chain.listings[1]
		l.sourceIds = new(big.Int).SetUint64(ids)
		l.decayPpmPerS = 20
		head := n.time*1000 + chain.subMs
		l.heldUntilMs, l.heldCutPpm[0] = head+1_000, 300
		l.outCap[0], l.fillableUnderCap = 5_000e6, true
		p := trackWith(t, n, listed(t, n), approving(testSource, testThird))
		e := extraOf(t, p)
		require.True(t, e.Guarded)
		require.NotEmpty(t, e.Ladders[0], "ids %x", ids)
		checkWindow(t, chain, p)
	}
}

// Where the count is a second or more off BaseTime, the front reads BaseTime in every block, and a block may store any
// sub-second: a moment inside a hold, up to the last before it ends, is no 200 ms step from the state's, so the blocks
// of a hold cannot all be sampled. A guarded listing is not quoted there, whatever the sub-second of the state's block
// and whether or not a print is held; the same listing on an entry not guarded is quoted and holds for its window.
func TestTrackerOnBaseTimeLeavesOutAGuardedListing(t *testing.T) {
	for _, sub := range []uint64{0, 200, 400, 600, 800} {
		for _, guarded := range []bool{true, false} {
			chain := trackedChain()
			chain.subMs = sub
			l := chain.listings[1]
			l.view.Scopes = 0
			l.sourceIds = big.NewInt(1)
			chain.holdMs = map[common.Address]int64{testSource: 400}
			l.decayPpmPerS = 100
			l.controls = ctlWord(200, 0, 0, 0)
			n := chain.node(t, 1234)
			head := n.time*1000 + chain.subMs
			chain.clockBlock0 = 1234 - 50
			chain.clockMs0 = head - 5_000 - 50*200
			l.view.Word = testWord(int64(head), 30)
			l.printMs, l.printMaxAgeMs = head-1_000, 20_000
			if guarded {
				// the hold cuts more than the whole window's drift, so the window's end does not bound it
				l.sourceIds = big.NewInt(4)
				l.heldUntilMs, l.heldCutPpm = head+400, [2]int64{3_000, 3_000}
			}
			p := track(t, n, track(t, n, listed(t, n)))
			if guarded {
				unserved(t, p, fmt.Sprintf("guarded, the state's block at sub-second %d", sub))
				continue
			}
			e := extraOf(t, p)
			require.NotEmpty(t, e.Ladders[0], "not guarded, sub-second %d", sub)
			require.NotEmpty(t, e.Ladders[1], "not guarded, sub-second %d", sub)
			checkWindow(t, chain, p)
		}
	}
}

// offCount is the tracked chain with an aging listing whose front's count, anchored 50 blocks before the head, is
// offMs off BaseTime's reading at the head (T.400).
func offCount(t *testing.T, offMs int64) (*fakeChain, *node) {
	chain := trackedChain()
	l := chain.listings[1]
	l.decayPpmPerS = 100
	l.controls = ctlWord(200, 0, 0, 0)
	n := chain.node(t, 1234)
	chain.clockBlock0 = 1234 - 50
	chain.clockMs0 = uint64(int64(n.time*1000+chain.subMs)+offMs) - 50*200
	return chain, n
}

// On Base's block-count clock, where the count is off BaseTime by less than a second, the front reads the count in the
// blocks where it falls in the block's second and BaseTime in the others, so its clock may read earlier at a later
// block: no few samples bound what the door pays over the state's window, and the listing is not quoted.
func TestTrackerLeavesOutAListingWhoseClockMayStepBack(t *testing.T) {
	for _, off := range []int64{-999, -400, -200, -1, 1, 200, 600, 999} {
		_, n := offCount(t, off)
		p := track(t, n, track(t, n, listed(t, n)))
		unserved(t, p, fmt.Sprintf("count off by %d ms", off))

		// a PRINT listing on an approved source, its print fresh, is left out too
		chain, n := offCount(t, off)
		l := chain.listings[1]
		l.view.Scopes = 0
		l.printMs, l.printMaxAgeMs = n.time*1000+chain.subMs-1_000, 8_000
		p = track(t, n, track(t, n, listed(t, n)))
		unserved(t, p, fmt.Sprintf("a PRINT listing, count off by %d ms", off))
	}
}

// Where the count is a second or more off BaseTime, the front reads BaseTime in every block: each sample is placed on
// BaseTime, the block override setting the second and BaseTime's stored sub-second set by a state override, the
// window's end at the last moment of its last second (999 ms), which no fill in that second is older than. The ladder
// holds for every fill of the window, on Base's 200 ms blocks and at the last moment of every second.
func TestTrackerOnACountOffBaseTimeSamplesOnBaseTime(t *testing.T) {
	for _, off := range []int64{-1_000, 1_000, -1_100, 1_300, -5_000, 18_200} {
		chain, n := offCount(t, off)
		p := track(t, n, track(t, n, listed(t, n)))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0], "count off by %d ms", off)
		require.NotEmpty(t, e.Ladders[1], "count off by %d ms", off)
		assert.Equal(t, uint64(p.Timestamp)+uint64(maxAge/time.Second)+e.Allowance, e.ValidUntil,
			"count off by %d ms: valid to the sampled window's end", off)
		checkWindow(t, chain, p)
		last := false
		for _, c := range n.calls() {
			if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == "0xb6466384" && c.at == e.ValidUntil {
				last = last || c.sub == 999
				nowMs, _ := chain.clockAtSub(c.at, c.number, c.sub)
				assert.Equal(t, c.at*1000+uint64(c.sub), nowMs, "count off by %d ms: the front reads BaseTime", off)
			}
		}
		assert.True(t, last, "count off by %d ms: the window's end sampled at its last moment", off)
	}
}

// A node that takes the state override of BaseTime's sub-second and ignores it answers at the pinned block's
// sub-second, younger than the moment asked: the BaseTime read in each sample catches it, and the listing is quoted no
// later than its grace.
func TestTrackerOnBaseTimeWithANodeIgnoringStateOverridesStopsAtTheGrace(t *testing.T) {
	chain, n := offCount(t, -5_000)
	l := chain.listings[1]
	l.controls = ctlWord(200, 0, 0, 5_000)
	n.ignoreStateOverrides = true
	p := track(t, n, track(t, n, listed(t, n)))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	makerMs := bits(l.view.Word, 72, 48)
	assert.Equal(t, max(uint64(p.Timestamp), (makerMs+5_000-999)/1000), e.ValidUntil, "the grace's last second")
}

// Where the count is BaseTime's reading at the state's block, the samples are placed on the count, by the block's
// number, with no state override, as before; the ladder holds for its window.
func TestTrackerOnACountOnBaseTimeSamplesOnTheCount(t *testing.T) {
	chain, n := offCount(t, 0)
	p := track(t, n, track(t, n, listed(t, n)))
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	checkWindow(t, chain, p)
	for _, c := range n.calls() {
		assert.Equal(t, int64(-1), c.sub, "no state override")
	}
}

// The chain's millisecond-time contract is the front's config field, required where the front's clock counts blocks:
// a block-count front with none named fails the refresh, and one with another named reads and overrides that one. A
// seconds clock needs none.
func TestTrackerReadsTheConfiguredBaseTime(t *testing.T) {
	none := testConfig()
	none.Fronts[0].BaseTime = ""
	chain := trackedChain()
	n := chain.node(t, 1234)
	tr, err := NewPoolTracker(none, n.client())
	require.NoError(t, err)
	_, err = tr.GetNewPoolState(context.Background(), listed(t, n), pool.GetNewPoolStateParams{})
	assert.ErrorIs(t, err, ErrInvalidConfig, "a block-count front names no BaseTime")

	seconds := trackedChain()
	seconds.clockMode = 0
	n = seconds.node(t, 1234)
	assert.NotEmpty(t, extraOf(t, trackWith(t, n, listed(t, n), none)).Ladders[0], "a seconds clock needs none")

	other := common.HexToAddress("0x420000000000000000000000000000000000003a")
	chain, n = offCount(t, -5_000)
	chain.baseTime, n.baseTime = other, other
	cfg := testConfig()
	cfg.Fronts[0].BaseTime = hexutil.Encode(other[:])
	p := trackWith(t, n, trackWith(t, n, listed(t, n), cfg), cfg)
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	checkWindow(t, chain, p)
	var read, set int
	for _, c := range n.calls() {
		if c.to == other {
			read++
		}
		if c.sub >= 0 {
			set++
		}
	}
	assert.NotZero(t, read, "the configured BaseTime is read")
	assert.NotZero(t, set, "and its sub-second set by the samples' state overrides")
}

// A premium on a listing whose count is a second or more off BaseTime is sampled on BaseTime in its window's last
// moment, wherever in its second the window ends.
func TestTrackerOnBaseTimeLadderHoldsThroughAPremiumWindow(t *testing.T) {
	for _, windowMs := range []int64{1_800, 2_100, 2_550, 6_000} {
		chain, n := offCount(t, -5_000)
		l := chain.listings[1]
		l.controls = ctlWord(200, 2_000, windowMs, 0)
		l.view.Word = testWord(int64(n.time*1000+chain.subMs), 30)
		checkWindow(t, chain, track(t, n, listed(t, n)))
	}
}

// A front is quoted only while its proxy runs an implementation its config lists, read from the proxy's ERC-1967 slot
// at the state's block: the source reads the front's storage by its layout and mirrors its clock and pricing rules, and
// an upgrade may change any of them. An implementation not listed, or a slot the node does not answer, quotes nothing:
// a WORD listing, a PRINT listing and a SESSION listing with no closed phase, each on an approved source.
func TestTrackerQuotesOnlyAListedImplementation(t *testing.T) {
	type kase struct {
		name   string
		impl   common.Address
		refuse bool
		served bool
	}
	var cases []kase
	for _, kind := range []string{"WORD", "PRINT", "SESSION"} {
		cases = append(cases,
			kase{kind + ", the listed implementation", testImpl, false, true},
			kase{kind + ", another implementation", common.HexToAddress("0x6000000000000000000000000000000000000002"),
				false, false},
			kase{kind + ", the slot unread", testImpl, true, false})
	}
	for _, c := range cases {
		chain := trackedChain()
		n := chain.node(t, 1234)
		switch {
		case strings.HasPrefix(c.name, "PRINT"):
			chain, n = printListing(t, 0, 8_000)
		case strings.HasPrefix(c.name, "SESSION"):
			chain, n = printListing(t, 6, 8_000)
		}
		chain.impl = c.impl
		if c.refuse {
			n.refuseSlot = testImplSlot
		}
		p := track(t, n, track(t, n, listed(t, n)))
		if c.served {
			assert.NotEmpty(t, extraOf(t, p).Ladders[0], c.name)
			assert.NotEmpty(t, extraOf(t, p).Ladders[1], c.name)
		} else {
			unserved(t, p, c.name)
		}
		reads := storageReads(chain, n, testImplSlot)
		assert.NotEmpty(t, reads, "%s: the implementation is read", c.name)
		for _, tag := range reads {
			assert.Equal(t, hexutil.EncodeUint64(p.BlockNumber), tag, "%s: at the state's block", c.name)
		}
	}
}

// A listing's gas class follows its place in its pool: on the hook pair (base index 0, quote 1) or past it (any other
// indices, a reversed pair's included), whatever its fence kind.
func TestTrackerMarksAListingPastTheHookPair(t *testing.T) {
	for _, c := range []struct {
		base, quote uint8
		past        bool
	}{{0, 1, false}, {2, 1, true}, {1, 2, true}, {1, 0, true}, {0, 2, true}} {
		chain := trackedChain()
		chain.listings[1].view.BaseIndex, chain.listings[1].view.QuoteIndex = c.base, c.quote
		n := chain.node(t, 1234)
		e := extraOf(t, track(t, n, listed(t, n)))
		require.NotEmpty(t, e.Ladders[0])
		assert.Equal(t, c.past, e.Past, "base index %d, quote index %d", c.base, c.quote)
	}
}

// A side bound by its output (its desk's bucket, its pool's room) pays the most at the first block at or after its
// premium's window closes, the premium lifted and the drift barely begun, where the door refuses a size it filled with
// the premium on and fills again once the drift has lowered the output by the window's end. That block is sampled, and
// a size refused there is no point: the ladder holds in every block of the window, on the count and on BaseTime, with
// the premium ending on a block's moment and between two.
func TestTrackerDropsASizeRefusedOnceAPremiumEnds(t *testing.T) {
	for _, off := range []int64{0, -5_000} {
		for _, sinceMs := range []uint64{0, 100} {
			chain, n := offCount(t, off)
			l := chain.listings[1]
			l.decayPpmPerS = 20_000
			l.controls = ctlWord(0, 300_000, 3_000, 0) // 30 % for 3 s after the post, no widening
			// at most 5,000 USDC out: 2 WETH pays 4,198 at the block, 5,637 as the premium ends and 4,438 at the end
			l.outCap[0] = 5_000e6
			// the word posted sinceMs before the state's block: the premium ends 3 s after the post
			l.view.Word = testWord(int64(n.time*1000+chain.subMs-sinceMs), 30)
			p := track(t, n, listed(t, n))
			e := extraOf(t, p)
			require.NotEmpty(t, e.Ladders[0], "count off by %d ms, posted %d ms before", off, sinceMs)
			// the first block at or after the premium's end: 15 blocks on, by the count or by BaseTime
			pend := n.time*1000 + chain.subMs + 3_000
			_, err := chain.quoteAtSub(l, 0, e.Ladders[0][len(e.Ladders[0])-1][0].ToBig(), pend/1000, 1234+15,
				int64(pend%1000))
			assert.NoError(t, err, "count off by %d ms, posted %d ms before: the ladder's top fills once the premium "+
				"ends", off, sinceMs)
			checkWindow(t, chain, p)
		}
	}
}

// outputBound is the tracked chain on the count (anchored 50 blocks before the head, on BaseTime's reading) with
// listing 1's bid bound to at most 5,000 USDC out, its top the largest prefix within that bound, its levels drifting
// driftPpmPerS ppm/s from the word's maker time and a premium of premiumPpm for windowMs from the premium's start.
func outputBound(t *testing.T, driftPpmPerS, premiumPpm, windowMs int64) (*fakeChain, *node) {
	chain, n := offCount(t, 0)
	l := chain.listings[1]
	l.decayPpmPerS = driftPpmPerS
	l.controls = ctlWord(0, premiumPpm, windowMs, 0)
	l.outCap[0], l.fillableUnderCap = 5_000e6, true
	return chain, n
}

// A side bound by its output whose premium is a little over its drift to the premium's end pays its top's bound at the
// block, over it only in the first block at or after the end (the premium lifted, the drift short of it), and under
// it from the next block on. That block, and no later one, is where the door refuses the top: it is sampled, and the
// top is no point. The premium runs from the latest of the word's maker time, the side's depth nonce and the controls'
// nonce, so a depth or controls re-post after the word restarts it: 2 % a second of drift, a premium of 6.2 % for 3 s
// from the head's moment, the word posted at the head or 700 ms before it with the re-post at the head.
func TestTrackerDropsASizeRefusedOnlyInTheFirstBlockAfterAPremium(t *testing.T) {
	for _, from := range []string{"the word", "the depth nonce", "the controls nonce"} {
		chain, n := outputBound(t, 20_000, 62_000, 3_000)
		l := chain.listings[1]
		head := n.time*1000 + chain.subMs
		l.view.Word = testWord(int64(head), 30)
		// a re-post at the head of a word posted 700 ms earlier: the drift runs from the word, 0.7 s ahead at every block
		switch from {
		case "the depth nonce":
			l.view.Word = testWord(int64(head-700), 30)
			l.view.Depth[0] = new(big.Int).Or(l.view.Depth[0], new(big.Int).Lsh(new(big.Int).SetUint64(head), 12))
		case "the controls nonce":
			l.view.Word = testWord(int64(head-700), 30)
			l.controls = new(big.Int).Or(l.controls, new(big.Int).Lsh(new(big.Int).SetUint64(head), 16))
		}
		p := track(t, n, listed(t, n))
		e := extraOf(t, p)
		require.NotEmpty(t, e.Ladders[0], "premium from %s", from)
		// the top the door fills at the head is refused 15 blocks on, as the premium ends, and filled again a block later
		_, err := chain.quoteAt(l, 0, new(big.Int).SetUint64(5e18), n.time, 1234)
		require.Error(t, err, "premium from %s: the bound binds", from)
		data, err := doorABI.Pack("quoteFillable", testWETH, testUSDC, new(big.Int).SetUint64(5e18))
		require.NoError(t, err)
		ok, ret := chain.respond(fakeDoorOf(chain.factory, 1), data, n.time, 1234, -1)
		require.True(t, ok)
		vals, err := doorABI.Methods["quoteFillable"].Outputs.Unpack(ret)
		require.NoError(t, err)
		fillable := vals[0].(*big.Int)
		pend := head + 3_000
		_, err = chain.quoteAt(l, 0, fillable, pend/1000, 1234+15)
		require.Error(t, err, "premium from %s: the top is refused in the first block after the premium", from)
		_, err = chain.quoteAt(l, 0, fillable, (pend+200)/1000, 1234+16)
		require.NoError(t, err, "premium from %s: the top fills again a block later", from)
		for _, pt := range e.Ladders[0] {
			assert.Less(t, pt[0].ToBig().Cmp(fillable), 0, "premium from %s: the top is no point", from)
		}
		checkWindow(t, chain, p)
	}
}

// A premium that ends at the start of the window's last second has its first block after the end in that second, a
// block a fill may still land in: it is sampled, and a size refused there only (an output bound, the premium a little
// over the drift to its end) is no point.
func TestTrackerSamplesTheFirstBlockAfterAPremiumInTheWindowsLastSecond(t *testing.T) {
	chain, n := outputBound(t, 10_000, 0, 0)
	l := chain.listings[1]
	head := n.time*1000 + chain.subMs
	end := n.time + uint64(maxAge/time.Second) + subSecondInclusionAllowance
	window := int64(end*1000 - head) // the premium ends at the start of the window's last second
	l.controls = ctlWord(0, 10*window+2_000, window, 0)
	l.view.Word = testWord(int64(head), 30)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	require.Equal(t, end, e.ValidUntil, "valid to the window's end")
	checkWindow(t, chain, p)
}

// A premium that ends late in the window's last second (at .650 of it) ends inside the window: its last block, the
// least a size pays there, is sampled, with no output bound on either side.
func TestTrackerSamplesAPremiumEndingLateInTheWindowsLastSecond(t *testing.T) {
	chain, n := offCount(t, 0)
	l := chain.listings[1]
	l.decayPpmPerS = 100
	head := n.time*1000 + chain.subMs
	end := n.time + uint64(maxAge/time.Second) + subSecondInclusionAllowance
	l.controls = ctlWord(0, 5_000, int64(end*1000+650-head), 0)
	l.view.Word = testWord(int64(head), 30)
	p := track(t, n, listed(t, n))
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	checkWindow(t, chain, p)
}

// On a seconds clock (mode 0) the first block after a premium that ends inside a second is the next second's: the
// premium still stands in the second it ends in. An output-bound side refused only there drops the size.
func TestTrackerOnASecondsClockSamplesTheSecondAfterAPremium(t *testing.T) {
	chain := trackedChain()
	chain.clockMode = 0
	l := chain.listings[1]
	l.decayPpmPerS = 20_000
	l.controls = ctlWord(0, 82_000, 3_500, 0) // ends at T+3.5: T+4 lifts it, T+5 drifts it back under the bound
	l.outCap[0], l.fillableUnderCap = 5_000e6, true
	n := chain.node(t, 1234)
	l.view.Word = testWord(int64(n.time*1000), 30)
	p := track(t, n, listed(t, n))
	require.NotEmpty(t, extraOf(t, p).Ladders[0])
	checkWindow(t, chain, p)
}

// Where the node refuses block overrides, a side whose premium ends inside the window is quoted no later than the last
// second before it ends, though nothing else ages its levels: once the premium lifts it pays more, past an output bound
// the door then refuses.
func TestTrackerWithoutOverridesStopsBeforeAPremiumEnds(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.controls = ctlWord(0, 2_000, 3_000, 0)
	n := chain.node(t, 1234)
	n.refuseOverrides = true
	l.view.Word = testWord(int64(n.time*1000+chain.subMs), 30)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, n.time+2, e.ValidUntil, "the premium ends at T+3.400: T+2 is the last second wholly under it")

	// a premium that runs past the window: no bound, the window's end sample covers it
	l.controls = ctlWord(0, 2_000, 20_000, 0)
	p = track(t, n, p)
	assert.Equal(t, uint64(p.Timestamp)+uint64(maxAge/time.Second)+subSecondInclusionAllowance, extraOf(t, p).ValidUntil)
}

// Pinned on BaseTime with the count a second ahead of it, a premium edge whose block the front reads by its count
// cannot be placed (place refuses it): the side is quoted no later than the last second before the premium ends, as
// where an end cannot be placed.
func TestTrackerCapsAPremiumEdgeItCannotPlace(t *testing.T) {
	chain := trackedChain()
	chain.subMs = 450
	l := chain.listings[1]
	l.controls = ctlWord(0, 2_000, 2_591, 0) // ends at T+3.041: its last moment T+3.040 is not placed
	n := chain.node(t, 1234)
	chain.clockBlock0 = 1234 - 50
	chain.clockMs0 = n.time*1000 + 450 + 1_000 - 50*200
	l.view.Word = testWord(int64(n.time*1000+450), 30)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, n.time+2, e.ValidUntil)
}

// A listing whose fills read a price source is charged the print figure of its fill mode and direction: a PRINT or
// SESSION listing. A listing of an asset-module pair (the pair word's bit 15) is marked so too, and is quoted only
// with gas of its own: its module's checks are in no class figure.
func TestTrackerMarksAListingThatReadsASource(t *testing.T) {
	for _, c := range []struct {
		scopes uint8
		asset  bool
		print  bool
	}{{0, false, true}, {6, false, true}, {2, false, false}, {4, false, false}, {2, true, true}} {
		chain, n := printListing(t, c.scopes, 8_000)
		if !printed(c.scopes) {
			chain.listings[1].printMaxAgeMs = 0 // reads no print
		}
		if c.asset {
			chain.listings[1].view.PairWord.SetBit(chain.listings[1].view.PairWord, 15, 1)
		}
		e := extraOf(t, track(t, n, listed(t, n)))
		assert.Equal(t, c.print, e.Print, "scopes %d, asset %v", c.scopes, c.asset)
		if !c.asset {
			require.NotEmpty(t, e.Ladders[0], "scopes %d", c.scopes)
			continue
		}
		assert.Empty(t, e.Ladders[0], "an asset-module pair with no gas of its own is not quoted")
		assert.Empty(t, e.Ladders[1])
		cfg := testConfig()
		cfg.Fronts[0].ListingGas = []ListingGas{{Listing: 1, BaseIn: 171_000, QuoteIn: 169_000}}
		e = extraOf(t, trackWith(t, n, listed(t, n), cfg))
		require.NotEmpty(t, e.Ladders[0], "with its own gas")
		require.NotNil(t, e.Gas)
		assert.Equal(t, [2]int64{171_000, 169_000}, *e.Gas)
	}
}

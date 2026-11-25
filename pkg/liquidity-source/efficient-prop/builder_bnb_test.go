package efficientprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// bnbChain is the tracked listing on a front of BNB Chain's builder build: the same desk code as Ethereum's, its clock
// the millisecond precompile's mode (3) with a 999 ms age slack, as BNB Chain's build is set up, and every listing's
// tip cap at BNB Chain's default floor, 0.1 gwei. Its levels lose 100 ppm a second of age, so a later second pays
// less.
func bnbChain() *fakeChain {
	chain := ethChain()
	chain.clockMode, chain.ageSlackMs = clockPrecompile, 999
	for _, l := range chain.listings {
		l.view.TipCap = 0x8001
	}
	chain.listings[1].decayPpmPerS = 100
	return chain
}

// bnbNode is the chain behind a BNB Chain node at block, whose head's milliseconds are headMs, which its block answer
// gives where ms is set (milliTimestamp). Its millisecond precompile answers nothing, as before BNB Chain's upgrade
// that adds it, unless answers is set, when it answers the block's second in milliseconds.
func bnbNode(t *testing.T, chain *fakeChain, block, headMs uint64, ms bool, answers *bool) *node {
	n := chain.node(t, block)
	n.time = headMs / 1000
	if ms {
		n.ms = headMs
	}
	n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64,
		state stateDiffs) (bool, []byte) {
		if to == msPrecompile {
			if answers != nil && *answers {
				return true, packOut(baseTimeABI, "timestampMs", new(big.Int).SetUint64(at*1000))
			}
			return true, nil
		}
		return chain.builderRespond(to, data, at, number, sub, state)
	}
	return n
}

// bnbConfig is the test front on BNB Chain, read through the /state server at url, its test listings 1 and 4 with gas
// of their own.
func bnbConfig(url string) *Config {
	cfg := ethConfig(url)
	cfg.ChainID = valueobject.ChainIDBSC
	return cfg
}

// msCalls is how many calls the node answered to the millisecond precompile.
func msCalls(n *node) int {
	k := 0
	for _, c := range n.calls() {
		if c.to == msPrecompile {
			k++
		}
	}
	return k
}

// On BNB Chain a /state answer's slot is its block number, and names no second: the target block, the head's next, is
// at the head's milliseconds (the node's milliTimestamp) plus one 450 ms block. The refresh samples the door there,
// at that block's second, and a second later too, a block that comes late, each size keeping its lower output. The
// state is valid to the target block's time in ms, and route finding stops 100 ms before it (BNB Chain's allowance).
func TestBNBSamplesTheTargetBlockAtTheHeadsMillisecondsPlusOneBlock(t *testing.T) {
	chain := bnbChain()
	now := uint64(time.Now().Unix())
	headMs := now*1000 + 700
	targetMs := headMs + 450 // in the second after the head's
	n := bnbNode(t, chain, 1234, headMs, true, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := bnbConfig(st.URL)
	p := ethListed(t, n, cfg)
	before := len(n.calls())

	got := ethTrack(t, n, cfg, p)
	e := extraOf(t, got)
	assert.True(t, e.Builder)
	assert.Equal(t, uint64(1234), got.BlockNumber)
	assert.Equal(t, int64(now+1), got.Timestamp, "the target block's second")
	assert.Equal(t, targetMs, e.ValidUntilMs, "the target block's time, ms")
	assert.Equal(t, uint64(100), e.AllowanceMs, "BNB Chain's allowance")
	assert.Equal(t, (targetMs-100)/1000, e.ValidUntil)
	assert.Equal(t, uint64(100_000_000), e.TipCapWei)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	assert.True(t, sim.validAtMs(targetMs-100), "until 100 ms before the target block")
	assert.False(t, sim.validAtMs(targetMs-99))

	seconds := map[uint64]int{}
	for _, q := range quotesAfter(n, before) {
		assert.Equal(t, uint64(1235), q.number, "no quote outside the target block")
		seconds[q.at]++
	}
	assert.Positive(t, seconds[now+1], "the target block's second")
	assert.Positive(t, seconds[now+2], "the second after, where the block comes late")
	assert.Len(t, seconds, 2)
	l := chain.listings[1]
	for dir := range 2 {
		require.NotEmpty(t, e.Ladders[dir], "direction %d", dir)
		for _, pt := range e.Ladders[dir] {
			on, err := chain.quoteAt(l, dir, pt[0].ToBig(), now+1, 1235)
			require.NoError(t, err)
			late, err := chain.quoteAt(l, dir, pt[0].ToBig(), now+2, 1235)
			require.NoError(t, err)
			assert.Equal(t, minBig(on, late), pt[1].ToBig(), "the lower of the two seconds")
		}
	}
	assert.Positive(t, msCalls(n), "the precompile is read beside the head read")
}

// A node that gives no milliseconds leaves the head at the start of its second: the target block is taken 450 ms
// after it, earlier than it may be, so the state ends no later than it would, and the late sample covers the second
// after.
func TestBNBHeadWithoutMillisecondsStartsAtItsSecond(t *testing.T) {
	chain := bnbChain()
	now := uint64(time.Now().Unix())
	n := bnbNode(t, chain, 1234, now*1000+700, false, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := bnbConfig(st.URL)
	e := extraOf(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)))
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, now*1000+450, e.ValidUntilMs)
	assert.Equal(t, now, e.ValidUntil)
}

// An answer is for the head's next block alone, and its slot, where it names one, is that block's number: an answer
// for another block, or whose slot is another number (a beacon slot), quotes nothing.
func TestBNBAnswerMustBeTheHeadsNextBlock(t *testing.T) {
	now := uint64(time.Now().Unix())
	for _, c := range []struct {
		name         string
		slot, number uint64
		quoted       bool
	}{
		{"the slot is the block", 1235, 1235, true},
		{"no slot", 0, 1235, true},
		{"a beacon slot", slotAfter(now), 1235, false},
		{"the head's block", 1234, 1234, false},
		{"two blocks ahead", 1236, 1236, false},
	} {
		chain := bnbChain()
		n := bnbNode(t, chain, 1234, now*1000+100, true, nil)
		st := newStateServer(t)
		st.set(c.slot, c.number, map[common.Address]*big.Int{testCellA: deskSlot0(c.number, false)})
		cfg := bnbConfig(st.URL)
		got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
		if c.quoted {
			assert.NotEmpty(t, extraOf(t, got).Ladders[0], c.name)
		} else {
			unserved(t, got, c.name)
		}
	}
}

// BNB Chain's front reads the millisecond precompile (mode 3), which answers nothing before the chain's upgrade that
// adds it: the front then reads the block's second, as a seal clock does, and is quoted. Once the precompile answers,
// the front reads the block's own milliseconds, which no block override sets, so nothing is quoted. A front on the
// seal clock (mode 0) is quoted whatever the precompile answers, and Ethereum's reads never call it.
func TestBNBQuotesThePrecompileModeOnlyWhileItAnswersNothing(t *testing.T) {
	now := uint64(time.Now().Unix())
	yes, no := true, false
	for _, c := range []struct {
		name    string
		mode    uint64
		answers *bool
		quoted  bool
	}{
		{"mode 3, no precompile", clockPrecompile, &no, true},
		{"mode 3, the precompile answers", clockPrecompile, &yes, false},
		{"mode 0, the precompile answers", clockSeconds, &yes, true},
		{"a block-count clock", clockBlocks, &no, false},
	} {
		chain := bnbChain()
		chain.clockMode = c.mode
		n := bnbNode(t, chain, 1234, now*1000+100, true, c.answers)
		st := newStateServer(t)
		st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		cfg := bnbConfig(st.URL)
		got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
		if c.quoted {
			assert.NotEmpty(t, extraOf(t, got).Ladders[0], c.name)
		} else {
			unserved(t, got, c.name)
		}
	}

	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	require.NotEmpty(t, extraOf(t, ethTrack(t, n, cfg, ethListed(t, n, cfg))).Ladders[0])
	assert.Zero(t, msCalls(n), "Ethereum's reads never call the precompile")
}

// Where the listing and the word both carry, a fill may still read the word in the block after the target, one block
// (450 ms) later: that block is sampled too, at its own second, and the state is valid to its time.
func TestBNBSamplesTheCarriedBlockOneBlockLater(t *testing.T) {
	chain := bnbChain()
	chain.listings[1].view.Scopes = 2 | 8
	now := uint64(time.Now().Unix())
	headMs := now*1000 + 300
	targetMs := headMs + 450 // now.750
	nextMs := targetMs + 450 // now+1.200
	n := bnbNode(t, chain, 1234, headMs, true, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	cfg := bnbConfig(st.URL)
	p := ethListed(t, n, cfg)
	before := len(n.calls())
	e := extraOf(t, ethTrack(t, n, cfg, p))
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, nextMs, e.ValidUntilMs, "valid to the carried block's time")
	assert.Equal(t, (nextMs-100)/1000, e.ValidUntil)
	var carried int
	for _, q := range quotesAfter(n, before) {
		if q.number == 1236 {
			carried++
			assert.Equal(t, nextMs/1000, q.at)
			assert.Contains(t, q.state, testCellA, "the same slots")
		}
	}
	assert.Positive(t, carried)

	// the carried block's sample not read: the target block's outputs stand, valid to its time alone
	n.refuseNumber = 1236
	e = extraOf(t, ethTrack(t, n, cfg, p))
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, targetMs, e.ValidUntilMs, "the target block alone")
	assert.Equal(t, (targetMs-100)/1000, e.ValidUntil)
}

// BNB Chain's builder build was not measured for a class figure: a listing is quoted there only with gas of its own
// (Front.ListingGas, or its credit listing's), and is idle without.
func TestBNBQuotesOnlyListingsWithGasOfTheirOwn(t *testing.T) {
	chain := bnbChain()
	now := uint64(time.Now().Unix())
	n := bnbNode(t, chain, 1234, now*1000+100, true, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := bnbConfig(st.URL)
	p := ethListed(t, n, cfg)
	cfg.Fronts[0].ListingGas = nil
	got := ethTrack(t, n, cfg, p)
	unserved(t, got, "no gas of its own")
	assert.True(t, extraOf(t, got).Idle)
}

// A listing whose tip cap is under BNB Chain's default floor, 0.1 gwei, is not quoted: its fills would revert at the
// tips BNB Chain's blocks take.
func TestBNBDefaultTipCapFloor(t *testing.T) {
	chain := bnbChain()
	now := uint64(time.Now().Unix())
	n := bnbNode(t, chain, 1234, now*1000+100, true, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := bnbConfig(st.URL)
	p := ethListed(t, n, cfg)
	require.NotEmpty(t, extraOf(t, ethTrack(t, n, cfg, p)).Ladders[0], "a cap at the floor")
	chain.listings[1].view.TipCap = 0x7009 // 0.09 gwei
	e := extraOf(t, ethTrack(t, n, cfg, p))
	assert.Empty(t, e.Ladders[0], "under the floor")
	assert.True(t, e.Idle)
}

// On BNB Chain a second is two blocks and more: a state that is not idle, quoted or waiting on its next block's
// update, is asked for at once, and an idle one idleRefreshEvery on, as on Ethereum, where a state that is not idle is
// asked for a second on.
func TestBNBAsksForAStateThatIsNotIdleAtOnce(t *testing.T) {
	chain := bnbChain()
	now := uint64(time.Now().Unix())
	n := bnbNode(t, chain, 1234, now*1000+100, true, nil)
	st := newStateServer(t)
	st.set(1235, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := bnbConfig(st.URL)
	p := ethListed(t, n, cfg)
	at := time.Unix(int64(now), 0)

	quoted := ethTrack(t, n, cfg, p)
	require.NotEmpty(t, extraOf(t, quoted).Ladders[0])
	assert.Equal(t, int64(now), ethTracker(t, n, cfg, at).NextUpdateAt(quoted), "a quoted state")

	st.set(1235, 1235, nil)
	waiting := ethTrack(t, n, cfg, p)
	e := extraOf(t, waiting)
	require.Empty(t, e.Ladders[0], "no update for the head's next block")
	require.False(t, e.Idle)
	assert.Equal(t, int64(now), ethTracker(t, n, cfg, at).NextUpdateAt(waiting), "a state waiting on its update")

	eth := ethConfig(st.URL)
	assert.Equal(t, int64(now+1), ethTracker(t, n, eth, at).NextUpdateAt(waiting), "Ethereum: a second on")

	cfg.Fronts[0].ListingGas = nil
	idle := ethTrack(t, n, cfg, p)
	require.True(t, extraOf(t, idle).Idle)
	assert.Equal(t, int64(now)+int64(idleRefreshEvery/time.Second), ethTracker(t, n, cfg, at).NextUpdateAt(idle),
		"an idle state")
}

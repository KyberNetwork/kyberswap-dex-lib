package efficientprop

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// stateServer is a fake /state endpoint: it answers titan_getPammStateOverrides with its map, delay after the request,
// and counts the reads.
type stateServer struct {
	*httptest.Server
	mu    sync.Mutex
	m     map[string]any
	hits  atomic.Int64
	delay time.Duration
}

func newStateServer(t *testing.T) *stateServer {
	s := &stateServer{m: map[string]any{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil || msg.Method != "titan_getPammStateOverrides" {
			t.Errorf("a request the /state server does not answer: %s, %v", msg.Method, err)
			return
		}
		s.hits.Add(1)
		s.mu.Lock()
		delay := s.delay
		s.mu.Unlock()
		time.Sleep(delay)
		s.mu.Lock()
		reply := map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": s.m}
		_ = json.NewEncoder(w).Encode(reply)
		s.mu.Unlock()
	}))
	t.Cleanup(s.Close)
	return s
}

// set makes the map's target block number at slot's second, with an entry per desk contract holding pair slot 0's
// quote word.
func (s *stateServer) set(slot, number uint64, cells map[common.Address]*big.Int) {
	m := map[string]any{"slot": slot, "blockNumber": hexutil.EncodeUint64(number)}
	for cell, w := range cells {
		m["entry-"+cell.Hex()] = map[string]any{"stateOverride": map[string]any{cell.Hex(): map[string]any{
			"stateDiff": map[string]string{quoteKey(0).Hex(): common.BigToHash(w).Hex()},
			"balance":   "0x0", "nonce": "0x1"}}}
	}
	s.mu.Lock()
	s.m = m
	s.mu.Unlock()
}

// deskSlot0 is a desk contract's quote word as the desk's relay entry for block target writes it at desk epoch 0: its
// stamp, the block's low 32 bits, and where carry, the quote's carry (bit 201).
func deskSlot0(target uint64, carry bool) *big.Int {
	w := new(big.Int).SetUint64(target & 0xffffffff)
	if carry {
		w.SetBit(w, 201, 1)
	}
	return w
}

// slotAfter is the first beacon slot whose second is after sec.
func slotAfter(sec uint64) uint64 {
	return (sec-beaconGenesis)/12 + 1
}

func slotTime(slot uint64) uint64 { return beaconGenesis + 12*slot }

// ethChain is the tracked listing on a front of Ethereum's builder build: its clock the block's second (mode 0) with
// a 2 s age slack, its listing bound to desk 7, whose desk contract is testCellA.
func ethChain() *fakeChain {
	chain := trackedChain()
	chain.clockMode, chain.subMs, chain.ageSlackMs = clockSeconds, 0, 2_000
	chain.listings[1].view.Desk = 7
	// a tip cap at Ethereum's default floor, 2 gwei
	for _, l := range chain.listings {
		l.view.TipCap = 0x9002
	}
	return chain
}

// stands is whether listing l's quote word stands in block number on the call's state: the state carries its desk
// contract's quote word at its pair slot (0x100 + wordIndex) stamped with that block, or with the block before where
// the listing (scopes bit 3) and the word (bit 201) both carry and its fence kind is not NONE, as the front's freshness
// reads it; and the desk, as the chain's head holds its desk word, is not stopped and holds the word's desk epoch.
func (c *fakeChain) stands(l *fakeListing, number uint64, state stateDiffs) bool {
	w, ok := state[c.cellFor(l.view.Desk)][quoteKey(l.view.WordIndex)]
	if !ok {
		return false
	}
	if d := c.deskWord(l.view.Desk); d.Bit(224) == 1 || bits(d, 160, 31) != bits(w.Big(), 32, 31) {
		return false
	}
	stamp := w.Big().Uint64() & 0xffffffff
	if stamp == number&0xffffffff {
		return true
	}
	return number&0xffffffff == stamp+1 && (l.view.Scopes>>3)&1 == 1 && (l.view.Scopes>>1)&3 != 2 &&
		w.Big().Bit(201) == 1
}

// builderRespond is the chain behind a builder chain's node: doorView answers the word with the stamp and carry of
// the desk contract's quote word (its pair slot's) in the call's state (the carry at the listing's carryBit, bit 4
// where unset), and a door refuses every size where the listing's word does not stand in the call's block (the
// front's NoWord, which its door answers as NoFill).
func (c *fakeChain) builderRespond(to common.Address, data []byte, at, number uint64, sub int64,
	state stateDiffs) (bool, []byte) {
	if to == c.front {
		// the front's plan answers 0 to a word that does not stand in the call's block (its NoWord, a state refusal)
		if m, err := frontABI.MethodById(data[:4]); err == nil && m.Name == "quoteFrom" {
			args, _ := m.Inputs.Unpack(data[4:])
			if l, ok := c.listings[uint32(args[3].(*big.Int).Uint64())]; ok && !c.stands(l, number, state) {
				return true, packOut(frontABI, "quoteFrom", new(big.Int))
			}
		}
		if m, err := frontABI.MethodById(data[:4]); err == nil && m.Name == "doorView" {
			args, _ := m.Inputs.Unpack(data[4:])
			if l, ok := c.listings[uint32(args[0].(*big.Int).Uint64())]; ok {
				v := l.view
				v.Word = new(big.Int).Set(l.view.Word)
				if w, ok := state[c.cellFor(l.view.Desk)][quoteKey(l.view.WordIndex)]; ok {
					stamp := new(big.Int).SetUint64(w.Big().Uint64() & 0xffffffff)
					keep := new(big.Int).Lsh(big.NewInt(0xffffffff), 168)
					v.Word.AndNot(v.Word, keep).Or(v.Word, stamp.Lsh(stamp, 168))
					v.Word.AndNot(v.Word, big.NewInt(0xf0))
					if w.Big().Bit(201) == 1 {
						v.Word.SetBit(v.Word, max(l.carryBit, 4), 1)
					}
				}
				return true, packOut(frontABI, "doorView", v)
			}
		}
	}
	if _, l := c.doorListing(to); l != nil {
		if m, err := doorABI.MethodById(data[:4]); err == nil && (m.Name == "quote" || m.Name == "quoteFillable") &&
			!c.stands(l, number, state) {
			return false, crypto.Keccak256([]byte("NoFill()"))[:4]
		}
	}
	return c.respond(to, data, at, number, sub)
}

// builderNode is the chain behind a node of a builder chain at block, its head's second on the beacon grid (the first
// slot's second after now), as every block of a beacon chain is.
func builderNode(t *testing.T, chain *fakeChain, block uint64) *node {
	n := chain.node(t, block)
	n.respondState = chain.builderRespond
	n.time = slotTime(slotAfter(n.time))
	return n
}

// ethConfig is the test front on Ethereum, read through the /state server at url. Its test listings read no price
// source (WORD), a class with no figure on Ethereum, so the config names listings 1 and 4 with gas of their own.
func ethConfig(url string) *Config {
	return &Config{ChainID: valueobject.ChainIDEthereum, ApprovedDesks: testDesks(), Fronts: []Front{{
		Address:         hexutil.Encode(testFront[:]),
		DoorFactory:     hexutil.Encode(testFactory[:]),
		CellCodehash:    hexutil.Encode(testCellHash[:]),
		Layout:          LayoutBuilder,
		FillMode:        FillModeDoor,
		Implementations: []string{hexutil.Encode(testImpl[:])},
		Titan:           &titan.Config{URLs: []string{url}, Timeout: 2 * time.Second},
		ListingGas: []ListingGas{{Listing: 1, BaseIn: testGasBaseIn, QuoteIn: testGasQuoteIn},
			{Listing: 4, BaseIn: testGasBaseIn, QuoteIn: testGasQuoteIn}},
	}}}
}

// testGasBaseIn and testGasQuoteIn are the test listings' own gas under ethConfig.
const (
	testGasBaseIn  = 107_621
	testGasQuoteIn = 106_052
)

// ethListed is the chain's listing 1 as the lister records it under cfg.
func ethListed(t *testing.T, n *node, cfg *Config) entity.Pool {
	t.Helper()
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	p := pools[0]
	p.Tokens[0].Decimals, p.Tokens[1].Decimals = 18, 6
	return p
}

// ethTracker is a tracker under cfg whose clock reads now.
func ethTracker(t *testing.T, n *node, cfg *Config, now time.Time) *PoolTracker {
	t.Helper()
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	tr.now = func() time.Time { return now }
	return tr
}

func ethTrack(t *testing.T, n *node, cfg *Config, p entity.Pool) entity.Pool {
	t.Helper()
	got, err := ethTracker(t, n, cfg, time.Now()).GetNewPoolState(context.Background(), p,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	return got
}

// quotesAfter is the door quotes the node answered after its first k calls.
func quotesAfter(n *node, k int) []sent {
	var out []sent
	for _, c := range n.calls()[k:] {
		if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == "0xb6466384" {
			out = append(out, c)
		}
	}
	return out
}

// On Ethereum a desk's update lands only in a block a builder built with it ahead of a fill, so the head never
// holds a word that stands: each refresh reads /state and samples the door under its overrides, at its target block's
// number and second. Every read the state is made of carries them, the listing's own read and its pool's balances
// included; the ladder pays the door's answer in the target block, and is valid to that block's second less the
// inclusion allowance, with no allowance left for route finding to take again.
func TestBuilderSamplesTheDoorUnderTheTargetBlocksState(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	target := slotTime(slot)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	before := len(n.calls())

	got := ethTrack(t, n, cfg, p)
	assert.Equal(t, int64(1), st.hits.Load(), "one /state read a refresh")
	e := extraOf(t, got)
	assert.True(t, e.Builder)
	assert.Equal(t, uint64(1234), got.BlockNumber, "the state's block is the head's")
	assert.Equal(t, int64(target), got.Timestamp, "its age runs from the target block's second")
	assert.Equal(t, target-1, e.ValidUntil, "the target block's second less the 1 s allowance")
	assert.Zero(t, e.Allowance, "the allowance is taken off validUntil already")
	assert.Equal(t, target*1000, e.ValidUntilMs, "the end of the state, the target block's time in ms")
	assert.Equal(t, uint64(1000), e.AllowanceMs, "the config's allowance (1 s on a builder front) in ms")
	// route finding quotes a fill until the target block's time less the allowance, to the millisecond
	sim, simErr := NewPoolSimulator(pool.FactoryParams{EntityPool: got, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, simErr)
	assert.True(t, sim.validAtMs(target*1000-1000), "at the target time less the allowance")
	assert.False(t, sim.validAtMs(target*1000-1000+1), "a millisecond after")
	assert.Equal(t, entity.PoolReserves{"9000000000000000000", "40000000000"}, got.Reserves)
	l := chain.listings[1]
	for dir := range 2 {
		require.NotEmpty(t, e.Ladders[dir], "direction %d", dir)
		for _, pt := range e.Ladders[dir] {
			want, err := chain.quoteAt(l, dir, pt[0].ToBig(), target, 1235)
			require.NoError(t, err)
			assert.Equal(t, want, pt[1].ToBig())
		}
	}

	// every read after the head's block is pinned to the head and runs in the target block on the map's state
	calls := n.calls()[before:]
	require.NotEmpty(t, calls)
	assert.Equal(t, "latest", calls[0].tag, "the first read takes the head's block")
	assert.Nil(t, calls[0].state)
	quotes, balances := 0, 0
	for _, c := range calls[1:] {
		if c.tag == "latest" || c.to == chain.front && len(c.data) == common.HashLength {
			continue // the head's own time, and the front's storage
		}
		if c.number != 1235 {
			continue // the head's time, read beside the state
		}
		assert.Equal(t, hexutil.EncodeUint64(1234), c.tag)
		assert.Equal(t, target, c.at)
		assert.Contains(t, c.state, testCellA, "the map's state rides on every call")
		if hexutil.Encode(c.data[:4]) == "0xb6466384" {
			quotes++
		}
		if hexutil.Encode(c.data[:4]) == "0x70a08231" {
			balances++
		}
	}
	assert.Greater(t, quotes, 20)
	assert.Equal(t, 2, balances, "the pool's balances are read under the state too")
	for _, c := range quotesAfter(n, before) {
		assert.Equal(t, uint64(1235), c.number, "no quote outside the target block")
	}

	// the simulator quotes it, and pays the door's answer at a sampled size
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	pt := e.Ladders[0][len(e.Ladders[0])/2]
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{
		Token: hexutil.Encode(testWETH[:]), Amount: pt[0].ToBig()}, TokenOut: hexutil.Encode(testUSDC[:])})
	require.NoError(t, err)
	assert.Equal(t, pt[1].ToBig(), res.TokenAmountOut.Amount)
}

// A map whose block is not ahead of the head (it was for a block already built), one with no block, and one with no
// entry quote nothing: the door is not even asked.
func TestBuilderQuotesNothingWithoutAMapAheadOfTheHead(t *testing.T) {
	for name, set := range map[string]func(*stateServer, uint64){
		"the head's block": func(s *stateServer, slot uint64) {
			s.set(slot, 1234, map[common.Address]*big.Int{testCellA: deskSlot0(1234, false)})
		},
		"an older block": func(s *stateServer, slot uint64) {
			s.set(slot, 1230, map[common.Address]*big.Int{testCellA: deskSlot0(1230, false)})
		},
		"no entry": func(s *stateServer, slot uint64) { s.set(slot, 1235, nil) },
		"no block": func(s *stateServer, slot uint64) {
			s.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
			delete(s.m, "blockNumber")
		},
		"no answer": func(*stateServer, uint64) {},
	} {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		set(st, slotAfter(n.time))
		cfg := ethConfig(st.URL)
		p := ethListed(t, n, cfg)
		before := len(n.calls())
		got := ethTrack(t, n, cfg, p)
		unserved(t, got, name)
		assert.Empty(t, quotesAfter(n, before), name)
		assert.True(t, extraOf(t, got).Builder, name)
	}
}

// A listing whose desk contract has no entry in the map is not quoted, though the map has others: its desk's update
// is not in the target block.
func TestBuilderQuotesNothingForADeskWithNoEntry(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellB: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	before := len(n.calls())
	got := ethTrack(t, n, cfg, p)
	unserved(t, got, "no entry for the listing's desk contract")
	assert.Empty(t, quotesAfter(n, before))
}

// The map's second must be after the head's: a map whose slot is not later than the head's block is not quoted.
func TestBuilderQuotesNothingOnAMapNotAfterTheHeadsSecond(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time)-1, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	unserved(t, got, "the target's second is not after the head's")
}

// A node that takes the overrides and runs the call elsewhere (the head's block or state) gives no sample.
func TestBuilderQuotesNothingWhereTheNodeIgnoresTheOverride(t *testing.T) {
	for name, edit := range map[string]func(*node){
		"block override": func(n *node) { n.ignoreOverrides = true },
		"block number":   func(n *node) { n.ignoreNumber = true },
		"state override": func(n *node) { n.ignoreStateOverrides = true },
		"no overrides":   func(n *node) { n.refuseOverrides = true },
	} {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		cfg := ethConfig(st.URL)
		p := ethListed(t, n, cfg)
		edit(n)
		tr := ethTracker(t, n, cfg, time.Now())
		got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		if err != nil {
			continue // a node refusing the call fails the refresh, and the last state ages out
		}
		unserved(t, got, name)
	}
}

// On a builder chain the front's clock is the block's second (SEAL, mode 0), which a time override places. A front on
// any other clock is not quoted, and the chain's millisecond-time contract is never read.
func TestBuilderQuotesNothingOffTheSealClock(t *testing.T) {
	chain := ethChain()
	chain.clockMode = clockBlocks
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	unserved(t, got, "a block-count clock")
	for _, c := range n.calls() {
		assert.NotEqual(t, testBaseTime, c.to, "no read of the millisecond-time contract")
	}
}

// carryChain is the tracked listing whose levels lose 100 ppm a second of age, bound with scopes.
func carryChain(scopes uint8) *fakeChain {
	chain := ethChain()
	chain.listings[1].view.Scopes = scopes
	chain.listings[1].decayPpmPerS = 100
	return chain
}

// Where the listing and the word both carry (and the fence kind is not NONE), a fill may still read the word in the
// block after its target, under the same slots: the refresh samples there too, keeps the lower output of each size,
// and is valid to that block's second less the allowance. Where either does not carry, the target block alone.
func TestBuilderSamplesTheNextBlockWhereBothCarry(t *testing.T) {
	const wordCarry, wordOnly, none = 2 | 8, 2, 4 | 8 // WORD with a carry life, WORD without, NONE with one
	for _, c := range []struct {
		name   string
		scopes uint8
		carry  bool
		next   bool
	}{
		{"both carry", wordCarry, true, true},
		{"the word alone", wordOnly, true, false},
		{"the listing alone", wordCarry, false, false},
		{"a NONE listing", none, true, false},
	} {
		chain := carryChain(c.scopes)
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		slot := slotAfter(n.time)
		target := slotTime(slot)
		st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, c.carry)})
		cfg := ethConfig(st.URL)
		p := ethListed(t, n, cfg)
		before := len(n.calls())
		got := ethTrack(t, n, cfg, p)
		e := extraOf(t, got)
		next := 0
		for _, q := range quotesAfter(n, before) {
			if q.number == 1236 {
				next++
				assert.Equal(t, target+12, q.at, c.name)
				assert.Contains(t, q.state, testCellA, "%s: the same slots", c.name)
			}
		}
		l := chain.listings[1]
		if !c.next {
			assert.Zero(t, next, c.name)
			assert.Equal(t, target-1, e.ValidUntil, c.name)
			continue
		}
		assert.Positive(t, next, c.name)
		assert.Equal(t, target+12-1, e.ValidUntil, "%s: valid to the next block's second", c.name)
		for dir := range 2 {
			require.NotEmpty(t, e.Ladders[dir], c.name)
			for _, pt := range e.Ladders[dir] {
				at, err := chain.quoteAt(l, dir, pt[0].ToBig(), target, 1235)
				require.NoError(t, err)
				later, err := chain.quoteAt(l, dir, pt[0].ToBig(), target+12, 1236)
				require.NoError(t, err)
				assert.Equal(t, minBig(at, later), pt[1].ToBig(), "%s: the lower of the two blocks", c.name)
				assert.Equal(t, later, pt[1].ToBig(), "%s: the older word pays less", c.name)
			}
		}
	}
}

func minBig(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

// A size the door refuses in the carried block is dropped, and where that block's sample is not read the state is
// the target block's alone.
func TestBuilderCarriedBlockRefusalsAndGaps(t *testing.T) {
	chain := carryChain(2 | 8)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	target := slotTime(slot)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	chain.listings[1].shrinkAt, chain.listings[1].shrinkTo = target+12, 1e18
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	got := ethTrack(t, n, cfg, p)
	e := extraOf(t, got)
	require.NotEmpty(t, e.Ladders[0])
	assert.LessOrEqual(t, e.Ladders[0][len(e.Ladders[0])-1][0].Uint64(), uint64(1e18),
		"no size the carried block refuses")
	assert.Equal(t, target+12-1, e.ValidUntil)

	chain.listings[1].shrinkAt = 0
	n.refuseNumber = 1236
	got = ethTrack(t, n, cfg, p)
	e = extraOf(t, got)
	require.NotEmpty(t, e.Ladders[0])
	assert.Equal(t, target-1, e.ValidUntil, "the carried block unread: the target block alone")
	for _, pt := range e.Ladders[0] {
		want, err := chain.quoteAt(chain.listings[1], 0, pt[0].ToBig(), target, 1235)
		require.NoError(t, err)
		assert.Equal(t, want, pt[1].ToBig())
	}
}

// A new map version, a higher target block, is picked up by the next refresh, which the tracker asks for at the
// earlier of the state's validUntil and a second from now; a state with no ladder a second from now.
func TestBuilderRefreshesOnAHigherVersion(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	target := slotTime(slot)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	got := ethTrack(t, n, cfg, p)
	e := extraOf(t, got)
	require.Equal(t, target-1, e.ValidUntil)

	at := func(now uint64, q entity.Pool) int64 {
		return ethTracker(t, n, cfg, time.Unix(int64(now), 0)).NextUpdateAt(q)
	}
	assert.Equal(t, int64(target-4), at(target-5, got), "a second from now")
	assert.Equal(t, int64(target-1), at(target-1, got), "no later than validUntil")
	assert.Equal(t, int64(target+3), at(target+3, got), "a state past its validUntil: now")
	unquoted := got
	unquoted.Extra = `{"builder":true}`
	assert.Equal(t, int64(target+8), at(target+7, unquoted), "no ladder: a second from now")

	// the head moves on, and the map with it
	n.block, n.time = 1235, target
	st.set(slot+1, 1236, map[common.Address]*big.Int{testCellA: deskSlot0(1236, false)})
	before := len(n.calls())
	got = ethTrack(t, n, cfg, got)
	e = extraOf(t, got)
	assert.Equal(t, int64(target+12), got.Timestamp)
	assert.Equal(t, target+12-1, e.ValidUntil)
	require.NotEmpty(t, e.Ladders[0])
	for _, q := range quotesAfter(n, before) {
		assert.Equal(t, uint64(1236), q.number)
	}
}

// A front on Base keeps its schedule: a second after the state's block time.
func TestBuilderScheduleLeavesBaseAsItIs(t *testing.T) {
	tr, err := NewPoolTracker(testConfig(), nil)
	require.NoError(t, err)
	tr.now = func() time.Time { return time.Unix(5000, 0) }
	b, err := json.Marshal(Extra{ValidUntil: 1030})
	require.NoError(t, err)
	assert.Equal(t, int64(1001), tr.NextUpdateAt(entity.Pool{Timestamp: 1000, Extra: string(b)}))
}

// On Ethereum no desk posts by a log of its own (DeskPush is the storage build's) and no stored price lane posts
// (Posted): a desk's update comes in /state with the block that lands it. So the parser maps neither, and reads no
// listing's price binding to learn lanes, while the front's logs naming a listing (its settings, its retirement and
// relisting among them), a desk contract's LaneFill (its cell learned from doorView and deskCellOf) and the fill's
// Swapped, which that contract logs there with the door its sender, still map to the doors they move.
func TestEventParserOnABuilderChain(t *testing.T) {
	chain := ethChain()
	chain.listings[1].view.Scopes, chain.listings[1].sourceIds = 0, big.NewInt(1) // PRINT on testSource
	chain.listings[4].view.Desk = 7
	n := builderNode(t, chain, 900)
	cfg := ethConfig("http://127.0.0.1:1")
	cfg.Fronts[0].ApprovedSources = []string{hexutil.Encode(testSource[:])}
	p := NewPoolFactory(cfg, n.client())

	front := func(event string, topics ...common.Hash) types.Log {
		return types.Log{Address: testFront, Topics: append([]common.Hash{frontABI.Events[event].ID}, topics...)}
	}
	push := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["DeskPush"].ID}}
	fill := types.Log{Address: testCellA, Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	swapped := swappedLog(t, testCellA, fakeDoorOf(testFactory, 4))
	settings, retired, relisted := front("SettingsSet", id(4)), front("PoolRetired", id(4)),
		front("ListingRelisted", id(4))
	post := posted(testSource, 3)
	got, err := p.Decode(context.Background(), []types.Log{push, post, fill, swapped, settings, retired, relisted})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{
		door(1): {fill},
		door(4): {fill, swapped, settings, retired, relisted},
	}, got)
	assert.Empty(t, storageReads(chain, n, recordSlot(1, 10)), "no lane binding read")
	assert.Empty(t, storageReads(chain, n, sourceEntrySlot(1)), "no source entry read")

	// the same logs on Base map a desk's post and its lane's
	base := testChain()
	base.listings[1].view.Desk, base.listings[1].view.Scopes, base.listings[1].sourceIds = 7, 0, big.NewInt(1)
	got, err = NewPoolFactory(testConfig(), base.node(t, 900).client()).Decode(context.Background(),
		[]types.Log{push, post})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {push, post}}, got)
}

// A listing read through /state is charged Ethereum's figures of its fill mode, by its pair's place (on its pool's
// hook pair, past it with the pool's asset its base, or past it with neither token the asset) and by its input token:
// the least first fill measured of its class on the deployed build. A class never measured there has no figure in
// either mode: a listing that reads no price source, a pair past the hook pair whose quote is the pool's asset, and a
// listing bound to a guarded entry. A state with a ladder of such a class is no pool, and one with no ladder is an
// empty pool. A listing's own figures, where its state carries them, stand in place of its class's in either mode.
func TestSimulatorChargesEthereumsGas(t *testing.T) {
	type figures struct{ door, front [2]int64 }
	want := map[[4]bool]figures{
		// {print, past, noAsset, assetQuote}: base in, quote in
		{true, false, false, false}: {door: [2]int64{134_808, 133_439}, front: [2]int64{123_100, 123_661}},
		{true, true, false, false}:  {door: [2]int64{161_844, 153_433}, front: [2]int64{150_136, 142_407}},
		{true, true, true, false}:   {door: [2]int64{167_896, 171_112}, front: [2]int64{158_011, 158_141}},
		// noAsset sets nothing on the hook pair, whose base is the pool's asset
		{true, false, true, false}: {door: [2]int64{134_808, 133_439}, front: [2]int64{123_100, 123_661}},
	}
	unmeasured := [][4]bool{
		{true, true, false, true}, // past the hook pair, the quote the pool's asset
		{false, false, false, false}, {false, true, false, false}, {false, true, false, true},
		{false, true, true, false}, // no price source read
	}
	builderPool := func(k [4]bool, guarded, base1 bool, mode uint8) entity.Pool {
		p := simPool(t, simLadders(), time.Now().Unix())
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		e.Builder, e.BaseToken1, e.Print, e.Guarded, e.Past, e.NoAsset = true, base1, k[0], guarded, k[1], k[2]
		e.AssetQuote = k[3]
		b, err := json.Marshal(e)
		require.NoError(t, err)
		p.Extra = string(b)
		return withStatic(t, p, func(se *StaticExtra) { se.Mode = mode })
	}
	for k, f := range want {
		for mode, w := range map[uint8][2]int64{modeDoor: f.door, modeFront: f.front} {
			for _, base1 := range []bool{false, true} {
				sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: builderPool(k, false, base1, mode)})
				require.NoError(t, err)
				token0, token1 := w[0], w[1]
				if base1 {
					token0, token1 = token1, token0
				}
				res, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
				require.NoError(t, err)
				assert.Equal(t, token0, res.Gas, "mode %d, token 0 in: %v, base token 1 %v", mode, k, base1)
				res, err = calc(sim, simUSDC, simWETH, big.NewInt(3000e6), nil)
				require.NoError(t, err)
				assert.Equal(t, token1, res.Gas, "mode %d, token 1 in: %v, base token 1 %v", mode, k, base1)
			}
			_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: builderPool(k, true, false, mode)})
			assert.ErrorIs(t, err, ErrInvalidState, "no guarded class on a builder chain: %v", k)
		}
		assert.Less(t, f.front[0]+f.front[1], f.door[0]+f.door[1], "the front's mode is under the door's: %v", k)
	}
	for _, k := range unmeasured {
		for _, mode := range []uint8{modeDoor, modeFront} {
			_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: builderPool(k, false, false, mode)})
			assert.ErrorIs(t, err, ErrInvalidState, "a class never measured, mode %d: %v", mode, k)
		}
	}
	// a class with no figure and no ladder is an empty pool, not an error
	empty := builderPool([4]bool{false, false, false, false}, false, false, modeFront)
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(empty.Extra), &e))
	e.Ladders = [2][]Point{}
	b, err := json.Marshal(e)
	require.NoError(t, err)
	empty.Extra = string(b)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: empty})
	require.NoError(t, err)
	// a listing's own figures stand in place of its class's, by base and quote, in either mode
	for _, mode := range []uint8{modeDoor, modeFront} {
		for _, base1 := range []bool{false, true} {
			p := builderPool([4]bool{true, false, false, false}, false, base1, mode)
			var e Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
			e.Gas = &[2]int64{145_430, 136_333}
			b, err := json.Marshal(e)
			require.NoError(t, err)
			p.Extra = string(b)
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			token0, token1 := int64(145_430), int64(136_333)
			if base1 {
				token0, token1 = token1, token0
			}
			res, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
			require.NoError(t, err)
			assert.Equal(t, token0, res.Gas, "its own figure, token 0 in, mode %d, base token 1 %v", mode, base1)
			res, err = calc(sim, simUSDC, simWETH, big.NewInt(3000e6), nil)
			require.NoError(t, err)
			assert.Equal(t, token1, res.Gas, "its own figure, token 1 in, mode %d, base token 1 %v", mode, base1)
		}
	}
	bad := builderPool([4]bool{true, true, false, false}, false, false, modeDoor)
	e = Extra{}
	require.NoError(t, json.Unmarshal([]byte(bad.Extra), &e))
	e.Gas = &[2]int64{145_430, 0}
	b, err = json.Marshal(e)
	require.NoError(t, err)
	bad.Extra = string(b)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: bad})
	assert.ErrorIs(t, err, ErrInvalidState, "a figure of its own missing")
	other := builderPool([4]bool{true, true, false, false}, false, false, 2)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: other})
	assert.ErrorIs(t, err, ErrInvalidState, "a mode the adapter does not take")
}

// A listing of a class with no measured figure (one reading no price source) is quoted only where its front's config
// names its own gas, in either mode: then its state carries the figures, and in the front's mode its pool's meta
// names the front as the one contract the adapter approves and the adapter's mode word is the front's.
func TestBuilderFrontModeQuotesAListingWithItsOwnGas(t *testing.T) {
	track := func(mode string, g ...ListingGas) entity.Pool {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		cfg := ethConfig(st.URL)
		cfg.Fronts[0].FillMode, cfg.Fronts[0].ListingGas = mode, g
		return ethTrack(t, n, cfg, ethListed(t, n, cfg))
	}
	own := ListingGas{Listing: 1, BaseIn: 141_000, QuoteIn: 133_000}

	dark := track(FillModeFront)
	e := extraOf(t, dark)
	assert.Empty(t, e.Ladders[0], "no figure in the front's mode, none of its own: not quoted")
	assert.Empty(t, e.Ladders[1])
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: dark})
	require.NoError(t, err, "an empty pool")

	lit := track(FillModeFront, own)
	e = extraOf(t, lit)
	require.NotEmpty(t, e.Ladders[0])
	require.NotNil(t, e.Gas)
	assert.Equal(t, [2]int64{141_000, 133_000}, *e.Gas)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: lit})
	require.NoError(t, err)
	res, err := calc(sim, lit.Tokens[0].Address, lit.Tokens[1].Address, e.Ladders[0][0][0].ToBig(), nil)
	require.NoError(t, err)
	base0 := int64(141_000)
	if e.BaseToken1 {
		base0 = 133_000
	}
	assert.Equal(t, base0, res.Gas)
	meta := sim.GetMetaInfo("", "").(PoolMeta)
	assert.Equal(t, modeFront, meta.Mode)
	assert.Equal(t, hexutil.Encode(testFront[:]), meta.ApprovalAddress, "the front is the one contract approved")

	door := track(FillModeDoor)
	e = extraOf(t, door)
	assert.Empty(t, e.Ladders[0], "no figure in the door's mode either, none of its own: not quoted")
	assert.Empty(t, e.Ladders[1])
	named := track(FillModeDoor, own)
	require.NotEmpty(t, extraOf(t, named).Ladders[0])
	require.NotNil(t, extraOf(t, named).Gas)
}

// A pool that holds none of a token: the state says so (Extra.NoneHeld, from the pool's balance read, never from the
// reserves, which a side with no ladder zeroes), and a fill with that token in is charged the write of the pool's
// balance from zero, on that input alone and until a fill of the route pays the pool some.
func TestAPoolHoldingNoneOfTheInputIsChargedItsFirstWrite(t *testing.T) {
	track := func(empty int) entity.Pool {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		cfg := ethConfig(st.URL)
		listed := ethListed(t, n, cfg)
		if empty >= 0 {
			token := common.HexToAddress(listed.Tokens[empty].Address)
			zeroed := 0
			for k := range chain.balances {
				if k[0] == token {
					chain.balances[k] = new(big.Int)
					zeroed++
				}
			}
			require.Positive(t, zeroed, "the chain holds a balance of token %d", empty)
		}
		return ethTrack(t, n, cfg, listed)
	}
	full := track(-1)
	e := extraOf(t, full)
	require.Zero(t, e.NoneHeld)
	require.NotEmpty(t, e.Ladders[0])
	require.NotEmpty(t, e.Ladders[1])
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: full})
	require.NoError(t, err)
	var plain [2]int64
	for in := range 2 {
		res, err := calc(sim, full.Tokens[in].Address, full.Tokens[1-in].Address, e.Ladders[in][0][0].ToBig(), nil)
		require.NoError(t, err)
		plain[in] = res.Gas
	}

	for empty := range 2 {
		p := track(empty)
		e := extraOf(t, p)
		require.Equal(t, uint8(1)<<empty, e.NoneHeld, "token %d", empty)
		require.NotEmpty(t, e.Ladders[empty], "the side that pays the token in is still quoted")
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		in, out := p.Tokens[empty].Address, p.Tokens[1-empty].Address
		size := e.Ladders[empty][0][0].ToBig()
		res, err := calc(sim, in, out, size, nil)
		require.NoError(t, err)
		assert.Equal(t, plain[empty]+17_044, res.Gas, "token %d in, the pool holding none of it", empty)
		// a state that says nothing of it (an older tracker's) charges the class alone
		var old Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &old))
		old.NoneHeld = 0
		b, err := json.Marshal(old)
		require.NoError(t, err)
		assert.NotContains(t, string(b), "noneHeld")
		// once a fill of the route paid the pool the token, the write is no longer from zero
		clone := sim.CloneState().(*PoolSimulator)
		clone.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: in, Amount: size},
			TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})
		assert.False(t, clone.noneHeld[empty])
		assert.True(t, sim.noneHeld[empty], "the clone's fill leaves the state it was cloned from")
	}
}

// On a builder chain a listing's class also says whether neither of its tokens is its pool's asset (token 0), so the
// pool asset's pairs past the hook pair and its other pairs are charged apart, and whether its quote token is the
// asset, so each side is charged by whether its input is the asset. Base's state carries neither.
func TestBuilderMarksAPairWithoutThePoolsAsset(t *testing.T) {
	for _, c := range []struct {
		base, quote               uint8
		past, noAsset, assetQuote bool
	}{{0, 1, false, false, false}, {0, 2, true, false, false}, {2, 0, true, false, true}, {1, 0, true, false, true},
		{2, 1, true, true, false}, {1, 2, true, true, false}} {
		chain := ethChain()
		chain.listings[1].view.BaseIndex, chain.listings[1].view.QuoteIndex = c.base, c.quote
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		cfg := ethConfig(st.URL)
		e := extraOf(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)))
		require.NotEmpty(t, e.Ladders[0])
		assert.Equal(t, c.past, e.Past, "base index %d, quote index %d", c.base, c.quote)
		assert.Equal(t, c.noAsset, e.NoAsset, "base index %d, quote index %d", c.base, c.quote)
		assert.Equal(t, c.assetQuote, e.AssetQuote, "base index %d, quote index %d", c.base, c.quote)

		base := trackedChain()
		base.listings[1].view.BaseIndex, base.listings[1].view.QuoteIndex = c.base, c.quote
		bn := base.node(t, 1234)
		e = extraOf(t, track(t, bn, listed(t, bn)))
		assert.False(t, e.NoAsset, "Base's state")
		assert.False(t, e.AssetQuote, "Base's state")
	}
}

// listed and quoted under any floor, and its pool meta carries no cap. A nonzero cap under the floor is left out.
func TestBuilderTipCapZeroIsNoCap(t *testing.T) {
	chain := ethChain()
	chain.listings[1].view.TipCap = 0
	chain.listings[4].view.TipCap = 0x8001 // 0.1 gwei, under the floor
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	cfg.MinTipCapWei = 1e9
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	var listed []string
	for _, q := range pools {
		listed = append(listed, q.Address)
	}
	assert.Contains(t, listed, door(1), "a cap of 0 passes the floor")
	assert.NotContains(t, listed, door(4), "0.1 gwei is under the floor")

	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	e := extraOf(t, got)
	assert.Zero(t, e.TipCapWei)
	assert.NotEmpty(t, e.Ladders[0], "quoted with no cap")
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	assert.Zero(t, sim.GetMetaInfo("", "").(PoolMeta).TipCapWei)
}

// Each read that runs in the target block checks that it did: a node that ignores the block override on the head read
// alone, or on the sampling read alone, gives no state to quote.
func TestBuilderChecksEachReadRanInTheTargetBlock(t *testing.T) {
	quote, fillable := common.FromHex("0xb6466384"), doorABI.Methods["quoteFillable"].ID
	for name, ignore := range map[string]func([]byte) bool{
		// the head read asks each side's top, and no quote
		"the head read": func(data []byte) bool {
			return bytes.Contains(data, fillable) && !bytes.Contains(data, quote)
		},
		// the pinned sampling read asks the quotes
		"the sampling read": func(data []byte) bool { return bytes.Contains(data, quote) },
	} {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		// the door fills on the map's state in any block, so only the read's own check can tell
		n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64,
			state stateDiffs) (bool, []byte) {
			return chain.builderRespond(to, data, at, 1235, sub, state)
		}
		cfg := ethConfig(st.URL)
		p := ethListed(t, n, cfg)
		require.NotEmpty(t, extraOf(t, ethTrack(t, n, cfg, p)).Ladders[0], "%s: quoted where the node runs it", name)
		n.ignoreOverridesIf = ignore
		unserved(t, ethTrack(t, n, cfg, p), name)
	}
}

// swapTaker is a fill's taker in the tests of Swapped: no door and no contract of ours.
var swapTaker = common.HexToAddress("0x7a7e7a7e7a7e7a7e7a7e7a7e7a7e7a7e7a7e7a7e")

// swappedLog is a fill's Swapped as the deployed contracts log it (IPropAMM's event, built from the embedded ABI):
// four topics, the event's and the sender, tokenIn and tokenOut, over amountIn, amountOut and the recipient.
func swappedLog(t *testing.T, emitter, sender common.Address) types.Log {
	t.Helper()
	ev := cellABI.Events["Swapped"]
	require.Equal(t, crypto.Keccak256Hash([]byte("Swapped(address,address,address,uint256,uint256,address)")), ev.ID)
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(1_000_000), big.NewInt(999_000), swapTaker)
	require.NoError(t, err)
	require.Len(t, data, 96)
	return types.Log{Address: emitter, Topics: []common.Hash{ev.ID, common.BytesToHash(sender[:]),
		common.BytesToHash(testWETH[:]), common.BytesToHash(testUSDC[:])}, Data: data}
}

// On a builder build the filling listing's quote contract logs the fill's one Swapped and no door does. The parser
// maps it to the door it names its sender where that door's listing is one the contract holds; a fill through the
// front's own entries, whose sender is no door, maps to every listing of the contract, as LaneFill does. A door's
// address logs none there, and a Swapped of an address the parser does not know (the topic is every IPropAMM
// venue's) maps to nothing and reads nothing more from the node.
func TestEventParserMapsADesksSwappedOnABuilderChain(t *testing.T) {
	chain := ethChain()
	chain.listings[1].view.Desk = 7
	chain.listings[4].view.Desk = 7
	n := builderNode(t, chain, 900)
	p := NewPoolFactory(ethConfig("http://127.0.0.1:1"), n.client())

	behindDoor := swappedLog(t, testCellA, fakeDoorOf(testFactory, 4))
	direct := swappedLog(t, testCellA, swapTaker)
	got, err := p.Decode(context.Background(), []types.Log{behindDoor, direct})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{
		door(1): {direct},
		door(4): {behindDoor, direct},
	}, got)

	// a door logs no Swapped on a builder build, and another venue's maps to nothing; neither makes the parser
	// learn again
	before := len(n.calls())
	fromDoor := swappedLog(t, fakeDoorOf(testFactory, 4), swapTaker)
	other := swappedLog(t, common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd"),
		fakeDoorOf(testFactory, 4))
	otherDesk := swappedLog(t, testCellB, fakeDoorOf(testFactory, 4))
	got, err = p.Decode(context.Background(), []types.Log{fromDoor, other, otherDesk})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Len(t, n.calls(), before, "no learning for a Swapped of an unknown address")

	// the same log on a storage build is the door's own, and maps to its emitter
	base, err := NewPoolFactory(testConfig(), nil).Decode(context.Background(),
		[]types.Log{swappedLog(t, fakeDoorOf(testFactory, 1), swapTaker)})
	require.NoError(t, err)
	assert.Len(t, base[door(1)], 1)
}

// Listings of two desks, two makers' on one front: the lister finds both from the front's own listing ids and the
// factory's doors, and each is quoted from its own desk contract's entry in the map, with nothing in the config naming
// a desk, a maker or a door. A desk whose update is not in the map is not quoted, and the other desk's still is.
func TestBuilderListsAndQuotesEveryDesksListings(t *testing.T) {
	chain := ethChain()
	second := chain.listings[4]
	second.view.Desk = 9
	second.sides = chain.listings[1].sides
	chain.balances[[2]common.Address{testWETH, testPoolB}] = big.NewInt(9e18)
	chain.balances[[2]common.Address{testUSDC, testPoolB}] = big.NewInt(40_000e6)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	cfg := ethConfig(st.URL)
	target := slotTime(slotAfter(n.time))

	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{
		testCellA: deskSlot0(1235, false), testCellB: deskSlot0(1235, false)})
	for _, id := range []uint32{1, 4} {
		p, _ := ruledTrack(t, n, cfg, id)
		served(t, chain, id, p, target, "a listing of each desk")
	}

	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellB: deskSlot0(1235, false)})
	p, _ := ruledTrack(t, n, cfg, 1)
	unserved(t, p, "desk 7's update is not in the map")
	p, _ = ruledTrack(t, n, cfg, 4)
	served(t, chain, 4, p, target, "desk 9's is")
}

// With no floor configured Ethereum's default stands, 2 gwei: a listing whose tip cap is 1 wei, which every
// transaction that tips at all would revert on, is neither listed nor quoted, one at the default is, and a config
// that names a floor of 1 wei lists every listing. A chain read at the head has no default.
func TestBuilderDefaultTipCapFloor(t *testing.T) {
	ctx := context.Background()
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	require.Zero(t, cfg.MinTipCapWei)
	assert.Equal(t, uint64(2e9), cfg.minTipCapWei())
	p := ethListed(t, n, cfg) // listed at a 2 gwei cap
	require.NotEmpty(t, extraOf(t, ethTrack(t, n, cfg, p)).Ladders[0], "a cap at the default floor")

	chain.listings[1].view.TipCap = 0x0001 // 1 wei
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	for _, q := range pools {
		assert.NotEqual(t, door(1), q.Address, "a 1 wei cap is not listed")
	}
	e := extraOf(t, ethTrack(t, n, cfg, p))
	assert.Equal(t, uint64(1), e.TipCapWei)
	assert.Empty(t, e.Ladders[0], "a 1 wei cap is not quoted")
	assert.Empty(t, e.Ladders[1])

	open := ethConfig(st.URL)
	open.MinTipCapWei = 1
	assert.NotEmpty(t, extraOf(t, ethTrack(t, n, open, p)).Ladders[0], "a configured floor of 1 wei")

	assert.Zero(t, testConfig().minTipCapWei(), "no default on a chain read at the head")
}

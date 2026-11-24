package efficientprop

import (
	"context"
	"errors"
	"maps"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// create3Child is the account a CREATE3 deployment of salt id by deployer lands at: the proxy at CREATE2(deployer, id,
// the proxy's creation code hash), then that proxy's first CREATE. A builder build's price contract of source id is
// its front's child (the front's own address for the fill's call).
func create3Child(deployer common.Address, id uint32) common.Address {
	h := common.HexToHash("0x21c35dbe1b344a2488cf3321d6ce542f8e9f305544ff09e4993a62319a497c1f")
	proxy := crypto.CreateAddress2(deployer, common.BigToHash(big.NewInt(int64(id))), h[:])
	return crypto.CreateAddress(proxy, 1)
}

// stateEntry is one /state entry: each account's slots its update writes.
type stateEntry map[common.Address]map[common.Hash]common.Hash

// setEntries makes the map's target block number at slot's second (no slot where slot is 0), with the entries named.
func (s *stateServer) setEntries(slot, number uint64, entries map[string]stateEntry) {
	m := map[string]any{"blockNumber": hexutil.EncodeUint64(number)}
	if slot != 0 {
		m["slot"] = slot
	}
	for key, e := range entries {
		so := map[string]any{}
		for a, slots := range e {
			sd := map[string]string{}
			for k, v := range slots {
				sd[k.Hex()] = v.Hex()
			}
			so[a.Hex()] = map[string]any{"stateDiff": sd, "balance": "0x0", "nonce": "0x1"}
		}
		m[key] = map[string]any{"stateOverride": so}
	}
	s.mu.Lock()
	s.m = m
	s.mu.Unlock()
}

// printIDs are the print listing's primary and backup source ids, each a builder build's price contract.
var printIDs = [2]uint32{9, 10}

// printSlots are a price contract's two slots as its update writes them.
var printSlots = map[common.Hash]common.Hash{{}: common.HexToHash("0x4001"),
	common.BigToHash(big.NewInt(1)): common.HexToHash("0x4002")}

// cellSlots is the listing's quote contract's pair slot 0 quote word as its update for block target writes it, with
// extra set in its
// unused bits so two answers' entries can be told apart.
func cellSlots(target uint64, carry bool, extra uint) map[common.Hash]common.Hash {
	w := deskSlot0(target, carry)
	if extra != 0 {
		w.SetBit(w, int(extra), 1)
	}
	return map[common.Hash]common.Hash{quoteKey(0): common.BigToHash(w)}
}

// printChain is ethChain's listing 1 bound PRINT to a primary and a backup price contract of the builder build, reading
// its backup where onBackup, the source it reads guarded where guarded.
func printChain(onBackup, guarded bool) *fakeChain {
	chain := ethChain()
	l := chain.listings[1]
	l.view.Scopes = 0 // PRINT
	l.sourceIds = new(big.Int).SetUint64(uint64(printIDs[0]) | uint64(printIDs[1])<<32)
	for k, id := range printIDs {
		e := new(big.Int).SetBytes(create3Child(testFront, id).Bytes())
		e.Or(e, new(big.Int).Lsh(big.NewInt(8), 240))
		if guarded && k == 0 {
			e.SetBit(e, 247, 1)
		}
		chain.sources[id] = e
	}
	// a builder build's live source word: the id it reads (bits 0-31), its window and, on the backup, bit 244
	live := new(big.Int).SetUint64(uint64(printIDs[0]))
	if onBackup {
		live.SetUint64(uint64(printIDs[1])).SetBit(live, 244, 1)
	}
	live.Or(live, new(big.Int).Lsh(big.NewInt(8), 240))
	if guarded {
		live.SetBit(live, 247, 1)
	}
	l.price = live
	return chain
}

// printConfig is ethConfig approving both of printChain's price contracts.
func printConfig(urls ...string) *Config {
	cfg := ethConfig(urls[0])
	cfg.Fronts[0].Titan.URLs = urls
	for _, id := range printIDs {
		a := create3Child(testFront, id)
		cfg.Fronts[0].ApprovedSources = append(cfg.Fronts[0].ApprovedSources, hexutil.Encode(a[:]))
	}
	return cfg
}

// targetCalls are the calls the node ran in block number under a state override.
func targetCalls(n *node, number uint64) []sent {
	var out []sent
	for _, c := range n.calls() {
		if c.number == number && c.state != nil {
			out = append(out, c)
		}
	}
	return out
}

// A builder build's price contract of a source is the front's CREATE3 child of the source's id, as the front calls it.
func TestPriceContractIsTheFrontsCreate3Child(t *testing.T) {
	for id, want := range map[uint32]string{
		1: "0xF166e802fb1994929b522f1ca5410a5f5BD1A366",
		2: "0xCa143657998DbeFb3AC6bb78342B8Cf3Db04dee8",
		3: "0xb9e048a763b0A6e57c8dEA7F53559b39B6c58949",
		4: "0x8f8F0f5c14F11024DBa87D3737Dd1c403F7d8D9b",
	} {
		assert.Equal(t, common.HexToAddress(want), priceContract(testFront, id), "source %d", id)
		assert.Equal(t, common.HexToAddress(want), create3Child(testFront, id), "source %d", id)
	}
}

// A listing that reads a print (PRINT or SESSION) is quoted only where the answer holds the entry of the price contract
// it reads in the target block, its live source (the primary, or the backup once the primary is darkened): without
// it the door would be sampled on the head's print, which is not the one the fill reads.
func TestBuilderPrintListingNeedsItsPriceContractsEntry(t *testing.T) {
	for _, c := range []struct {
		name     string
		onBackup bool
		holds    []uint32
		quoted   bool
	}{
		{"the primary's entry, on the primary", false, []uint32{9}, true},
		{"both entries, on the primary", false, []uint32{9, 10}, true},
		{"no price contract's entry", false, nil, false},
		{"the backup's alone, on the primary", false, []uint32{10}, false},
		{"the backup's, on the backup", true, []uint32{10}, true},
		{"the primary's alone, on the backup", true, []uint32{9}, false},
	} {
		chain := printChain(c.onBackup, false)
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		e := stateEntry{testCellA: cellSlots(1235, false, 0)}
		for _, id := range c.holds {
			e[create3Child(testFront, id)] = printSlots
		}
		st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"a": e})
		cfg := printConfig(st.URL)
		got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
		if !c.quoted {
			unserved(t, got, c.name)
			continue
		}
		ex := extraOf(t, got)
		assert.NotEmpty(t, ex.Ladders[0], c.name)
		assert.NotEmpty(t, ex.Ladders[1], c.name)
	}
}

// Only the accounts a listing reads ride on its calls: its quote contract and its bound price contracts. Every other
// account of the answer, another listing's quote contract, the front or a token among them, is dropped.
func TestBuilderAppliesOnlyTheAccountsItReads(t *testing.T) {
	chain := printChain(false, false)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	stranger := map[common.Hash]common.Hash{{}: common.HexToHash("0x1")}
	st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{
		"a": {testCellA: cellSlots(1235, false, 0), create3Child(testFront, 9): printSlots, testFront: stranger},
		"b": {testCellB: cellSlots(1235, false, 0), testUSDC: stranger, create3Child(testFront, 3): printSlots},
	})
	cfg := printConfig(st.URL)
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	require.NotEmpty(t, extraOf(t, got).Ladders[0])
	calls := targetCalls(n, 1235)
	require.NotEmpty(t, calls)
	read := map[common.Address]bool{testCellA: true, create3Child(testFront, 9): true, create3Child(testFront, 10): true}
	for _, c := range calls {
		assert.Contains(t, c.state, testCellA)
		assert.Contains(t, c.state, create3Child(testFront, 9))
		for a := range c.state {
			assert.True(t, read[a], "an account the listing does not read rides on its call: %s", a.Hex())
		}
	}
}

// A quote contract's entry and its price contracts' come from one answer: of the answers for the highest block, the
// first in the configured url order that holds them all. Slots of one account are never mixed across answers, and a
// later url's answer is taken over an earlier one only where the earlier lacks an entry the listing needs, whole.
func TestBuilderTakesAListingsEntriesFromOneAnswer(t *testing.T) {
	pc := create3Child(testFront, 9)
	full := func(extra uint) stateEntry {
		return stateEntry{testCellA: cellSlots(1235, false, extra), pc: printSlots}
	}

	// the first url answers last, within the merge window: its answer is still the one read
	chain := printChain(false, false)
	n := builderNode(t, chain, 1234)
	first, second := newStateServer(t), newStateServer(t)
	first.delay = 100 * time.Millisecond
	first.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"x": full(100)})
	second.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"x": full(101)})
	cfg := printConfig(first.URL, second.URL)
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	require.NotEmpty(t, extraOf(t, got).Ladders[0])
	want := cellSlots(1235, false, 100)[quoteKey(0)]
	for _, c := range targetCalls(n, 1235) {
		assert.Equal(t, want, c.state[testCellA][quoteKey(0)], "the first url's entry")
	}

	// the first url's answer lacks the price contract's entry: the second url's is read whole, never the first's quote
	// contract beside the second's price contract
	chain = printChain(false, false)
	n = builderNode(t, chain, 1234)
	first, second = newStateServer(t), newStateServer(t)
	first.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"x": {testCellA: cellSlots(1235, false, 100)}})
	second.delay = 50 * time.Millisecond
	second.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"x": full(101)})
	cfg = printConfig(first.URL, second.URL)
	got = ethTrack(t, n, cfg, ethListed(t, n, cfg))
	require.NotEmpty(t, extraOf(t, got).Ladders[0])
	want = cellSlots(1235, false, 101)[quoteKey(0)]
	for _, c := range targetCalls(n, 1235) {
		assert.Equal(t, want, c.state[testCellA][quoteKey(0)], "the second url's entry, whole")
		assert.Contains(t, c.state, pc)
	}

	// two entries of one answer that give the quote contract's slot two values: the account is left out, and the
	// listing with it
	chain = printChain(false, false)
	n = builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"x": full(100), "y": full(101)})
	cfg = printConfig(st.URL)
	unserved(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)), "one answer, two values of a slot")
}

// Two entries of one answer that give an account the same slots but a different balance or nonce split it: the
// balance and nonce are left out (the head's stand). A desk contract's listing, whose own words are slots, is still
// served (the contract takes no value and creates nothing, and no fill reads its balance); a price contract a listing
// needs is not taken from a split answer (TestBuilderPrintListingNeedsItsPriceContractsEntry's neighbours).
func TestBuilderDeskContractServedWhereItsBalanceOrNonceSplits(t *testing.T) {
	for name, second := range map[string][2]string{
		"balance": {"0x1", "0x1"},
		"nonce":   {"0x0", "0x2"},
		"neither": {"0x0", "0x1"},
	} {
		chain := ethChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		slots := map[string]string{quoteKey(0).Hex(): common.BigToHash(deskSlot0(1235, false)).Hex()}
		entry := func(balance, nonce string) map[string]any {
			return map[string]any{"stateOverride": map[string]any{testCellA.Hex(): map[string]any{
				"stateDiff": slots, "balance": balance, "nonce": nonce}}}
		}
		st.mu.Lock()
		st.m = map[string]any{"slot": slotAfter(n.time), "blockNumber": hexutil.EncodeUint64(1235),
			"x": entry("0x0", "0x1"), "y": entry(second[0], second[1])}
		st.mu.Unlock()
		cfg := ethConfig(st.URL)
		got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
		assert.NotEmpty(t, extraOf(t, got).Ladders[0], "the desk contract's balance or nonce split: %s", name)
	}
}

// The merge window starts at the first answer that holds an account, not at the first answer: an empty answer from a
// fast url does not cut off a slower url's answer with the entries, which is read.
func TestBuilderWaitsPastAnEmptyAnswer(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	empty, slow := newStateServer(t), newStateServer(t)
	empty.set(slotAfter(n.time), 1235, nil)
	slow.delay = titan.DefaultMergeWindow + 150*time.Millisecond
	slow.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(empty.URL)
	cfg.Fronts[0].Titan.URLs = []string{empty.URL, slow.URL}
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	assert.NotEmpty(t, extraOf(t, got).Ladders[0], "the slower url's answer, after the empty one")
	assert.Equal(t, int64(1), slow.hits.Load())
}

// A /state url that fails is logged, at most once a minute per url, its key-bearing parts (userinfo and query) left
// out of every field, the error's text included; a url whose call the merge window cancels has not failed and is not
// logged, nor is a url that answers. The listing is still read from the answers that came.
func TestBuilderLogsAFailedURLOnceAMinute(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	good, slow := newStateServer(t), newStateServer(t)
	good.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	slow.delay = titan.DefaultMergeWindow + 500*time.Millisecond
	slow.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	dead := newStateServer(t)
	dead.Close()
	deadURL := strings.Replace(dead.URL, "http://", "http://keyuser:keypass@", 1) + "/v1/1/state?key=SECRETKEY"
	cfg := ethConfig(good.URL)
	cfg.Fronts[0].Titan.URLs = []string{deadURL, good.URL, slow.URL}
	p := ethListed(t, n, cfg)

	var mu sync.Mutex
	logged := map[string]int{}
	tr := ethTracker(t, n, cfg, time.Unix(1_800_000_000, 0))
	tr.warn = func(url string, err error) {
		require.Error(t, err)
		keyless(t, url, err)
		mu.Lock()
		logged[url]++
		mu.Unlock()
	}
	count := func() map[string]int {
		mu.Lock()
		defer mu.Unlock()
		return maps.Clone(logged)
	}
	shown := strings.TrimPrefix(dead.URL, "http://")
	for k, at := range []int64{0, 30, 59, 60, 90, 125} {
		tr.now = func() time.Time { return time.Unix(1_800_000_000+at, 0) }
		got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotEmpty(t, extraOf(t, got).Ladders[0], "read from the url that answered")
		want := map[int]int{0: 1, 1: 1, 2: 1, 3: 2, 4: 2, 5: 3}[k]
		assert.Equal(t, map[string]int{"http://" + shown + "/v1/1/state": want}, count(), "after %d s", at)
	}
	time.Sleep(slow.delay)
	assert.Len(t, count(), 1, "the url the window cancelled is not logged")
}

// The once-a-minute limit is per url: two urls failing together are each logged once, so a second dead url is not
// hidden while the first keeps failing.
func TestBuilderLimitsTheFailedURLLogPerURL(t *testing.T) {
	tr := &PoolTracker{now: func() time.Time { return time.Unix(1_800_000_000, 0) }}
	logged := map[string]int{}
	tr.warn = func(url string, _ error) { logged[url]++ }
	for range 3 {
		tr.stateFailed("http://a/v1/1/state", errors.New("down"))
		tr.stateFailed("http://b/v1/1/state", errors.New("down"))
	}
	assert.Equal(t, map[string]int{"http://a/v1/1/state": 1, "http://b/v1/1/state": 1}, logged)
	tr.now = func() time.Time { return time.Unix(1_800_000_060, 0) }
	tr.stateFailed("http://b/v1/1/state", errors.New("down"))
	assert.Equal(t, map[string]int{"http://a/v1/1/state": 1, "http://b/v1/1/state": 2}, logged, "a minute on")
}

// keyless fails t where a logged url or error names a key-bearing part of the tests' urls: their username, password or
// query.
func keyless(t *testing.T, url string, err error) {
	t.Helper()
	for _, secret := range []string{"keyuser", "keypass", "SECRETKEY", "key="} {
		assert.NotContains(t, url, secret, "the logged url")
		assert.NotContains(t, err.Error(), secret, "the logged error")
	}
}

// A /state url that does not dial (one that does not parse, or a websocket url that does not connect) is logged by the
// tracker as it starts, its userinfo and query left out of the url and of the error, and left out of the reads; the
// urls that dial are read.
func TestBuilderLogsAURLThatDoesNotDialWithoutItsKey(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	good := newStateServer(t)
	good.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	dead := newStateServer(t)
	dead.Close()
	unparsed := "http://keyuser:keypass@127.0.0.1:port/v1/1/state?key=SECRETKEY"
	unreached := strings.Replace(dead.URL, "http://", "ws://keyuser:keypass@", 1) + "/v1/1/state?key=SECRETKEY"
	cfg := ethConfig(good.URL)
	cfg.Fronts[0].Titan.URLs = []string{unparsed, unreached, good.URL}
	var logged []string
	tr, err := newPoolTracker(cfg, n.client(), func(url string, err error) {
		require.Error(t, err)
		t.Logf("logged url=%s error=%v", url, err)
		keyless(t, url, err)
		logged = append(logged, url)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"<a /state url that does not parse>",
		"ws://" + strings.TrimPrefix(dead.URL, "http://") + "/v1/1/state"}, logged)
	require.Len(t, tr.titan[cfg.Fronts[0].Address], 1, "the url that dialled")
	tr.now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	got, err := tr.GetNewPoolState(context.Background(), ethListed(t, n, cfg), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.NotEmpty(t, extraOf(t, got).Ladders[0], "read from the url that dialled")
}

// Only the target block: a map is read only where its block is the head's next. One further ahead (a node a block or
// more behind the api) quotes nothing, even beside an answer for the head's next block or at the head's next slot, and
// so does an answer with no slot, whose time cannot be derived; the next refresh reads the next answer as usual.
func TestBuilderQuotesOnlyTheHeadsNextBlock(t *testing.T) {
	chain := printChain(false, false)
	n := builderNode(t, chain, 1234)
	pc := create3Child(testFront, 9)
	entry := func(target uint64) map[string]stateEntry {
		return map[string]stateEntry{"a": {testCellA: cellSlots(target, false, 0), pc: printSlots}}
	}

	st := newStateServer(t)
	st.setEntries(slotAfter(n.time)+1, 1236, entry(1236))
	cfg := printConfig(st.URL)
	p := ethListed(t, n, cfg)
	unserved(t, ethTrack(t, n, cfg, p), "a map two blocks ahead")
	assert.Empty(t, quotesAfter(n, 0), "no door asked")

	lagging := newStateServer(t)
	lagging.setEntries(slotAfter(n.time), 1235, entry(1235))
	cfg = printConfig(lagging.URL, st.URL)
	unserved(t, ethTrack(t, n, cfg, p), "the highest block is two ahead")

	// the head's next slot, but a block two ahead: the block is checked on its own, not only through the slot
	misnumbered := newStateServer(t)
	misnumbered.setEntries(slotAfter(n.time), 1236, entry(1236))
	cfg = printConfig(misnumbered.URL)
	unserved(t, ethTrack(t, n, cfg, p), "the head's next slot naming a block two ahead")

	// no slot: the answer's time is not derived from another account's slots
	noSlot := newStateServer(t)
	e := entry(1235)
	stamp := new(big.Int).Lsh(new(big.Int).SetUint64(n.time+5), 224)
	e["other"] = stateEntry{testCellB: {{}: common.BigToHash(stamp)}}
	noSlot.setEntries(0, 1235, e)
	cfg = printConfig(noSlot.URL)
	unserved(t, ethTrack(t, n, cfg, p), "an answer with no slot")

	// the next poll reads the head's next block as usual
	st.setEntries(slotAfter(n.time), 1235, entry(1235))
	cfg = printConfig(st.URL)
	assert.NotEmpty(t, extraOf(t, ethTrack(t, n, cfg, p)).Ladders[0], "the next answer for the head's next block")
}

// A map whose second is the head's own is not ahead of it: nothing is quoted.
func TestBuilderQuotesNothingOnAMapAtTheHeadsSecond(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	slot := slotAfter(n.time)
	n.time = slotTime(slot)
	st := newStateServer(t)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	unserved(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)), "the map's second is the head's")
}

// The slot must be the next slot: an answer is read only where its slot is the head's slot plus one, so its second is
// the head's plus 12 s. A later slot, whether a missed slot or an answer ahead of the wall clock, quotes nothing for
// that refresh. It does not back off: the next refresh comes a second later as for any unquoted state, and reads the
// next answer as usual.
func TestBuilderQuotesOnlyTheNextSlot(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	head := (n.time - beaconGenesis) / slotSeconds
	require.Equal(t, slotTime(head), n.time, "the head on the beacon grid")
	st := newStateServer(t)
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	for _, later := range []uint64{2, 3, 6} {
		st.set(head+later, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
		before := len(n.calls())
		got := ethTrack(t, n, cfg, p)
		unserved(t, got, "a slot later than the head's next")
		assert.Empty(t, quotesAfter(n, before), "slot +%d: no door asked", later)
		now := slotTime(head) + 3
		assert.Equal(t, int64(now+1), ethTracker(t, n, cfg, time.Unix(int64(now), 0)).NextUpdateAt(got),
			"slot +%d: the next refresh a second on, no back-off", later)
	}

	// the next poll's answer names the head's next slot: quoted, valid to its second less the allowance
	st.set(head+1, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	e := extraOf(t, ethTrack(t, n, cfg, p))
	require.NotEmpty(t, e.Ladders[0], "the head's next slot")
	assert.Equal(t, slotTime(head+1)-1, e.ValidUntil)

	// a carried word from a later slot is no more read: nothing is quoted past the true target block
	carry := carryChain(2 | 8)
	n = builderNode(t, carry, 1234)
	head = (n.time - beaconGenesis) / slotSeconds
	st.set(head+2, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	cfg = ethConfig(st.URL)
	unserved(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)), "a carried word, a slot later")
}

// The word carries where any of its carry bits (4-7) is set, not bit 4 alone.
func TestBuilderReadsEveryCarryBitOfTheWord(t *testing.T) {
	chain := carryChain(2 | 8)
	chain.listings[1].carryBit = 6
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	cfg := ethConfig(st.URL)
	before := len(n.calls())
	e := extraOf(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)))
	next := 0
	for _, q := range quotesAfter(n, before) {
		if q.number == 1236 {
			next++
		}
	}
	assert.Positive(t, next, "the carried block sampled")
	assert.Equal(t, slotTime(slot)+12-1, e.ValidUntil)
}

// The listing's desk is read twice, at the head and in the pinned read: where they differ (the head moved between the
// reads), nothing is quoted.
func TestBuilderQuotesNothingWhereTheDeskMovedBetweenReads(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64,
		state stateDiffs) (bool, []byte) {
		ok, ret := chain.builderRespond(to, data, at, number, sub, state)
		if m, err := frontABI.MethodById(data[:4]); err == nil && m.Name == "doorView" && number == 1235 && ok {
			vals, err := m.Outputs.Unpack(ret)
			require.NoError(t, err)
			var out doorViewOut
			require.NoError(t, m.Outputs.Copy(&out, vals))
			out.View.Desk = 8
			return true, packOut(frontABI, "doorView", out.View)
		}
		return ok, ret
	}
	unserved(t, ethTrack(t, n, cfg, p), "another desk in the pinned read")
}

// Where the listing and the word both carry but the carried block refuses every size (the word's life ends before it),
// the target block's ladder stands, valid to that block's second less the allowance.
func TestBuilderKeepsTheTargetBlockWhereTheCarriedBlockRefusesAll(t *testing.T) {
	chain := carryChain(2 | 8)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	target := slotTime(slot)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	chain.listings[1].shrinkAt, chain.listings[1].shrinkTo = target+12, 0
	cfg := ethConfig(st.URL)
	e := extraOf(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)))
	assert.Equal(t, target-1, e.ValidUntil, "the target block alone")
	for dir := range 2 {
		require.NotEmpty(t, e.Ladders[dir], "direction %d", dir)
		for _, pt := range e.Ladders[dir] {
			want, err := chain.quoteAt(chain.listings[1], dir, pt[0].ToBig(), target, 1235)
			require.NoError(t, err)
			assert.Equal(t, want, pt[1].ToBig(), "the target block's output")
		}
	}
}

// Ethereum's sources are price contracts, never guarded: a listing bound to a guarded entry is not quoted on chain 1.
func TestBuilderQuotesNothingOnAGuardedEntry(t *testing.T) {
	chain := printChain(false, true)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{
		"a": {testCellA: cellSlots(1235, false, 0), create3Child(testFront, 9): printSlots}})
	cfg := printConfig(st.URL)
	unserved(t, ethTrack(t, n, cfg, ethListed(t, n, cfg)), "a guarded entry on chain 1")
}

// The listings of one front refreshed together share one /state fetch a url: a second listing's refresh inside the
// share window reads the first's answers, and one after it fetches again.
func TestBuilderListingsRefreshedTogetherShareOneStateFetch(t *testing.T) {
	chain := sharedChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	a, b := wordsOf(slotA, 1235, 0, false, false), wordsOf(slotB, 1235, 0, false, false)
	st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"a": {cellOf(7): a}, "b": {cellOf(7): b}})
	cfg := ethConfig(st.URL)
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	now := time.Now()
	tr := ethTracker(t, n, cfg, now)
	tr.now = func() time.Time { return now }
	refresh := func() {
		for _, p := range pools {
			p.Tokens[0].Decimals, p.Tokens[1].Decimals = 18, 6
			got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.NotEmpty(t, extraOf(t, got).Ladders[0], p.Address)
		}
	}
	refresh()
	assert.Equal(t, int64(1), st.hits.Load(), "both listings read one fetch")
	now = now.Add(stateShareWindow - time.Millisecond)
	refresh()
	assert.Equal(t, int64(1), st.hits.Load(), "inside the share window")
	now = now.Add(time.Millisecond)
	refresh()
	assert.Equal(t, int64(2), st.hits.Load(), "the window over: fetched again")
}

// A listing not quoted for a standing reason (its desk not approved) is idle: its state says so and its next refresh
// is asked for idleRefreshEvery on, on a builder chain and on a chain read at the head. One with no ladder for its
// block alone (no answer yet) is not idle and is read again a second on, as a quoted one is.
func TestIdleListingIsRefreshedLessOften(t *testing.T) {
	chain := ethChain()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	now := time.Unix(int64(n.time)+1, 0)
	next := func(cfg *Config) (Extra, int64) {
		tr := ethTracker(t, n, cfg, now)
		got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		return extraOf(t, got), tr.NextUpdateAt(got) - now.Unix()
	}
	e, in := next(cfg)
	require.NotEmpty(t, e.Ladders[0])
	assert.False(t, e.Idle)
	assert.Equal(t, int64(refreshEvery/time.Second), in, "quoted")

	unapproved := ethConfig(st.URL)
	unapproved.ApprovedDesks = nil
	e, in = next(unapproved)
	assert.Empty(t, e.Ladders[0])
	assert.True(t, e.Idle)
	assert.Equal(t, int64(idleRefreshEvery/time.Second), in, "its desk not approved")

	st.set(slotAfter(n.time), 1235, nil)
	e, in = next(cfg)
	assert.Empty(t, e.Ladders[0])
	assert.False(t, e.Idle, "no answer for the next block yet: its next is a block away")
	assert.Equal(t, int64(refreshEvery/time.Second), in)

	// read at the head
	base := trackedChain()
	bn := base.node(t, 1234)
	bp := listed(t, bn)
	none := testConfig()
	none.ApprovedDesks = nil
	tr, err := NewPoolTracker(none, bn.client())
	require.NoError(t, err)
	got, err := tr.GetNewPoolState(context.Background(), bp, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.True(t, extraOf(t, got).Idle)
	assert.Equal(t, got.Timestamp+int64(idleRefreshEvery/time.Second), tr.NextUpdateAt(got))
	quoted := track(t, bn, bp)
	assert.False(t, extraOf(t, quoted).Idle)
	assert.Equal(t, quoted.Timestamp+int64(refreshEvery/time.Second), tr.NextUpdateAt(quoted))
}

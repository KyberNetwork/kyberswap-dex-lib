package efficientprop

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The pair slots of the two listings the tests put on one desk, 7.
const (
	slotA uint8 = 0
	slotB uint8 = 7
)

// sharedChain is ethChain with listing 4 quoted like listing 1 and bound to the same desk, 7, at pair slot 7: both
// listings' words sit in one desk contract, cellOf(7), each at its own pair slot.
func sharedChain() *fakeChain {
	chain := ethChain()
	l1, l4 := chain.listings[1], chain.listings[4]
	l4.sides = l1.sides
	l4.view.Desk = 7
	l1.view.WordIndex, l4.view.WordIndex = slotA, slotB
	chain.balances[[2]common.Address{testWETH, testPoolB}] = big.NewInt(9e18)
	chain.balances[[2]common.Address{testUSDC, testPoolB}] = big.NewInt(40_000e6)
	return chain
}

// quoteWordAt is a pair's quote word as its entry for block target writes it under desk epoch epoch: the stamp, the
// epoch at bits 32-62 and, where carry, the carry bit; firm sets bit 205. Bit 63 stays 0, as the desk contract writes
// it.
func quoteWordAt(target, epoch uint64, carry, firm bool) *big.Int {
	w := deskSlot0(target, carry)
	w.Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(epoch&0x7fffffff), 32))
	if firm {
		w.SetBit(w, quoteFirmBit, 1)
	}
	return w
}

// wordsOf is pair slot i's slots as its entry writes them: the quote word, and a firm word where the quote is firm.
func wordsOf(i uint8, target, epoch uint64, carry, firm bool) map[common.Hash]common.Hash {
	m := map[common.Hash]common.Hash{quoteKey(i): common.BigToHash(quoteWordAt(target, epoch, carry, firm))}
	if firm {
		m[firmKey(i)] = common.HexToHash("0xf1f1")
	}
	return m
}

// joined is the slots of every map given, a later map's value winning.
func joined(ms ...map[common.Hash]common.Hash) map[common.Hash]common.Hash {
	out := map[common.Hash]common.Hash{}
	for _, m := range ms {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ethListedAt is the chain's listing id as the lister records it under cfg.
func ethListedAt(t *testing.T, n *node, cfg *Config, id uint32) entity.Pool {
	t.Helper()
	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	for _, p := range pools {
		if p.Address == door(id) {
			p.Tokens[0].Decimals, p.Tokens[1].Decimals = 18, 6
			return p
		}
	}
	require.FailNow(t, "listing not listed", "listing %d", id)
	return entity.Pool{}
}

// ruledTrack is listing id's state under cfg and the node's calls the refresh sent.
func ruledTrack(t *testing.T, n *node, cfg *Config, id uint32) (entity.Pool, []sent) {
	t.Helper()
	p := ethListedAt(t, n, cfg, id)
	tr := ethTracker(t, n, cfg, time.Now())
	before := len(n.calls())
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	return got, n.calls()[before:]
}

// quotes is the door quotes among calls.
func quotes(calls []sent) []sent {
	var out []sent
	for _, c := range calls {
		if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == "0xb6466384" {
			out = append(out, c)
		}
	}
	return out
}

// served checks that listing id's state on chain is quoted, each point the door's answer in the target block.
func served(t *testing.T, chain *fakeChain, id uint32, p entity.Pool, target uint64, msg string) {
	t.Helper()
	e := extraOf(t, p)
	assert.Equal(t, target-1, e.ValidUntil, msg)
	for dir := range 2 {
		require.NotEmpty(t, e.Ladders[dir], "%s: direction %d", msg, dir)
		for _, pt := range e.Ladders[dir] {
			want, err := chain.quoteAt(chain.listings[id], dir, pt[0].ToBig(), target, 1235)
			require.NoError(t, err)
			assert.Equal(t, want, pt[1].ToBig(), msg)
		}
	}
}

// tracked runs both listings of a shared chain against one answer of entries (a map of entry name to its slots in
// desk 7's contract) and returns each listing's state and the calls its refresh sent.
func trackedBoth(t *testing.T, chain *fakeChain, entries map[string]map[common.Hash]common.Hash) (
	map[uint32]entity.Pool, map[uint32][]sent, uint64) {
	t.Helper()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	es := map[string]stateEntry{}
	for k, v := range entries {
		es[k] = stateEntry{cellOf(7): v}
	}
	st.setEntries(slot, 1235, es)
	cfg := ethConfig(st.URL)
	got, calls := map[uint32]entity.Pool{}, map[uint32][]sent{}
	for _, id := range []uint32{1, 4} {
		got[id], calls[id] = ruledTrack(t, n, cfg, id)
	}
	return got, calls, slotTime(slot)
}

// Two listings of one desk, at pair slots 0 and 7: an answer covers a listing only where it holds that pair's quote
// word at 0x100 + its slot, so an update of pair A is not taken for pair B's, whose door is not even asked. Each
// listing's read carries the desk contract whole from its answer, the other pair's words in it included.
func TestBuilderDeskServesOnlyTheListingWhoseQuoteWordIsWritten(t *testing.T) {
	a, b := wordsOf(slotA, 1235, 0, false, false), wordsOf(slotB, 1235, 0, false, false)
	for _, c := range []struct {
		name           string
		entries        map[string]map[common.Hash]common.Hash
		servesA, servB bool
	}{
		{"pair A's update", map[string]map[common.Hash]common.Hash{"a": a}, true, false},
		{"pair B's update", map[string]map[common.Hash]common.Hash{"b": b}, false, true},
		{"both updates, two entries", map[string]map[common.Hash]common.Hash{"a": a, "b": b}, true, true},
	} {
		chain := sharedChain()
		got, calls, target := trackedBoth(t, chain, c.entries)
		for id, want := range map[uint32]bool{1: c.servesA, 4: c.servB} {
			if want {
				served(t, chain, id, got[id], target, c.name)
				for _, call := range calls[id] {
					if call.number == 1235 && call.state != nil {
						assert.Equal(t, joined(c.entries["a"], c.entries["b"]), call.state[cellOf(7)],
							"%s: listing %d: the desk contract whole", c.name, id)
					}
				}
			} else {
				unserved(t, got[id], c.name)
				assert.Empty(t, quotes(calls[id]), "%s: listing %d's door is not asked", c.name, id)
			}
		}
	}
}

// Pair slot 255, the last: its quote word is at 0x1ff and its firm word at 0x2ff, both pairs' own, and an answer that
// writes them covers it. Slot 0x300 is no pair's.
func TestBuilderDeskServesPairSlot255(t *testing.T) {
	chain := sharedChain()
	chain.listings[4].view.WordIndex = 255
	got, calls, target := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{
		"b": wordsOf(255, 1235, 0, false, true)})
	assert.Equal(t, common.BigToHash(big.NewInt(0x1ff)), quoteKey(255))
	assert.Equal(t, common.BigToHash(big.NewInt(0x2ff)), firmKey(255))
	served(t, chain, 4, got[4], target, "slot 255")
	unserved(t, got[1], "slot 0 is not written")
	assert.Empty(t, quotes(calls[1]))
}

// A stop at the head darkens every listing of the desk, whatever the answer holds: a pair's word never shows a stop
// (its bit 63 is 0), so the stopped bit is read from the head's desk word and never from the pair's word.
func TestBuilderDeskStopAtTheHeadDarkensBothListings(t *testing.T) {
	// a stop raises the epoch: the desk stopped at epoch 4, the words of epoch 3 (before it) and of epoch 4 (the
	// epoch the stop set, which nothing writes under while it stands) both well stamped, bit 63 clear
	for _, word := range []uint64{3, 4} {
		chain := sharedChain()
		chain.deskWords = map[uint32]*big.Int{7: deskWordOf(4, true, 1200)}
		both := map[string]map[common.Hash]common.Hash{"a": wordsOf(slotA, 1235, word, false, false),
			"b": wordsOf(slotB, 1235, word, false, false)}
		for _, w := range both {
			for _, v := range w {
				assert.Zero(t, v.Big().Bit(63), "a pair's word never shows a stop")
			}
		}
		got, calls, _ := trackedBoth(t, chain, both)
		for _, id := range []uint32{1, 4} {
			unserved(t, got[id], "the desk is stopped at the head")
			assert.Empty(t, quotes(calls[id]), "word epoch %d: listing %d: the door is not asked", word, id)
		}
	}
}

// A desk stopped and revived holds a higher epoch: a word written before the stop is of the old epoch and empty, a word
// written under the new one stands. The epoch is read from the head's desk word, all 31 bits, never from the
// pair's own word.
func TestBuilderDeskEpochOfTheWordMustBeTheHeadsAfterAStopAndRevive(t *testing.T) {
	for _, c := range []struct {
		name        string
		head, word  uint64
		servedPairs bool
	}{
		{"a word of the epoch before the revive", 4, 3, false},
		{"a word of the epoch after it", 3, 4, false},
		{"a word of the head's epoch", 4, 4, true},
		{"epoch 0, the head's", 0, 0, true},
		// 16,847,216 is 0x1011170 and 4,464 is 0x1170: equal in their low 16 bits, so a 16-bit tag would take one
		// for the other
		{"the epoch whose low 16 bits are the head's", 16_847_216, 4_464, false},
		{"the head with the low 16 bits of the word's", 4_464, 16_847_216, false},
		{"a past 24-bit epoch, the head's", 16_847_216, 16_847_216, true},
		{"the epoch's top bit", 0x7fffffff, 0x7fffffff, true},
		{"differs in the top bit alone", 0x7fffffff, 0x3fffffff, false},
	} {
		chain := sharedChain()
		chain.deskWords = map[uint32]*big.Int{7: deskWordOf(c.head, false, 0)}
		got, calls, target := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{
			"a": wordsOf(slotA, 1235, c.word, false, false), "b": wordsOf(slotB, 1235, c.word, false, false)})
		for _, id := range []uint32{1, 4} {
			if c.servedPairs {
				served(t, chain, id, got[id], target, c.name)
			} else {
				unserved(t, got[id], c.name)
				assert.Empty(t, quotes(calls[id]), "%s: the door is not asked", c.name)
			}
		}
	}
}

// An answer that writes a slot of a desk contract that is no pair's quote or firm word (slots 0 to 3 and every other)
// covers none of the desk's listings, the pair's own word well stamped beside it; so does a conflict between two
// entries over such a slot. A write of another pair's firm word is a pair's, and harmless.
func TestBuilderDeskForeignSlotWriteDarkensBothListings(t *testing.T) {
	a, b := wordsOf(slotA, 1235, 0, false, false), wordsOf(slotB, 1235, 0, false, false)
	for _, slot := range []int64{0, 1, 2, 3, 4, 0xff, 0x300, 0x301, 0x1000} {
		chain := sharedChain()
		foreign := map[common.Hash]common.Hash{common.BigToHash(big.NewInt(slot)): common.HexToHash("0xbad")}
		got, calls, _ := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{
			"a": a, "b": b, "x": foreign})
		for _, id := range []uint32{1, 4} {
			unserved(t, got[id], "a foreign slot written")
			assert.Empty(t, quotes(calls[id]), "slot %#x: listing %d's door is not asked", slot, id)
		}
	}
	// the same slot given two values by two entries is left out of the override, and still covers none
	chain := sharedChain()
	got, _, _ := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{"a": a, "b": b,
		"x": {common.BigToHash(big.NewInt(1)): common.HexToHash("0x1")},
		"y": {common.BigToHash(big.NewInt(1)): common.HexToHash("0x2")}})
	for _, id := range []uint32{1, 4} {
		unserved(t, got[id], "a foreign slot in conflict")
	}
	// a pair's slots of the quote and firm ranges, of whatever pair, are not foreign
	chain = sharedChain()
	got, _, target := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{"a": a, "b": b,
		"z": {pairSlot(firmWordBase, 99): common.HexToHash("0x1"), pairSlot(quoteWordBase, 99): common.HexToHash("0x2")}})
	served(t, chain, 1, got[1], target, "another pair's words")
}

// Two entries of one answer that give one of pair A's words two values leave pair A unserved, and pair B, on the
// same desk contract, served: the split slot is left out of the contract's override, and the rest kept. Pair A's firm
// word is its own only where its quote word has the firm bit.
func TestBuilderDeskSplitSlotDropsOnlyItsListing(t *testing.T) {
	for name, c := range map[string]struct {
		first, second map[common.Hash]common.Hash
		split         common.Hash
		aServed       bool
	}{
		"pair A's quote word": {wordsOf(slotA, 1235, 0, false, false),
			map[common.Hash]common.Hash{quoteKey(slotA): common.BigToHash(new(big.Int).SetBit(quoteWordAt(1235, 0,
				false, false), 100, 1))}, quoteKey(slotA), false},
		"pair A's firm word, firm": {wordsOf(slotA, 1235, 0, false, true),
			map[common.Hash]common.Hash{firmKey(slotA): common.HexToHash("0xf2f2")}, firmKey(slotA), false},
		"pair A's firm word, no firm bit": {joined(wordsOf(slotA, 1235, 0, false, false),
			map[common.Hash]common.Hash{firmKey(slotA): common.HexToHash("0xf1f1")}),
			map[common.Hash]common.Hash{firmKey(slotA): common.HexToHash("0xf2f2")}, firmKey(slotA), true},
	} {
		chain := sharedChain()
		got, calls, target := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{
			"x": joined(c.first, wordsOf(slotB, 1235, 0, false, false)), "y": c.second})
		if c.aServed {
			served(t, chain, 1, got[1], target, name+": pair A")
		} else {
			unserved(t, got[1], name+": pair A")
			assert.Empty(t, quotes(calls[1]), "%s: pair A's door is not asked", name)
		}
		served(t, chain, 4, got[4], target, name+": pair B")
		read := 0
		for _, call := range calls[4] {
			if call.number != 1235 || call.state == nil {
				continue
			}
			read++
			for k, v := range wordsOf(slotB, 1235, 0, false, false) {
				assert.Equal(t, v, call.state[cellOf(7)][k], "%s: pair B's words", name)
			}
			assert.NotContains(t, call.state[cellOf(7)], c.split, "%s: the split slot left out", name)
		}
		assert.Positive(t, read, name)
	}
}

// A listing whose quote word the answer writes with another block's stamp, older or newer than the target's, is not
// served, and its door is not asked; the other listing of the desk contract, stamped with the target, is.
func TestBuilderDeskOffStampIsNotServed(t *testing.T) {
	for _, stamp := range []uint64{1230, 1233, 1236} {
		chain := sharedChain()
		got, calls, target := trackedBoth(t, chain, map[string]map[common.Hash]common.Hash{
			"a": wordsOf(slotA, stamp, 0, false, false), "b": wordsOf(slotB, 1235, 0, false, false)})
		unserved(t, got[1], "pair A stamped with another block")
		assert.Empty(t, quotes(calls[1]), "stamp %d: pair A's door is not asked", stamp)
		served(t, chain, 4, got[4], target, "pair B stamped with the target")
	}
}

// An update stamped with the block before the target's stands in the target block where the listing and the word
// both carry, as the front's freshness reads it: served, valid to the target block alone, since the word is read in
// no later block. Where the word or the listing does not carry, or the fence kind is NONE, it is not served. The
// carried word is judged under the desk epoch like any other.
func TestBuilderServesAnUpdateCarriedIntoTheTarget(t *testing.T) {
	const wordCarry, wordOnly, none = 2 | 8, 2, 4 | 8 // WORD with a carry life, WORD without, NONE with one
	for _, c := range []struct {
		name   string
		scopes uint8
		carry  bool
		epoch  uint64
		served bool
		asked  bool
	}{
		{"both carry", wordCarry, true, 0, true, true},
		{"the listing alone", wordCarry, false, 0, false, true},
		{"the word alone", wordOnly, true, 0, false, false},
		{"a NONE listing", none, true, 0, false, false},
		{"both carry, the old epoch", wordCarry, true, 2, false, false},
	} {
		chain := carryChain(c.scopes)
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		slot := slotAfter(n.time)
		target := slotTime(slot)
		word := wordsOf(0, 1234, c.epoch, c.carry, false)
		chain.headSlots = map[common.Address]map[common.Hash]common.Hash{cellOf(7): {quoteKey(0): word[quoteKey(0)]}}
		st.setEntries(slot, 1235, map[string]stateEntry{"a": {cellOf(7): word}})
		cfg := ethConfig(st.URL)
		got, calls := ruledTrack(t, n, cfg, 1)
		if !c.served {
			unserved(t, got, c.name)
			if !c.asked {
				assert.Empty(t, quotes(calls), "%s: the door is not asked", c.name)
			}
			continue
		}
		served(t, chain, 1, got, target, c.name)
		for _, q := range quotes(calls) {
			assert.Equal(t, uint64(1235), q.number, "%s: no later block sampled", c.name)
		}
	}
}

// covers, on the per-desk rule: the quote word at 0x100 + i present, of the target block (or the block before, where
// the listing carries), of the head's epoch, the desk not stopped, no slot of the account written that is no pair's
// quote or firm word, and none of the listing's own words split. A desk word that was not read counts no word.
func TestAnswerCoversByThePerDeskRule(t *testing.T) {
	rule := deskRule{cellHash: testCellHash}
	holder := cellOf(7)
	l, ok := rule.slots(testFront, binding{listing: 1, desk: 7, wordIndex: 7}, &holder)
	require.True(t, ok)
	require.Equal(t, cellOf(7), l.account)
	l.deskKnown = true
	live := l
	live.deskWord = common.BigToHash(deskWordOf(3, false, 0))
	live.headQuote, live.headKnown = common.BigToHash(quoteWordAt(1234, 3, false, false)), true
	q := func(n, epoch uint64, firm bool) common.Hash {
		return common.BigToHash(quoteWordAt(n, epoch, false, firm))
	}
	own := quoteKey(7)
	hex := func(n int64) common.Hash { return common.BigToHash(big.NewInt(n)) }
	answerOf := func(slots map[common.Hash]common.Hash, split *splitValues) *answer {
		a := &answer{number: big.NewInt(1235), accounts: map[common.Address]gethclient.OverrideAccount{
			cellOf(7): {StateDiff: slots}}, split: map[common.Address]*splitValues{}}
		if split != nil {
			a.split[cellOf(7)] = split
		}
		return a
	}
	stopped := live
	stopped.deskWord = common.BigToHash(deskWordOf(3, true, 1200))
	unread := live
	unread.deskKnown = false
	for _, c := range []struct {
		name             string
		l                listingSlots
		slots            map[common.Hash]common.Hash
		split            *splitValues
		carryLife        bool
		covered, carried bool
	}{
		{"its word", live, map[common.Hash]common.Hash{own: q(1235, 3, false)}, nil, false, true, false},
		{"pair 6's word", live, map[common.Hash]common.Hash{quoteKey(6): q(1235, 3, false)}, nil, false, false, false},
		{"pair 7's firm word alone", live, map[common.Hash]common.Hash{firmKey(7): hex(1)}, nil, false, false, false},
		{"an old stamp", live, map[common.Hash]common.Hash{own: q(1230, 3, false)}, nil, true, false, false},
		{"the block before, no carry", live, map[common.Hash]common.Hash{own: q(1234, 3, false)}, nil, false, false,
			false},
		{"the block before, carried", live, map[common.Hash]common.Hash{own: q(1234, 3, false)}, nil, true, true, true},
		{"the block before, the head holds another word", withHead(live, quoteWordAt(1200, 3, false, false)),
			map[common.Hash]common.Hash{own: q(1234, 3, false)}, nil, true, false, false},
		{"the block before, the head's word not read", withHeadUnread(live),
			map[common.Hash]common.Hash{own: q(1234, 3, false)}, nil, true, false, false},
		{"a firm quote with no firm word, the head's is firm", withHead(live, quoteWordAt(1234, 3, false, true)),
			map[common.Hash]common.Hash{own: q(1235, 3, true)}, nil, false, true, false},
		{"a firm quote with no firm word, the head's is not firm", live,
			map[common.Hash]common.Hash{own: q(1235, 3, true)}, nil, false, false, false},
		{"a firm quote with no firm word, the head's firm word is of another epoch",
			withHead(live, quoteWordAt(1234, 2, false, true)),
			map[common.Hash]common.Hash{own: q(1235, 3, true)}, nil, false, false, false},
		{"a firm quote with its firm word", live,
			map[common.Hash]common.Hash{own: q(1235, 3, true), firmKey(7): hex(1)}, nil, false, true, false},
		{"carried, the head's firm word differs", withFirm(live, 0xf1), map[common.Hash]common.Hash{
			own: q(1234, 3, true), firmKey(7): hex(0xf2)}, nil, true, false, false},
		{"carried, the head's firm word equal", withFirm(live, 0xf1), map[common.Hash]common.Hash{
			own: q(1234, 3, true), firmKey(7): hex(0xf1)}, nil, true, true, true},
		{"carried, the head's firm word not read", live, map[common.Hash]common.Hash{
			own: q(1234, 3, true), firmKey(7): hex(0xf1)}, nil, true, false, false},
		{"a later stamp", live, map[common.Hash]common.Hash{own: q(1236, 3, false)}, nil, true, false, false},
		{"the epoch before", live, map[common.Hash]common.Hash{own: q(1235, 2, false)}, nil, false, false, false},
		{"the epoch after", live, map[common.Hash]common.Hash{own: q(1235, 4, false)}, nil, false, false, false},
		{"the desk stopped", stopped, map[common.Hash]common.Hash{own: q(1235, 3, false)}, nil, false, false, false},
		{"the desk word not read", unread, map[common.Hash]common.Hash{own: q(1235, 3, false)}, nil, false, false,
			false},
		{"slot 0 written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(0): hex(1)}, nil, false,
			false, false},
		{"the desk word written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(1): hex(1)}, nil,
			false, false, false},
		{"the registry slot written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(2): hex(1)}, nil,
			false, false, false},
		{"the word key slot written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(3): hex(1)}, nil,
			false, false, false},
		{"slot 0xff written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(0xff): hex(1)}, nil,
			false, false, false},
		{"slot 0x300 written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(0x300): hex(1)}, nil,
			false, false, false},
		{"pair 255's words written", live, map[common.Hash]common.Hash{own: q(1235, 3, false), hex(0x1ff): hex(1),
			hex(0x2ff): hex(1)}, nil, false, true, false},
		{"its word split", live, map[common.Hash]common.Hash{own: q(1235, 3, false)},
			&splitValues{slots: map[common.Hash]bool{own: true}}, false, false, false},
		{"its firm word split, firm", live, map[common.Hash]common.Hash{own: q(1235, 3, true)},
			&splitValues{slots: map[common.Hash]bool{firmKey(7): true}}, false, false, false},
		{"its firm word split, no firm", live, map[common.Hash]common.Hash{own: q(1235, 3, false)},
			&splitValues{slots: map[common.Hash]bool{firmKey(7): true}}, false, true, false},
		{"another pair's word split", live, map[common.Hash]common.Hash{own: q(1235, 3, false)},
			&splitValues{slots: map[common.Hash]bool{quoteKey(8): true}}, false, true, false},
		{"a foreign slot split", live, map[common.Hash]common.Hash{own: q(1235, 3, false)},
			&splitValues{slots: map[common.Hash]bool{hex(3): true}}, false, false, false},
		{"a split balance", live, map[common.Hash]common.Hash{own: q(1235, 3, false)},
			&splitValues{slots: map[common.Hash]bool{}, account: true}, false, true, false},
	} {
		covered, carried := answerOf(c.slots, c.split).covers(c.l, c.carryLife)
		assert.Equal(t, c.covered, covered, c.name)
		assert.Equal(t, c.carried, carried, c.name)
	}
	other := live
	other.account = cellOf(9)
	covered, _ := answerOf(map[common.Hash]common.Hash{own: q(1235, 3, false)}, nil).covers(other, false)
	assert.False(t, covered, "another account")
}

// The words and desk words of the desk contract's own vectors (testdata/desk_vectors.json: pair slots 0, 7 and 255,
// desk
// epochs 0, 3 and 16,847,216): each record's word, written at its block, is covered under its desk word; a desk word
// of another epoch, the epoch's low 16 bits alone, or a stop, covers none.
func TestAnswerCoversTheDeskContractsVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/desk_vectors.json")
	require.NoError(t, err)
	var vec struct {
		Records []struct {
			Name        string
			BlockNumber uint64
			DeskEpoch   uint64
			DeskWord    string
			Slot        uint8
			QuoteSlot   string
			QuoteWord   string
			FirmSlot    string
			FirmWord    string
		}
	}
	require.NoError(t, json.Unmarshal(raw, &vec))
	require.Len(t, vec.Records, 3)
	rule := deskRule{cellHash: testCellHash}
	for _, r := range vec.Records {
		holder := cellOf(1)
		l, ok := rule.slots(testFront, binding{listing: 1, desk: 1, wordIndex: r.Slot}, &holder)
		require.True(t, ok, r.Name)
		l.deskKnown = true
		require.Equal(t, common.HexToHash(r.QuoteSlot), l.stampAt, r.Name)
		require.Equal(t, common.HexToHash(r.FirmSlot), l.firmAt, r.Name)
		slots := map[common.Hash]common.Hash{common.HexToHash(r.QuoteSlot): common.HexToHash(r.QuoteWord)}
		if common.HexToHash(r.FirmWord) != (common.Hash{}) {
			slots[common.HexToHash(r.FirmSlot)] = common.HexToHash(r.FirmWord)
		}
		a := &answer{number: new(big.Int).SetUint64(r.BlockNumber), accounts: map[common.Address]gethclient.OverrideAccount{
			cellOf(1): {StateDiff: slots}}, split: map[common.Address]*splitValues{}}
		withHead := func(w common.Hash) bool {
			l.deskWord = w
			ok, _ := a.covers(l, false)
			return ok
		}
		head := common.HexToHash(r.DeskWord)
		assert.True(t, withHead(head), "%s: its desk word", r.Name)
		other := new(big.Int).SetBytes(head[:])
		other.Add(other, new(big.Int).Lsh(big.NewInt(1), 160))
		assert.False(t, withHead(common.BigToHash(other)), "%s: the next epoch", r.Name)
		// the epoch's low 16 bits alone: the same low 16 bits, another epoch
		trap := new(big.Int).SetBytes(head[:])
		trap.Xor(trap, new(big.Int).Lsh(big.NewInt(1), 160+16))
		assert.False(t, withHead(common.BigToHash(trap)), "%s: the same low 16 bits, another epoch", r.Name)
		stop := new(big.Int).SetBytes(head[:])
		assert.False(t, withHead(common.BigToHash(stop.SetBit(stop, deskStoppedBit, 1))), "%s: stopped", r.Name)
	}
}

// parseAnswer merges an answer's entries slot by slot: a slot two entries give two values is left out of its
// account's override and marked split, the account's other slots kept; a balance or nonce two entries give two
// values is left out (the head's stand) and marks the account split as a whole.
func TestParseAnswerSplitsBySlot(t *testing.T) {
	entry := func(balance string, slots map[common.Hash]common.Hash) json.RawMessage {
		sd := map[string]string{}
		for k, v := range slots {
			sd[k.Hex()] = v.Hex()
		}
		b, err := json.Marshal(map[string]any{"stateOverride": map[string]any{cellOf(7).Hex(): map[string]any{
			"stateDiff": sd, "balance": balance, "nonce": "0x1"}}})
		require.NoError(t, err)
		return b
	}
	one, two := common.HexToHash("0x1"), common.HexToHash("0x2")
	qa, fa, qb := quoteKey(slotA), firmKey(slotA), quoteKey(slotB)
	cell := cellOf(7)
	a := parseAnswer(0, map[string]json.RawMessage{"blockNumber": json.RawMessage(`"0x4d3"`),
		"x": entry("0x5", map[common.Hash]common.Hash{qa: one, fa: one, qb: two}),
		"y": entry("0x5", map[common.Hash]common.Hash{fa: two, qb: two})})
	acct := a.accounts[cell]
	assert.Equal(t, map[common.Hash]common.Hash{qa: one, qb: two}, acct.StateDiff)
	assert.Equal(t, big.NewInt(5), acct.Balance)
	assert.Equal(t, uint64(1), acct.Nonce)
	require.NotNil(t, a.split[cell])
	assert.Equal(t, map[common.Hash]bool{fa: true}, a.split[cell].slots)
	assert.False(t, a.split[cell].account)
	assert.False(t, a.whole(cell))

	a = parseAnswer(0, map[string]json.RawMessage{"blockNumber": json.RawMessage(`"0x4d3"`),
		"x": entry("0x5", map[common.Hash]common.Hash{qa: one}),
		"y": entry("0x6", map[common.Hash]common.Hash{qb: two})})
	acct = a.accounts[cell]
	assert.Equal(t, map[common.Hash]common.Hash{qa: one, qb: two}, acct.StateDiff)
	assert.Nil(t, acct.Balance, "the balance left out")
	assert.Zero(t, acct.Nonce, "the nonce left out")
	assert.True(t, a.split[cell].account)

	a = parseAnswer(0, map[string]json.RawMessage{"blockNumber": json.RawMessage(`"0x4d3"`),
		"x": entry("0x5", map[common.Hash]common.Hash{qa: one}),
		"y": entry("0x5", map[common.Hash]common.Hash{qa: one})})
	assert.True(t, a.whole(cell), "two entries that agree")
}

// Each listing is read from the first answer in the url order that covers it, the desk contract whole from that
// answer: where the first url's answer holds pair A's update alone and the second's both, pair A is read from the
// first and pair B from the second, and pair B's read carries none of the first answer's slots.
func TestBuilderDeskTakesEachListingFromOneAnswer(t *testing.T) {
	chain := sharedChain()
	n := builderNode(t, chain, 1234)
	first, second := newStateServer(t), newStateServer(t)
	slot := slotAfter(n.time)
	onlyA := joined(wordsOf(slotA, 1235, 0, false, false), map[common.Hash]common.Hash{firmKey(slotA): id(7)})
	both := joined(wordsOf(slotA, 1235, 0, false, false), map[common.Hash]common.Hash{firmKey(slotA): id(8)},
		wordsOf(slotB, 1235, 0, false, false))
	first.setEntries(slot, 1235, map[string]stateEntry{"a": {cellOf(7): onlyA}})
	second.delay = 50 * time.Millisecond
	second.setEntries(slot, 1235, map[string]stateEntry{"ab": {cellOf(7): both}})
	cfg := ethConfig(first.URL)
	cfg.Fronts[0].Titan.URLs = []string{first.URL, second.URL}

	for id, want := range map[uint32]map[common.Hash]common.Hash{1: onlyA, 4: both} {
		got, calls := ruledTrack(t, n, cfg, id)
		served(t, chain, id, got, slotTime(slot), "listing")
		read := 0
		for _, c := range calls {
			if c.number == 1235 && c.state != nil {
				read++
				assert.Equal(t, want, c.state[cellOf(7)], "listing %d: one answer's desk contract", id)
			}
		}
		assert.Positive(t, read, "listing %d", id)
	}
}

// The desk contract is the CREATE2 child of the front at salt (1 << 252) | desk under the front's CELL_CODEHASH, and
// the front's deskCellOf must name it: where it names another account the listing is not served; where the front
// answers another CELL_CODEHASH than the config names, the refresh fails.
func TestBuilderDeskAccountIsTheDerivedOneUnderTheFrontsCellCodehash(t *testing.T) {
	assert.Equal(t, deskContractOf(testFront, testCellHash, 7), cellOf(7))
	assert.NotEqual(t, cellOf(7), cellOf(8))
	assert.NotEqual(t, cellOf(7), deskContractOf(testFront, common.HexToHash("0x01"), 7))

	chain := ethChain()
	other := common.HexToAddress("0x1234")
	chain.cells = map[uint32]common.Address{7: other}
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.setEntries(slotAfter(n.time), 1235, map[string]stateEntry{"a": {other: {quoteKey(0): common.BigToHash(
		deskSlot0(1235, false))}}})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	unserved(t, ethTrack(t, n, cfg, p), "deskCellOf names an account that is not the derived one")

	chain = ethChain()
	n = builderNode(t, chain, 1234)
	st = newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{cellOf(7): deskSlot0(1235, false)})
	cfg = ethConfig(st.URL)
	p = ethListed(t, n, cfg)
	chain.cellHash = common.HexToHash("0xeb1c76ea") // the front answers another after the listing
	_, err := ethTracker(t, n, cfg, time.Now()).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	assert.ErrorIs(t, err, ErrCellCodehash)
}

// The head's desk word is read from the desk contract's slot 1 at the head's block, in the same batch as the other head
// facts, and from nowhere else: an answer's own slot 1 is not taken for it (the foreign-slot rule).
func TestBuilderDeskWordIsReadFromTheHeadsSlot1(t *testing.T) {
	chain := ethChain()
	chain.deskWords = map[uint32]*big.Int{7: deskWordOf(5, false, 0)}
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{cellOf(7): quoteWordAt(1235, 5, false, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	reads := 0
	inner := n.storage
	n.storage = func(a common.Address, slot common.Hash) common.Hash {
		if a == cellOf(7) && slot == common.BigToHash(big.NewInt(deskWordSlot)) {
			reads++
		}
		return inner(a, slot)
	}
	got := ethTrack(t, n, cfg, p)
	served(t, chain, 1, got, slotTime(slotAfter(n.time)), "the epoch the head holds")
	assert.Equal(t, 1, reads, "the desk word read once")
}

// The event parser learns each listing's desk contract by its front's slot rule: several listings whose words one
// contract holds are each refreshed by its logs (a LaneFill names no pair), the rule's call that they share is sent
// once, and a listing the rule names no slots for is not mapped.
func TestEventParserMapsADeskContractToEveryListingItHolds(t *testing.T) {
	cellCalls := func(n *node, from int) (k int) {
		for _, c := range n.calls()[from:] {
			if c.to == testFront && len(c.data) >= 4 && common.Bytes2Hex(c.data[:4]) ==
				common.Bytes2Hex(frontABI.Methods["deskCellOf"].ID) {
				k++
			}
		}
		return k
	}
	fill := types.Log{Address: cellOf(7), Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	chain := sharedChain()
	n := builderNode(t, chain, 900)
	p := NewPoolFactory(ethConfig("http://127.0.0.1:1"), n.client())
	got, err := p.Decode(context.Background(), []types.Log{fill})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {fill}, door(4): {fill}}, got)
	assert.Equal(t, 1, cellCalls(n, 0), "desk 7's call sent once")

	// a listing whose deskCellOf names an account that is not the derived one is not mapped
	chain = sharedChain()
	chain.cells = map[uint32]common.Address{7: common.HexToAddress("0x1234")}
	n = builderNode(t, chain, 900)
	p = NewPoolFactory(ethConfig("http://127.0.0.1:1"), n.client())
	got, err = p.Decode(context.Background(), []types.Log{fill})
	require.NoError(t, err)
	assert.Empty(t, got)
}

// KT2. The config names no door code hash: a door's address moves with the doors' creation code, so the source reads
// DOOR_CODEHASH() from the deployed factory and derives every door from that. A factory answering another hash than the
// default's gets its doors named under its own, by the lister (the factory's doorOf) and by the parser (a log
// naming a listing), and the hash is read once.
func TestKT2DoorsComeFromTheFactorysOwnCodeHash(t *testing.T) {
	_, has := reflect.TypeOf(Front{}).FieldByName("DoorCodehash")
	assert.False(t, has, "no door code hash in the config")

	chain := sharedChain()
	chain.factoryCodehash = common.HexToHash("0xabcdef01")
	n := builderNode(t, chain, 900)
	cfg := ethConfig("http://127.0.0.1:1")
	d := func(id uint32) string { a := chain.doorAddr(id); return hexutil.Encode(a[:]) }
	require.NotEqual(t, d(1), door(1))

	u, err := NewPoolsListUpdater(cfg, n.client())
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	listed := map[string]bool{}
	for _, p := range pools {
		listed[p.Address] = true
	}
	assert.True(t, listed[d(1)] && listed[d(4)], "the lister's doors are the factory's")

	p := NewPoolFactory(cfg, n.client())
	log := types.Log{Address: testFront,
		Topics: []common.Hash{frontABI.Events["PoolStatus"].ID, id(1)}}
	for range 3 {
		got, err := p.Decode(context.Background(), []types.Log{log})
		require.NoError(t, err)
		assert.Equal(t, map[string][]types.Log{d(1): {log}}, got, "the parser's door is under the factory's hash")
	}
	reads := 0
	for _, c := range n.calls() {
		if c.to == testFactory && len(c.data) >= 4 &&
			common.Bytes2Hex(c.data[:4]) == common.Bytes2Hex(factoryABI.Methods["DOOR_CODEHASH"].ID) {
			reads++
		}
	}
	assert.LessOrEqual(t, reads, 2, "the hash is read once for the lister and once for the parser, not a log")

	// learning (a desk's log) maps the same doors
	push := types.Log{Address: cellOf(7), Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	chain.listings[1].view.Desk, chain.listings[4].view.Desk = 7, 7
	got, err := p.Decode(context.Background(), []types.Log{push})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{d(1): {push}, d(4): {push}}, got)
}

// A factory that answers no door code hash (a revert, or zero) gives the parser no door to name for that log, which
// maps
// nothing, where a hash held in config would have named one regardless.
func TestKT2ParserFailsWhereTheFactoryAnswersNoDoorCodeHash(t *testing.T) {
	chain := sharedChain()
	chain.noDoorHash = true
	n := builderNode(t, chain, 900)
	p := NewPoolFactory(ethConfig("http://127.0.0.1:1"), n.client())
	log := types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["PoolStatus"].ID, id(1)}}
	got, err := p.Decode(context.Background(), []types.Log{log})
	assert.NoError(t, err, "one log maps nothing and the batch goes on")
	assert.Empty(t, got)
}

func withHead(l listingSlots, w *big.Int) listingSlots {
	l.headQuote, l.headKnown = common.BigToHash(w), true
	return l
}

// withFirm is l with the head's quote word the firm word of block 1234, epoch 3, and its firm word v.
func withFirm(l listingSlots, v int64) listingSlots {
	l.headQuote = common.BigToHash(quoteWordAt(1234, 3, false, true))
	l.headFirm, l.headFirmKnown = common.BigToHash(big.NewInt(v)), true
	return l
}

func withHeadUnread(l listingSlots) listingSlots {
	l.headKnown = false
	return l
}

// An update stamped with the block before the target's (the head's own block) stands in the target block only where the
// head landed it: the head's quote word is then the answer's. Where the head holds an older word, a block's fill reads
// that older word, stale under the front's freshness, so the listing is not served and its door is not asked.
func TestBuilderCarriedUpdateMustStandAtTheHead(t *testing.T) {
	const wordCarry = 2 | 8
	for _, c := range []struct {
		name   string
		head   *big.Int // the head's quote word of pair slot 0; nil: unwritten
		served bool
	}{
		{"the head holds it", quoteWordAt(1234, 0, true, false), true},
		{"the head holds an older word", quoteWordAt(1200, 0, true, false), false},
		{"the head holds none", nil, false},
	} {
		chain := carryChain(wordCarry)
		n := builderNode(t, chain, 1234)
		if c.head != nil {
			chain.headSlots = map[common.Address]map[common.Hash]common.Hash{
				cellOf(7): {quoteKey(0): common.BigToHash(c.head)}}
		}
		reads := 0
		inner := n.storage
		n.storage = func(a common.Address, s common.Hash) common.Hash {
			if a == cellOf(7) && s == quoteKey(0) {
				reads++
			}
			return inner(a, s)
		}
		st := newStateServer(t)
		slot := slotAfter(n.time)
		st.setEntries(slot, 1235, map[string]stateEntry{"a": {cellOf(7): wordsOf(0, 1234, 0, true, false)}})
		got, calls := ruledTrack(t, n, ethConfig(st.URL), 1)
		assert.Equal(t, 1, reads, "%s: the head's quote word read once", c.name)
		if c.served {
			served(t, chain, 1, got, slotTime(slot), c.name)
			continue
		}
		unserved(t, got, c.name)
		assert.Empty(t, quotes(calls), "%s: the door is not asked", c.name)
	}
}

// twoFronts is ethConfig with a second front after the first, whose calls the node answers as the first front's.
func twoFronts(t *testing.T, chain *fakeChain, front2 common.Address, factory2 string, cell2 common.Hash) (*node,
	*Config) {
	n := builderNode(t, chain, 900)
	base := n.respondState
	n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64, s stateDiffs) (bool, []byte) {
		if to == front2 {
			to = chain.front
		}
		return base(to, data, at, number, sub, s)
	}
	cfg := ethConfig("http://127.0.0.1:1")
	f2 := cfg.Fronts[0]
	f2.Address, f2.DoorFactory, f2.CellCodehash = hexutil.Encode(front2[:]), factory2, hexutil.Encode(cell2[:])
	cfg.Fronts = append(cfg.Fronts, f2)
	return n, cfg
}

var testFront2 = common.HexToAddress("0x00000000000000000000000000000000000f2f2f")

// One front whose CELL_CODEHASH() is not the config's, or whose door code hash is not answered, leaves only that front
// out of the parser's learning and decoding: the other front's desk and listing logs map as usual.
func TestParserSkipsAFrontWhoseHashFailsAndKeepsTheOthers(t *testing.T) {
	fill := types.Log{Address: cellOf(7), Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	n, cfg := twoFronts(t, sharedChain(), testFront2, hexutil.Encode(testFactory[:]), common.HexToHash("0x02"))
	p := NewPoolFactory(cfg, n.client())
	require.NoError(t, p.cfgErr)
	got, err := p.Decode(context.Background(), []types.Log{fill})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {fill}, door(4): {fill}}, got,
		"the first front's desk log maps to its listings whatever the second front answers")

	n, cfg = twoFronts(t, sharedChain(), testFront2, "0x000000000000000000000000000000000000fac2", testCellHash)
	p = NewPoolFactory(cfg, n.client())
	l1 := types.Log{Address: testFront, Topics: []common.Hash{frontABI.Events["PoolStatus"].ID, id(1)}}
	l2 := types.Log{Address: testFront2, Topics: []common.Hash{frontABI.Events["PoolStatus"].ID, id(1)}}
	got, err = p.Decode(context.Background(), []types.Log{l1, l2})
	require.NoError(t, err, "a front whose factory answers nothing does not fail the batch")
	assert.Equal(t, []types.Log{l1}, got[door(1)])
	assert.Len(t, got, 1, "the failing front's log maps nothing")

	// a desk log of the first front, learned while the second front's factory answers nothing
	fill = types.Log{Address: cellOf(7), Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	p = NewPoolFactory(cfg, n.client())
	got, err = p.Decode(context.Background(), []types.Log{fill})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {fill}, door(4): {fill}}, got,
		"the first front's desk log maps whatever the second front's factory answers")
}

// A hash kept for one front is answered at once while another front's read is slow: the lock covers the maps, never a
// read.
func TestCodeHashReadDoesNotBlockOtherFronts(t *testing.T) {
	n, cfg := twoFronts(t, sharedChain(), testFront2, hexutil.Encode(testFactory[:]), testCellHash)
	slow := n.respondState
	n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64, s stateDiffs) (bool, []byte) {
		if to == testFront2 {
			time.Sleep(400 * time.Millisecond)
		}
		return slow(to, data, at, number, sub, s)
	}
	h := newCodeHashes(n.client())
	_, err := h.cell(context.Background(), cfg.Fronts[0])
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		_, _ = h.cell(context.Background(), cfg.Fronts[1])
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	_, err = h.cell(context.Background(), cfg.Fronts[0])
	took := time.Since(start)
	require.NoError(t, err)
	<-done
	assert.Less(t, took, 100*time.Millisecond, "a front's kept hash is answered at once")
}

// A failed read, or a hash that is not the config's, is kept for a short back-off and not read again before it ends; a
// read after it goes to the node again, and a front that has been set right is then kept.
func TestCodeHashFailureIsKeptForABackoff(t *testing.T) {
	chain := sharedChain()
	chain.cellHash = common.HexToHash("0xeb1c76ea")
	n := builderNode(t, chain, 900)
	cfg := ethConfig("http://127.0.0.1:1")
	h := newCodeHashes(n.client())
	clock := time.Now()
	h.now = func() time.Time { return clock }
	count := func() (k int) {
		for _, c := range n.calls() {
			if c.to == testFront && len(c.data) >= 4 &&
				common.Bytes2Hex(c.data[:4]) == common.Bytes2Hex(frontABI.Methods["CELL_CODEHASH"].ID) {
				k++
			}
		}
		return k
	}
	for range 3 {
		_, err := h.cell(context.Background(), cfg.Fronts[0])
		assert.ErrorIs(t, err, ErrCellCodehash)
	}
	assert.Equal(t, 1, count(), "the mismatch is read once inside the back-off")
	clock = clock.Add(backoff + time.Second)
	chain.cellHash = testCellHash
	v, err := h.cell(context.Background(), cfg.Fronts[0])
	require.NoError(t, err)
	assert.Equal(t, testCellHash, v)
	assert.Equal(t, 2, count(), "read again after the back-off")
	_, err = h.cell(context.Background(), cfg.Fronts[0])
	require.NoError(t, err)
	assert.Equal(t, 2, count(), "kept once right")
}

// A carried update's firm word, like its quote word, stands in the target block only where the head landed it: nothing
// lands the pair in the target block, so the chain's firm word there is the head's. An answer whose carried quote
// word is the head's but whose firm word differs from the head's is not covered; with an equal firm word it is.
func TestBuilderCarriedFirmWordMustBeTheHeads(t *testing.T) {
	const wordCarry = 2 | 8
	word := wordsOf(0, 1234, 0, true, true) // the carried quote word with the firm bit, firm word 0xf1f1
	for _, c := range []struct {
		name   string
		answer common.Hash // the answer's firm word
		served bool
	}{
		{"the head's firm word", word[firmKey(0)], true},
		{"another firm word", common.HexToHash("0xf2f2"), false},
	} {
		chain := carryChain(wordCarry)
		n := builderNode(t, chain, 1234)
		chain.headSlots = map[common.Address]map[common.Hash]common.Hash{
			cellOf(7): {quoteKey(0): word[quoteKey(0)], firmKey(0): word[firmKey(0)]}}
		st := newStateServer(t)
		slot := slotAfter(n.time)
		st.setEntries(slot, 1235, map[string]stateEntry{"a": {cellOf(7): {
			quoteKey(0): word[quoteKey(0)], firmKey(0): c.answer}}})
		got, calls := ruledTrack(t, n, ethConfig(st.URL), 1)
		if c.served {
			served(t, chain, 1, got, slotTime(slot), c.name)
			continue
		}
		unserved(t, got, c.name)
		assert.Empty(t, quotes(calls), "%s: the door is not asked", c.name)
	}
}

// A read the caller cancelled (or whose deadline passed) is not kept as a failure: a live read within the back-off
// reaches the node. A failure that is kept still carries its sentinel under errors.Is.
func TestCancelledHashReadIsNotKept(t *testing.T) {
	chain := sharedChain()
	n := builderNode(t, chain, 900)
	cfg := ethConfig("http://127.0.0.1:1")
	h := newCodeHashes(n.client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.cell(ctx, cfg.Fronts[0])
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "the cancellation stays visible through the sentinel's wrap")
	_, err = h.cell(context.Background(), cfg.Fronts[0])
	assert.NoError(t, err, "a read after a cancelled one goes to the node")
	_, err = h.door(ctx, cfg.Fronts[0])
	require.Error(t, err)
	_, err = h.door(context.Background(), cfg.Fronts[0])
	assert.NoError(t, err, "a door hash read after a cancelled one goes to the node")

	late, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	front2 := cfg.Fronts[0]
	front2.Address = hexutil.Encode(common.HexToAddress("0x00000000000000000000000000000000000f2f2f").Bytes())
	_, err = h.cell(late, front2)
	require.Error(t, err)
	_, err = h.cell(context.Background(), front2)
	assert.False(t, errors.Is(err, context.DeadlineExceeded), "a passed deadline is not kept: the live read ran")
}

// A kept failure carries its sentinel: a hash that is not the config's is ErrCellCodehash, and a factory that answers
// nothing is ErrDoorCodehash, both when read and when answered from the back-off.
func TestKeptHashFailureCarriesItsSentinel(t *testing.T) {
	chain := sharedChain()
	chain.cellHash = common.HexToHash("0xeb1c76ea")
	chain.noDoorHash = true
	n := builderNode(t, chain, 900)
	cfg := ethConfig("http://127.0.0.1:1")
	h := newCodeHashes(n.client())
	for range 2 {
		_, err := h.cell(context.Background(), cfg.Fronts[0])
		assert.ErrorIs(t, err, ErrCellCodehash)
		_, err = h.door(context.Background(), cfg.Fronts[0])
		assert.ErrorIs(t, err, ErrDoorCodehash)
	}
}

// A front whose listings cannot be scanned (its factory's doorOf fails) is left out of the parser's learning like a
// front whose hash fails: the other front's desk log maps as usual.
func TestParserSkipsAFrontWhoseScanFailsAndKeepsTheOthers(t *testing.T) {
	factory2 := common.HexToAddress("0x000000000000000000000000000000000000fac2")
	n, cfg := twoFronts(t, sharedChain(), testFront2, hexutil.Encode(factory2[:]), testCellHash)
	inner := n.respondState
	n.respondState = func(to common.Address, data []byte, at, number uint64, sub int64, s stateDiffs) (bool, []byte) {
		if to == factory2 {
			if m, err := factoryABI.MethodById(data[:4]); err == nil && m.Name == "doorOf" {
				return false, nil
			}
			to = testFactory
		}
		return inner(to, data, at, number, sub, s)
	}
	p := NewPoolFactory(cfg, n.client())
	require.NoError(t, p.cfgErr)
	fill := types.Log{Address: cellOf(7), Topics: []common.Hash{cellABI.Events["LaneFill"].ID}}
	got, err := p.Decode(context.Background(), []types.Log{fill})
	require.NoError(t, err)
	assert.Equal(t, map[string][]types.Log{door(1): {fill}, door(4): {fill}}, got)
}

// rawEntry is a /state entry's JSON for cell with the given slots and the nonce as given (a number is a malformed
// entry: the reader takes the nonce as a string).
func rawEntry(cell common.Address, slots map[common.Hash]common.Hash, nonce any) map[string]any {
	sd := map[string]string{}
	for k, v := range slots {
		sd[k.Hex()] = v.Hex()
	}
	return map[string]any{"stateOverride": map[string]any{cell.Hex(): map[string]any{
		"stateDiff": sd, "balance": "0x0", "nonce": nonce}}}
}

// An answer with any entry that cannot be read covers no listing for its block: what that entry writes is unknown, and
// it may be a write the rules refuse (a desk's slot 1). Here entry a-1 writes pair A's fresh word and a-2 writes the
// desk word under a numeric nonce. The same answer with a well-formed a-2 is refused by the foreign-slot rule, and an
// answer whose only bad entry is another desk's covers nothing too, the rule being the answer's. A later clean answer
// for the block still covers.
func TestAnswerWithAMalformedEntryCoversNoListing(t *testing.T) {
	word := wordsOf(slotA, 1235, 0, false, false)
	deskWrite := map[common.Hash]common.Hash{common.BigToHash(big.NewInt(1)): common.HexToHash("0xbad")}
	for _, c := range []struct {
		name  string
		extra map[string]any // the second entry
		other bool           // the second entry's account is another desk's
	}{
		{"a-2 writes the desk word under a numeric nonce", rawEntry(cellOf(7), deskWrite, 1), false},
		{"a-2 is well formed: the foreign-slot rule refuses", rawEntry(cellOf(7), deskWrite, "0x1"), false},
		{"the only bad entry is another desk's", rawEntry(cellOf(9), word, 1), true},
		{"a stateOverride that is not an object", map[string]any{"stateOverride": "x"}, true},
	} {
		chain := sharedChain()
		n := builderNode(t, chain, 1234)
		st := newStateServer(t)
		slot := slotAfter(n.time)
		st.setEntries(slot, 1235, map[string]stateEntry{"a-1": {cellOf(7): word}})
		st.mu.Lock()
		st.m["a-2"] = c.extra
		st.mu.Unlock()
		got, calls := ruledTrack(t, n, ethConfig(st.URL), 1)
		unserved(t, got, c.name)
		assert.Empty(t, quotes(calls), "%s: the door is not asked", c.name)
	}

	// the same bad entry in one url's answer does not darken a later clean answer for the block
	chain := sharedChain()
	n := builderNode(t, chain, 1234)
	bad, clean := newStateServer(t), newStateServer(t)
	slot := slotAfter(n.time)
	bad.setEntries(slot, 1235, map[string]stateEntry{"a-1": {cellOf(7): word}})
	bad.mu.Lock()
	bad.m["a-2"] = rawEntry(cellOf(7), deskWrite, 1)
	bad.mu.Unlock()
	clean.setEntries(slot, 1235, map[string]stateEntry{"a-1": {cellOf(7): word}})
	cfg := ethConfig(bad.URL)
	cfg.Fronts[0].Titan.URLs = []string{bad.URL, clean.URL}
	got, _ := ruledTrack(t, n, cfg, 1)
	served(t, chain, 1, got, slotTime(slot), "a clean answer for the block, after a malformed one")
}

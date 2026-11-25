package efficientprop

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// On a front read through /state a listing's refresh reads, at the chain's head, things that do not change while the
// head stands (the door's view, the facts batch, the source binding, the controls and implementation words), and then
// samples the door under the state override of the answer for the head's next block. The cache holds, a listing, the
// head-fixed reads under the head's hash and the priced result under the hash and the bytes of the listing's own
// override, so a refresh at the same head asks the node for the override's calls alone, and one at the same head
// under the same override for nothing but the head's hash. It is keyed on the hash, never the number: a head the chain
// replaced at the same height reads again. One entry a listing, held in the tracker's process; a refresh that finds
// the hash changed starts a new entry, and a refresh that cannot read the hash is not cached at all.

// headID is the chain's head as the tracker read it: its number and hash, and its milliseconds where the node gives
// them (BNB Chain's milliTimestamp; 0: not given).
type headID struct {
	number *big.Int
	hash   common.Hash
	ms     uint64
}

// sharedHead is one front's head as last probed. Its lock is held through a probe, so the refreshes of the front's
// listings that come together wait for one probe and read its answer, which nothing writes after. A probe that failed
// is not kept: the next caller probes again.
type sharedHead struct {
	mu   sync.Mutex
	at   time.Time
	id   headID
	held bool
}

// headProbe is the chain's head, number and hash, for front f: the latest block read from the node, or the last probe's
// where that is under stateEvery old (the window the front's /state read is shared over: 250 ms, 100 ms on BNB Chain),
// so the listings of one front refreshed together ask the node once. Sharing delays seeing a new head by up to that
// window, which does no harm: a state's validity ends on the wall clock (ValidUntilMs less the allowance), the
// allowance before its target block, so a state read at a head the chain has already moved past is refused by then; and
// an answer for the new head's next block is not the held head's next, so nothing is quoted from it. ok is false where
// the node does not answer a block.
func (t *PoolTracker) headProbe(ctx context.Context, f *Front) (id headID, ok bool) {
	s := t.heads[f.Address]
	if s == nil || t.stateEvery <= 0 {
		return t.readHead0(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if age := t.now().Sub(s.at); s.held && age >= 0 && age < t.stateEvery {
		return s.id, true
	}
	if id, ok = t.readHead0(ctx); ok && ctx.Err() == nil {
		s.id, s.at, s.held = id, t.now(), true
	}
	return id, ok
}

// readHead0 reads the chain's latest block's number and hash from the node, and its milliseconds where it gives them
// in its second (milliTimestamp, beside timestamp).
func (t *PoolTracker) readHead0(ctx context.Context) (id headID, ok bool) {
	var res struct {
		Number *hexutil.Big    `json:"number"`
		Hash   common.Hash     `json:"hash"`
		Time   *hexutil.Uint64 `json:"timestamp"`
		Ms     *hexutil.Uint64 `json:"milliTimestamp"`
	}
	if err := t.ethrpcClient.GetETHClient().Client().CallContext(ctx, &res, "eth_getBlockByNumber", "latest",
		false); err != nil || res.Number == nil || res.Hash == (common.Hash{}) {
		return id, false
	}
	id = headID{number: res.Number.ToInt(), hash: res.Hash}
	if res.Ms != nil && res.Time != nil && uint64(*res.Ms)/1000 == uint64(*res.Time) {
		id.ms = uint64(*res.Ms)
	}
	return id, true
}

// builderFacts are the head-fixed reads of readBuilderHead: the listing's doorView (and whether it answered), the
// multicall contract, the head's second, the listing's slots, and its source ids and live source words where read.
type builderFacts struct {
	at        doorViewOut
	atOK      bool
	block     *big.Int
	multicall common.Address
	headTime  *big.Int
	// headMs is the head's milliseconds where the probe that named this head read them (headID.ms; 0: not read)
	headMs    uint64
	slots     listingSlots
	slotted   bool
	ids, live *big.Int
}

// probeFixed are the words probe reads beside its multicall that the head fixes: the listing's controls, the front's
// implementation, and where asked the live source and session words. quoted and session say which were asked.
type probeFixed struct {
	quoted, session, lanes bool
	ctl, impl, live, ses   hexutil.Bytes
}

// pricedOut is a refresh's result under one override: the pool's fields the refresh writes, and what the gate that
// stands outside the reads said (the desk's approval).
type pricedOut struct {
	extra       string
	reserves    entity.PoolReserves
	blockNumber uint64
	timestamp   int64
	desk        uint32
	approved    bool
}

// headEntry is one listing's cache at one head.
type headEntry struct {
	id headID

	mu       sync.Mutex
	facts    *builderFacts
	srcSet   bool
	srcOK    bool
	srcLanes []common.Address
	fixed    *probeFixed
	key      string
	priced   *pricedOut
}

// listingCache is the tracker's cache, an entry a listing door.
type listingCache struct {
	mu sync.Mutex
	m  map[common.Address]*headEntry
}

func newListingCache() *listingCache { return &listingCache{m: map[common.Address]*headEntry{}} }

// at is door's entry at head id, a new one (replacing any of another head) where it holds none.
func (c *listingCache) at(door common.Address, id headID) *headEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.m[door]; e != nil && e.id.hash == id.hash {
		return e
	}
	e := &headEntry{id: id}
	c.m[door] = e
	return e
}

func (e *headEntry) getFacts() *builderFacts {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.facts
}

func (e *headEntry) setFacts(f *builderFacts) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.facts = f
	e.mu.Unlock()
}

func (e *headEntry) getSources() (ok bool, lanes []common.Address, held bool) {
	if e == nil {
		return false, nil, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.srcOK, slices.Clone(e.srcLanes), e.srcSet
}

func (e *headEntry) setSources(ok bool, lanes []common.Address) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.srcOK, e.srcLanes, e.srcSet = ok, slices.Clone(lanes), true
	e.mu.Unlock()
}

// getFixed is the held probe words where they were read for the same asks.
func (e *headEntry) getFixed(quoted, session, lanes bool) *probeFixed {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if f := e.fixed; f != nil && f.quoted == quoted && f.session == session && f.lanes == lanes {
		return f
	}
	return nil
}

func (e *headEntry) setFixed(f *probeFixed) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.fixed = f
	e.mu.Unlock()
}

func (e *headEntry) getPriced(key string) *pricedOut {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.priced != nil && e.key == key {
		return e.priced
	}
	return nil
}

func (e *headEntry) setPriced(key string, out *pricedOut) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.key, e.priced = key, out
	e.mu.Unlock()
}

// overrideKey is the bytes of the listing's own override in a form that names it whole: its target block's number and
// second, and each account with its balance, nonce and slots in order. "none" where the head is read with no override.
func overrideKey(o override) string {
	if o.time == 0 || o.state == nil {
		return "none"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d:%d", o.number, o.time)
	accounts := make([]common.Address, 0, len(o.state.Overrides))
	for a := range o.state.Overrides {
		accounts = append(accounts, a)
	}
	slices.SortFunc(accounts, func(x, y common.Address) int { return strings.Compare(x.Hex(), y.Hex()) })
	for _, a := range accounts {
		acct := o.state.Overrides[a]
		writeAccount(&b, a, acct)
	}
	return b.String()
}

func writeAccount(b *strings.Builder, a common.Address, acct gethclient.OverrideAccount) {
	fmt.Fprintf(b, "|%s,%v,%d", a.Hex(), acct.Balance, acct.Nonce)
	slots := make([]common.Hash, 0, len(acct.StateDiff))
	for s := range acct.StateDiff {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Hex() < slots[j].Hex() })
	for _, s := range slots {
		fmt.Fprintf(b, ",%s=%s", s.Hex(), acct.StateDiff[s].Hex())
	}
}

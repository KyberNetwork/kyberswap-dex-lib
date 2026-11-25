package efficientprop

import (
	"context"
	"errors"
	"math"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// slotSeconds is the time between two of a builder chain's blocks (Ethereum's slot), s: the block after a /state
// answer's target block, where a carried word may still fill, is that much later.
const slotSeconds = 12

// beaconGenesis is the beacon chain's genesis second: a /state answer's slot s names the block of second
// beaconGenesis + slotSeconds * s.
const beaconGenesis = 1_606_824_023

// stateMethod is the JSON-RPC method a /state url answers.
const stateMethod = "titan_getPammStateOverrides"

// fenceNone is the fence kind NONE (scopes bits 1-2), whose word the front never carries into a later block.
const fenceNone = 2

// create3ProxyCodehash is the hash of the proxy creation code the front's CREATE3 deploys through: a builder build's
// price contract of source id is the front's CREATE3 child of id (priceContract).
var create3ProxyCodehash = common.HexToHash("0x21c35dbe1b344a2488cf3321d6ce542f8e9f305544ff09e4993a62319a497c1f")

// priceContract is the contract a builder build's front calls for source id, and that its entry names: the front's
// CREATE3 child of salt id, the proxy at CREATE2(front, id, create3ProxyCodehash), then that proxy's first CREATE.
func priceContract(front common.Address, id uint32) common.Address {
	proxy := crypto.CreateAddress2(front, common.BigToHash(new(big.Int).SetUint64(uint64(id))),
		create3ProxyCodehash[:])
	return crypto.CreateAddress(proxy, 1)
}

// answer is one url's /state answer: its place in the front's url list (rank), its target block (nil where it names
// none), the second its beacon slot names (0 where it names none), and each account's override. Where two of its
// entries give an account different values, those values are left out of the override (a slot, or its balance and
// nonce) and the account is split (split): a listing whose own slots split is not read from the answer (covers), and
// any other account split anywhere is left out whole where a listing needs or carries it (whole).
type answer struct {
	rank     int
	number   *big.Int
	time     uint64
	accounts map[common.Address]gethclient.OverrideAccount
	split    map[common.Address]*splitValues
	// malformed: an entry of the answer could not be read, so what it writes is unknown and the answer covers no
	// listing: a write it holds, a desk's slot 1 among them, may be one the rules refuse
	malformed bool
}

// splitValues is what two entries of one answer gave an account different values of: those slots, and whether its
// balance or nonce.
type splitValues struct {
	slots   map[common.Hash]bool
	account bool
}

// whole is whether a holds account with none of its values split.
func (a *answer) whole(account common.Address) bool {
	_, held := a.accounts[account]
	return held && a.split[account] == nil
}

// covers is whether a holds listing l's desk update for its target block: it writes l's stamp slot on l's account,
// with a stamp the front takes in that block, and no two of its entries gave one of l's own slots two values (nor,
// where the whole account is l's, its balance or nonce). The front takes the target block's stamp (its low 32 bits),
// and the block before's where the listing carries a word one block (carryLife, the listing's half of carries): that
// update is carried, and the caller checks the word's own carry once it reads the word in the target block.
func (a *answer) covers(l listingSlots, carryLife bool) (ok, carried bool) {
	acct, held := a.accounts[l.account]
	if a.malformed || !held || a.number == nil || !a.number.IsUint64() {
		return false, false
	}
	w, wrote := acct.StateDiff[l.stampAt]
	if !wrote {
		return false, false
	}
	// a slot of the account no answer may write (every one but a pair's quote and firm words), written or split,
	// covers none of its listings
	for slot := range acct.StateDiff {
		if l.foreign(slot) {
			return false, false
		}
	}
	if s := a.split[l.account]; s != nil {
		for slot := range s.slots {
			if l.foreign(slot) || l.owns(slot, w) {
				return false, false
			}
		}
	}
	if !l.fresh(w) {
		return false, false
	}
	// a firm quote the answer gives no firm word for takes the head's: it counts only where the head's quote word is
	// of the desk's epoch and firm too
	if _, wroteFirm := acct.StateDiff[l.firmAt]; firm(w) && !wroteFirm {
		if !l.headKnown || !firm(l.headQuote) || !l.fresh(l.headQuote) {
			return false, false
		}
	}
	stamp, target := uint64(lowStamp(w)), a.number.Uint64()&0xffffffff
	switch {
	case stamp == target:
		return true, false
	case carryLife && stamp+1 == target:
		// a word stamped with the block before the target's stands in the target block only where that block, the
		// head, landed it: the head's own quote word is the answer's
		// and where the answer writes the firm word too, that is the head's as well
		if firmW, wroteFirm := acct.StateDiff[l.firmAt]; wroteFirm && (!l.headFirmKnown || l.headFirm != firmW) {
			return false, false
		}
		if l.headKnown && l.headQuote == w {
			return true, true
		}
		return false, false
	}
	return false, false
}

type statePayload struct {
	StateOverride map[string]stateAccount `json:"stateOverride"`
}

type stateAccount struct {
	StateDiff map[string]string `json:"stateDiff"`
	Balance   string            `json:"balance"`
	Nonce     string            `json:"nonce"`
}

// stateClient is one of a front's /state urls, as a log names it (shownURL), and its client.
type stateClient struct {
	url    string
	client *rpc.Client
}

// fetchAnswers asks each of a front's /state urls (clients, in the config's order) for its answer, as Kyber's titan
// client does: in parallel, until each has answered or failed, or until titan.DefaultMergeWindow after the first that
// holds an account (an empty answer does not start the window: urls often disagree on which entries they carry), or
// the timeout. The one difference from the client is the timeout's default, defaultStateTimeout (1 s) where the
// client's is titan.DefaultTimeout (10 s): the fetch holds the front's lock. It answers them in the url order, never
// merged. A
// url whose call fails is passed to failed, before its answer is counted; one the window or the caller cancelled
// (context.Canceled) has not failed. titan.FetchState is not called in its place: it merges the urls' answers into one
// override map, and a listing here is read from one answer whole (pick), with the slots an answer's entries disagree
// on marked (parseAnswer).
func fetchAnswers(ctx context.Context, clients []stateClient, timeout time.Duration,
	failed func(url string, err error)) []answer {
	if len(clients) == 0 {
		return nil
	}
	if timeout == 0 {
		timeout = defaultStateTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type got struct {
		rank   int
		result map[string]json.RawMessage
	}
	results := make(chan got, len(clients))
	for k, c := range clients {
		go func() {
			var result map[string]json.RawMessage
			if err := c.client.CallContext(ctx, &result, stateMethod); err != nil {
				if failed != nil && !errors.Is(err, context.Canceled) {
					failed(c.url, err)
				}
				result = nil
			}
			results <- got{rank: k, result: result}
		}()
	}
	var answers []answer
	var window <-chan time.Time
collect:
	for range clients {
		select {
		case g := <-results:
			if g.result == nil {
				continue
			}
			a := parseAnswer(g.rank, g.result)
			if window == nil && len(a.accounts) > 0 {
				window = time.After(titan.DefaultMergeWindow)
			}
			answers = append(answers, a)
		case <-window:
			break collect
		case <-ctx.Done():
			break collect
		}
	}
	slices.SortFunc(answers, func(a, b answer) int { return a.rank - b.rank })
	return answers
}

// parseAnswer is a /state result as one answer: its blockNumber, its slot's second, and every entry's accounts (each
// entry's stateOverride, an account with no slot skipped as Kyber's titan client skips it), merged slot by slot. A
// slot two entries give different values is left out of its account's override, and a balance or nonce two entries
// give different values leaves both out (the head's stand); the account is marked split with them.
func parseAnswer(rank int, result map[string]json.RawMessage) answer {
	a := answer{rank: rank, accounts: map[common.Address]gethclient.OverrideAccount{},
		split: map[common.Address]*splitValues{}}
	var number string
	if raw, ok := result["blockNumber"]; ok && json.Unmarshal(raw, &number) == nil {
		if n, ok := new(big.Int).SetString(strings.TrimPrefix(number, "0x"), 16); ok {
			a.number = n
		}
	}
	var slot uint64
	if raw, ok := result["slot"]; ok && json.Unmarshal(raw, &slot) == nil && slot > 0 &&
		slot <= (math.MaxUint64-beaconGenesis)/slotSeconds {
		a.time = beaconGenesis + slotSeconds*slot
	}
	mark := func(account common.Address) *splitValues {
		s := a.split[account]
		if s == nil {
			s = &splitValues{slots: map[common.Hash]bool{}}
			a.split[account] = s
		}
		return s
	}
	for key, raw := range result {
		if key == "slot" || key == "blockNumber" {
			continue
		}
		// a member that is not an object (the builder's `timestamp`, a string, an array, null) or names no stateOverride
		// is passed over, as Kyber's titan client passes it; one that names a stateOverride it cannot read whole makes
		// the answer malformed
		var member map[string]json.RawMessage
		if json.Unmarshal(raw, &member) != nil {
			continue
		}
		override, named := member["stateOverride"]
		if !named {
			continue
		}
		var p statePayload
		if json.Unmarshal(override, &p.StateOverride) != nil {
			a.malformed = true
			continue
		}
		for addr, sd := range p.StateOverride {
			if len(sd.StateDiff) == 0 {
				continue
			}
			account := common.HexToAddress(addr)
			balance, nonce := common.HexToHash(sd.Balance).Big(), common.HexToHash(sd.Nonce).Big().Uint64()
			acct, seen := a.accounts[account]
			if !seen {
				acct = gethclient.OverrideAccount{StateDiff: make(map[common.Hash]common.Hash, len(sd.StateDiff)),
					Balance: balance, Nonce: nonce}
			} else if acct.Balance.Cmp(balance) != 0 || acct.Nonce != nonce {
				mark(account).account = true
			}
			for s, v := range sd.StateDiff {
				k, val := common.HexToHash(s), common.HexToHash(v)
				if old, ok := acct.StateDiff[k]; ok && old != val {
					mark(account).slots[k] = true
				}
				acct.StateDiff[k] = val
			}
			a.accounts[account] = acct
		}
	}
	for account, s := range a.split {
		acct := a.accounts[account]
		for k := range s.slots {
			delete(acct.StateDiff, k)
		}
		if s.account {
			acct.Balance, acct.Nonce = nil, 0
		}
		a.accounts[account] = acct
	}
	return a
}

// pick is the answer a listing is read from: of the answers for the highest block any names, the first in the url
// order that covers the listing's desk update (l, its slots; carryLife, whether the listing carries a word: covers)
// and holds every account of need whole; nil where none does. carried is whether the update it covers is stamped
// with the block before the target's. A listing's accounts are never taken from two answers.
func pick(answers []answer, l listingSlots, carryLife bool, need ...common.Address) (a *answer, carried bool) {
	var top *big.Int
	for _, a := range answers {
		if a.number != nil && (top == nil || a.number.Cmp(top) > 0) {
			top = a.number
		}
	}
	for k := range answers {
		a := &answers[k]
		if a.number == nil || a.number.Cmp(top) != 0 {
			continue
		}
		ok, carried := a.covers(l, carryLife)
		for _, account := range need {
			ok = ok && a.whole(account)
		}
		if ok {
			return a, carried
		}
	}
	return nil, false
}

// headRead is a listing's head read: its doorView, its pool's balances of both tokens, the front's clock, each side's
// top (the door's quoteFillable of the most the front takes) and the words its rung boundaries are read from (rungs),
// result saying which of the seven answered, the block they were read at and the multicall that read them (nil where
// the node's answer did not name it). On a front read through /state, carried is whether the listing's update in the
// answer is stamped with the block before the target's, which a fill reads in the target block alone
// (readBuilderHead).
type headRead struct {
	view      doorViewOut
	balances  [2]*big.Int
	clock     frontClock
	tops      [2]fillableOut
	rungs     rungRead
	result    []bool
	block     *big.Int
	multicall *common.Address
	carried   bool
	// ce is the listing's cache entry at the head the refresh read (nil: none), key the bytes of its override, and held
	// what was priced under both before: the refresh then reads nothing more
	ce   *headEntry
	key  string
	held *pricedOut
	// sourceIds is the listing's source ids word (record word 10) as the facts batch read it at block, nil where it
	// was not read: sourcesApproved takes it in place of a read of its own
	sourceIds *big.Int
}

// headReads is how many calls a head read makes (headCalls).
const headReads = 7

// headCalls are a listing's seven head reads, in headRead's order.
func headCalls(front, door common.Address, listing uint32, tokens [2]common.Address, custody common.Address,
	maxIn *big.Int) []call {
	calls := []call{{abi: &frontABI, target: front, method: "doorView", params: []any{big.NewInt(int64(listing))}}}
	for _, token := range tokens {
		calls = append(calls, call{abi: &abi.Erc20ABI, target: token, method: "balanceOf", params: []any{custody}})
	}
	calls = append(calls, call{abi: &frontABI, target: front, method: "clock"})
	for dir := range 2 {
		calls = append(calls, call{abi: &doorABI, target: door, method: "quoteFillable",
			params: []any{tokens[dir], tokens[1-dir], maxIn}})
	}
	return append(calls, rungCall(front, listing))
}

// readBuilderHead is the head read of a listing of a front read through /state (f, a builder chain's), in three steps:
//
//   - the /state answers, fetched while the head's block and the multicall are read (by a doorView at the head, which
//     names the listing's desk);
//   - at the head's block, its second, the listing's slots (its front's slot rule: the desk's quote contract, the
//     listing's own slots in it and its stamp's slot), the listing's bound source ids and the source it reads now
//     (record words 10 and 2), each bound id's price contract its front's CREATE3 child;
//   - the listing's head read pinned to the head's block, run in the answer's target block, at its second and on its
//     overrides (o), beside the slot rule's read again.
//
// The answer is the one pick takes: of the highest block's, the first in the url order that covers the listing's
// update (it writes the listing's stamp slot with the target block's stamp, or the block before's where the listing
// carries, and none of the listing's own slots split in it) and, on a listing that reads a print (PRINT or SESSION),
// holds the price contract it reads now. A quote contract's other slots in the answer do not make it cover the
// listing: another listing's update on the same contract is not this one's. Of that answer o carries only the quote
// contract, whole (any other listing's slots of it in the answer with it, a split slot left out), and the bound price
// contracts, each where nothing of it split: no other account rides on the listing's calls. mapped is whether the
// listing is quoted on that state: the answer's block is the head's next, its slot is the head's slot plus one (its
// second the head's plus slotSeconds), the node ran the read in that block, the desk and its slots are the ones the
// head named, and a carried update's word carries (carries). A later slot (a missed slot, or an answer ahead of the
// wall clock) would sample the door at a second its fill does not run at, so it quotes nothing for that refresh, and
// the next refresh reads the next answer as usual. Where no answer is such, the read runs at the head with no
// override, for the reserves alone, and o is zero. A node that refuses a read fails the refresh.
func (t *PoolTracker) readBuilderHead(ctx context.Context, f *Front, p entity.Pool, se StaticExtra,
	tokens [2]common.Address, maxIn *big.Int, skipHeld bool) (h headRead, o override, mapped bool, err error) {
	fetched := make(chan []answer, 1)
	go func() { fetched <- t.stateAnswers(ctx, f) }()
	front, door := common.HexToAddress(se.Front), common.HexToAddress(p.Address)
	// the chain's head, by hash, and what is held of this listing at it
	hd, probed := t.headProbe(ctx, f)
	var ce *headEntry
	if probed {
		ce = t.cache.at(door, hd)
	}
	bf := ce.getFacts()
	if bf == nil {
		var cacheable bool
		bf, cacheable, err = t.readBuilderFacts(ctx, f, se, front, hd, probed)
		if err != nil {
			return h, o, false, err
		}
		if cacheable {
			ce.setFacts(bf)
		} else {
			ce = nil
		}
	}
	answers := <-fetched
	h.block = bf.block
	multicall := bf.multicall
	h.multicall = &multicall
	h.ce = ce
	at := bf.at
	desk := uint32(0)
	if bf.atOK {
		desk = at.View.Desk
	}
	rule, err := t.slotRule(ctx, f)
	if err != nil {
		return h, o, false, err
	}
	b := binding{listing: se.Listing, desk: desk, wordIndex: at.View.WordIndex}
	headTime, slots, slotted := bf.headTime, bf.slots, bf.slotted
	keep := []common.Address{slots.account}
	if bf.ids != nil {
		h.sourceIds = bf.ids
		for _, id := range []uint64{bits(bf.ids, 0, 32), bits(bf.ids, 32, 32)} {
			if id != 0 {
				keep = append(keep, priceContract(front, uint32(id)))
			}
		}
	}
	var need []common.Address
	if printed(at.View.Scopes) {
		// a builder build's live source word names the id it reads at bits 0-31 (0: none, a dark listing)
		reads := common.Address{}
		if bf.live != nil {
			if id := bits(bf.live, 0, 32); id != 0 {
				reads = priceContract(front, uint32(id))
			}
		}
		need = append(need, reads)
		if !slices.Contains(keep, reads) {
			keep = append(keep, reads)
		}
	}
	var a *answer
	carried := false
	if slotted && desk != 0 {
		a, carried = pick(answers, slots, carryLife(at.View.Scopes), need...)
	}
	ahead := a != nil && !slices.Contains(need, common.Address{}) &&
		headTime != nil && headTime.IsUint64() && a.number.IsUint64() && a.number.Uint64() == h.block.Uint64()+1 &&
		a.time == headTime.Uint64()+slotSeconds
	if ahead {
		accounts := make(map[common.Address]gethclient.OverrideAccount, len(keep))
		for _, account := range keep {
			// the quote contract whole as the answer holds it (covers checked the listing's own slots), every
			// other account only where nothing of it split
			if acct, ok := a.accounts[account]; ok && (account == slots.account || a.whole(account)) {
				accounts[account] = acct
			}
		}
		o = override{time: a.time, number: a.number.Uint64(),
			state: &titan.State{Overrides: accounts, BlockNumber: a.number, BlockTimestamp: a.time}}
	}

	// the same head under the same override: what the refresh priced then stands
	h.key = overrideKey(o)
	if held := ce.getPriced(h.key); held != nil && !skipHeld {
		h.held = held
		return h, o, false, nil
	}
	calls := headCalls(front, door, se.Listing, tokens, common.HexToAddress(se.Custody), maxIn)
	head := len(calls)
	calls = append(calls, blockCalls(multicall, o)...)
	slotsAt := len(calls)
	again, againHolder := rule.read(front, b)
	calls = append(calls, again)
	var raw hexutil.Bytes
	elem, err := multicallElem(multicall, calls, h.block, o, &raw)
	if err != nil {
		return h, o, false, err
	}
	batch := []rpc.BatchElem{elem}
	if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), batch,
		eth.DefaultBatchRetry); err != nil {
		return h, o, false, err
	}
	if batch[0].Error != nil {
		return h, o, false, batch[0].Error
	}
	rets, err := unpackTry(raw, len(calls))
	if err != nil {
		return h, o, false, err
	}
	h.result = make([]bool, head)
	for k, ret := range rets[:head] {
		h.result[k] = ret != nil
	}
	// a call that answered something else is left zero, as the head read's own unpacking leaves it
	if h.result[0] {
		_ = frontABI.UnpackIntoInterface(&h.view, "doorView", rets[0])
	}
	for i := range h.balances {
		h.balances[i] = unpackUint(&abi.Erc20ABI, "balanceOf", rets[1+i])
	}
	if h.result[3] {
		_ = frontABI.UnpackIntoInterface(&h.clock, "clock", rets[3])
	}
	for dir := range h.tops {
		if h.result[4+dir] {
			_ = doorABI.UnpackIntoInterface(&h.tops[dir], "quoteFillable", rets[4+dir])
		}
	}
	h.rungs = unpackRungRead(rets[6])
	if !ahead || !readsBlock(rets[head:slotsAt], o, o.number) {
		return h, o, false, nil
	}
	pinned, ok := slotsOf(rule, front, b, again, againHolder, rets[slotsAt])
	// an update stamped with the block before the target's stands there only where its word carries
	mapped = h.view.View.Desk == desk && ok && pinned.same(slots) && (!carried || carries(h.view.View))
	h.carried = mapped && carried
	return h, o, mapped, nil
}

// readBuilderFacts reads a listing's head-fixed facts: its doorView at the head (a multicall that names the head's
// block), then, pinned to that block, the head's second, the listing's slots and its price contracts. cacheable is
// whether the head the doorView named is the one probed (hd, where probed): its facts are then the cache's at hd's
// hash.
func (t *PoolTracker) readBuilderFacts(ctx context.Context, f *Front, se StaticExtra, front common.Address, hd headID,
	probed bool) (bf *builderFacts, cacheable bool, err error) {
	bf = &builderFacts{}
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: se.Front, Method: "doorView",
			Params: []any{big.NewInt(int64(se.Listing))}}, []any{&bf.at}).
		TryBlockAndAggregate()
	if err != nil {
		return nil, false, err
	}
	if resp.BlockNumber == nil {
		return nil, false, ErrNoBlockNumber
	}
	if resp.Request == nil || resp.Request.RawCallMsg.To == nil || len(resp.Result) != 1 {
		return nil, false, ErrInvalidState
	}
	bf.block = resp.BlockNumber
	bf.multicall = *resp.Request.RawCallMsg.To
	bf.atOK = resp.Result[0]
	cacheable = probed && hd.number != nil && hd.number.Cmp(bf.block) == 0
	desk := uint32(0)
	if bf.atOK {
		desk = bf.at.View.Desk
	}
	multicall := bf.multicall
	rule, err := t.slotRule(ctx, f)
	if err != nil {
		return nil, false, err
	}
	b := binding{listing: se.Listing, desk: desk, wordIndex: bf.at.View.WordIndex}
	named, holder := rule.read(front, b)
	block := bf.block

	// the head's second, the listing's slots and its price contracts, at the head
	var factsRaw, ids, live, deskWord, headQuote, headFirm hexutil.Bytes
	factsElem, err := multicallElem(multicall, []call{
		{abi: &abi.Multicall3ABI, target: multicall, method: "getCurrentBlockTimestamp"}, named}, block, override{},
		&factsRaw)
	if err != nil {
		return nil, false, err
	}
	facts := []rpc.BatchElem{factsElem,
		{Method: "eth_getStorageAt", Args: []any{front, sourceIdsSlot(se.Listing), hexutil.EncodeBig(block)},
			Result: &ids},
		{Method: "eth_getStorageAt", Args: []any{front, liveSourceSlot(se.Listing), hexutil.EncodeBig(block)},
			Result: &live}}
	// the desk word, from the chain's head and no answer
	deskAccount, deskSlot, guarded := rule.deskWord(front, desk)
	if guarded {
		facts = append(facts, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{deskAccount, deskSlot, hexutil.EncodeBig(block)}, Result: &deskWord},
			// and the listing's own quote word, which an update stamped with the head's block must equal
			rpc.BatchElem{Method: "eth_getStorageAt",
				Args:   []any{deskAccount, pairSlot(quoteWordBase, b.wordIndex), hexutil.EncodeBig(block)},
				Result: &headQuote},
			// and its firm word, which a carried update's own must equal: nothing lands the pair in the target block
			rpc.BatchElem{Method: "eth_getStorageAt",
				Args:   []any{deskAccount, pairSlot(firmWordBase, b.wordIndex), hexutil.EncodeBig(block)},
				Result: &headFirm})
	}
	if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), facts,
		eth.DefaultBatchRetry); err != nil {
		return nil, false, err
	}
	if facts[0].Error == nil {
		if r, err := unpackTry(factsRaw, 2); err == nil {
			bf.headTime = unpackUint(&abi.Multicall3ABI, "getCurrentBlockTimestamp", r[0])
			bf.slots, bf.slotted = slotsOf(rule, front, b, named, holder, r[1])
		}
	}
	slots := &bf.slots
	if bf.slotted && guarded && facts[3].Error == nil && len(deskWord) == common.HashLength {
		slots.deskWord, slots.deskKnown = common.BytesToHash(deskWord), true
	}
	if bf.slotted && guarded && facts[4].Error == nil && len(headQuote) == common.HashLength {
		slots.headQuote, slots.headKnown = common.BytesToHash(headQuote), true
	}
	if bf.slotted && guarded && facts[5].Error == nil && len(headFirm) == common.HashLength {
		slots.headFirm, slots.headFirmKnown = common.BytesToHash(headFirm), true
	}
	if facts[1].Error == nil && len(ids) == common.HashLength {
		bf.ids = new(big.Int).SetBytes(ids)
	}
	if facts[2].Error == nil && len(live) == common.HashLength {
		bf.live = new(big.Int).SetBytes(live)
	}
	return bf, cacheable, nil
}

// slotRule is front f's slot rule: the per-desk rule under the front's CELL_CODEHASH, read from the front once and held
// to the config's.
func (t *PoolTracker) slotRule(ctx context.Context, f *Front) (deskRule, error) {
	return slotRuleOf(ctx, t.hashes, f)
}

// carries is whether a fill in the block after a word's landing still reads it, as the builder build's front judges
// a desk's word fresh: the listing's carry life (carryLife) and the word's own carry (bits 4-7) both set.
func carries(v DoorView) bool {
	return carryLife(v.Scopes) && bits(v.Word, 4, 4) != 0
}

// carryLife is the listing's half of carries: its carry life (scopes bit 3) set, on a fence kind that is not NONE.
// The front's carry life is two bits (scopes bits 3-4), of which a life of one block sets bit 3 alone, and bit 4 is a
// tied listing's mark; reading bit 3 alone treats any longer life as none, which shortens the quote's validity and
// never lengthens it.
func carryLife(scopes uint8) bool {
	return (scopes>>3)&1 == 1 && (scopes>>1)&3 != fenceNone
}

// builderWindow is the window of a listing read through /state, whose state is good for the answer's target block
// alone (the pinned sample s, which ran there): each direction is valid to that block's second less the inclusion
// allowance, the last second a route finding quotes it may be built and still reach the block's builder. Where the
// listing and the word both carry (carries), a fill may still read the word in the next block under the same slots:
// that block is sampled too, a slot later, each size keeping its lower output, and the state is valid to that block's
// second less the allowance. Where that sample is not read, or refuses every size of a direction (a word whose life
// ends before that block), the direction keeps the target block's outputs, valid to that block's (cap, keep). An
// update already carried into the target block (carried: stamped with the block before it) is read in no later block,
// so its window is the target block alone. No other moment needs a sample: a fill lands only in those blocks, and the
// front's clock (the block's second, SEAL) reads the same throughout one.
func builderWindow(v DoorView, s pinned, o override, allowance uint64, carried bool) (w window) {
	end := uint64(0)
	if s.time > allowance {
		end = s.time - allowance
	}
	for dir := range 2 {
		w.end[dir] = end
		if carries(v) && !carried && end != 0 {
			next := o
			next.time, next.number = s.time+slotSeconds, s.number+1
			w.at[dir] = []override{next}
			w.cap[dir] = end
			w.keep[dir] = true
			w.end[dir] = end + slotSeconds
		}
	}
	w.guarded = s.guarded
	return w
}

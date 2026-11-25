package efficientprop

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	neturl "net/url"
	"regexp"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	// titan is each front's /state urls and their clients, by its address, on a front read through /state
	// (Front.Titan)
	titan map[string][]stateClient
	// now is the wall clock the schedule of a front read through /state runs on, and its failed urls' log
	now func() time.Time
	// warn logs a /state url's failure (stateFailed), warned when each url's was last logged
	warn   func(url string, err error)
	warnMu sync.Mutex
	warned map[string]time.Time
	// hashes are the code hashes the fronts and their door factories answer, read once
	hashes *codeHashes
	// desks is the maker gate: the desk contracts whose listings are quoted
	desks *approvedDesks
	// shared is each front's last /state answers, by its address, kept for stateEvery (stateShareWindow)
	shared     map[string]*sharedAnswers
	stateEvery time.Duration
}

// sharedAnswers is one front's /state answers as last fetched. Its lock is held through a fetch, so the refreshes of
// the front's listings that come together wait for one fetch and read its answers, which nothing writes after.
type sharedAnswers struct {
	mu      sync.Mutex
	at      time.Time
	answers []answer
	held    bool
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return newPoolTracker(cfg, ethrpcClient, nil)
}

// newPoolTracker is NewPoolTracker with warn as the log of a failed /state url, the logger's Warn where nil.
func newPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client, warn func(url string, err error)) (*PoolTracker, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	t := &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient, titan: map[string][]stateClient{}, now: time.Now,
		warned: map[string]time.Time{}, warn: warn,
		hashes: newCodeHashes(ethrpcClient), desks: newApprovedDesks(cfg),
		shared: map[string]*sharedAnswers{}, stateEvery: stateShareWindow}
	if t.warn == nil {
		t.warn = func(url string, err error) {
			logger.WithFields(logger.Fields{"dexId": cfg.dexID(), "url": url, "error": err.Error()}).
				Warn("efficient-prop: /state url failed")
		}
	}
	for _, f := range cfg.Fronts {
		if !f.builder() {
			continue
		}
		t.shared[f.Address] = &sharedAnswers{}
		for _, url := range f.Titan.URLs {
			// one url at a time, so each client keeps its url; a url that does not dial is logged as a failed one
			// (its key left out) and left out of the reads
			c, err := rpc.DialContext(context.Background(), url)
			if err != nil {
				t.stateFailed(shownURL(url), err)
				continue
			}
			t.titan[f.Address] = append(t.titan[f.Address], stateClient{url: shownURL(url), client: c})
		}
	}
	return t, nil
}

// stateAnswers is front f's /state answers: fetched from its urls (fetchAnswers), or the last fetch's where that is
// under stateEvery old, so the listings of one front refreshed together share one fetch a url. A fetch its caller cut
// short is not kept.
func (t *PoolTracker) stateAnswers(ctx context.Context, f *Front) []answer {
	s := t.shared[f.Address]
	if s == nil || t.stateEvery <= 0 {
		return fetchAnswers(ctx, t.titan[f.Address], f.Titan.Timeout, t.stateFailed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if age := t.now().Sub(s.at); s.held && age >= 0 && age < t.stateEvery {
		return s.answers
	}
	answers := fetchAnswers(ctx, t.titan[f.Address], f.Titan.Timeout, t.stateFailed)
	if ctx.Err() == nil {
		s.answers, s.at, s.held = answers, t.now(), true
	}
	return answers
}

// stateWarnEvery is how often a failing /state url is logged at most.
const stateWarnEvery = time.Minute

// stateFailed logs url's failure (err, its text as shownError gives it), once every stateWarnEvery at most per url, by
// the tracker's wall clock.
func (t *PoolTracker) stateFailed(url string, err error) {
	now := t.now()
	t.warnMu.Lock()
	last, seen := t.warned[url]
	due := !seen || now.Sub(last) >= stateWarnEvery
	if due {
		if t.warned == nil {
			t.warned = map[string]time.Time{}
		}
		t.warned[url] = now
	}
	t.warnMu.Unlock()
	if due && t.warn != nil {
		t.warn(url, shownError(err))
	}
}

// urlInText is a url as an error's text names it: a scheme, then everything up to a space or a quote.
var urlInText = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s"'<>]+`)

// shownError is err as a log gives it: its text with every url in it shown as shownURL shows it. A failed HTTP call's
// error names its request's url, query and username included (net/http hides only the password), and a url that does
// not parse is named whole.
func shownError(err error) error {
	return errors.New(urlInText.ReplaceAllStringFunc(err.Error(), shownURL))
}

// shownURL is url as a log names it: its userinfo and query, where a key may ride, left out.
func shownURL(url string) string {
	u, err := neturl.Parse(url)
	if err != nil {
		return "<a /state url that does not parse>"
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}

// maxPointInput is the largest size sampled: the front takes no input above 2^128.
var maxPointInput = new(big.Int).Lsh(big.NewInt(1), maxPointBits)

// GetNewPoolState samples the door's own quote at a ladder of sizes in both directions, every read pinned to one
// block. The front prices a listing from its operator's posted words and levels and from its fill memory, which this
// source does not replicate: the door's quote is what its fill pays at that state, for a listing that reads no price
// source or only approved ones (below).
//
// The quotes are made by eth_call with no gas price, so at a tip of 0: the door takes the tip from the call
// (tx.gasprice - block.basefee), and the front prices no level by it. A price source a listing reads runs in the quote
// and again in the fill, and can tell them apart (README, How to approve a source), so a listing that reads one
// (PRINT or SESSION) is quoted only where every source it is bound to is one its front's config approves
// (sourcesApproved), none by default. Nor is any listing quoted while the front's proxy runs an implementation its
// config does not list, read at the pinned block (probe). A size the door refuses (it reverts: under the listing's
// minimum, past its depth or custody, the listing dark) is no point. A direction's ladder is the run of passing sizes
// from the smallest, up to the first refusal after it.
//
// A listing with a credit mode is quoted only on a front read through /state whose config names it with its gas
// (Front.CreditListings): past its custody its fill goes to its pool's own swap (its credit route), at the pool's own
// preview and at a route's gas. There every size's class is asked of the front's plan beside its quote, in every block
// sampled (classify): the tip-taking quoteFrom with PLAN_ONLY answers a size it routes tagged ROUTED, and one it fills
// inside custody untagged, equal to the door's quote. A size of no class, or whose class differs between the blocks
// sampled, is no point; a side's ladder is custody points, then routed points (collectClassedRun), kept in Extra.Credit
// alone. Anywhere else a listing with a credit mode is not quoted. Nor is one whose taker tip cap is under the
// configured floor (Config.MinTipCapWei), nor a SESSION listing with a closed phase (closedSession), nor one bound to a
// price source not approved, nor a tied listing (sourcesApproved), nor any listing of a front whose block-count clock
// may read earlier at a later block (frontClock.pin). A listing bound to a guarded source is quoted with its hold's
// blocks sampled too, and not where the front's clock reads BaseTime (window).
//
// Each side's top, the door's quoteFillable of the most the front takes, is sampled too, so the depth above the grid's
// last size is quoted. Every size is sampled again at the later block times of the state's window (window,
// sampleWindow), and the least output kept, so the ladder holds for every fill quoted from it on a listing whose
// levels drift or widen with age. A listing that reads a price print is quoted only as far as that print lives on the
// sampled state (printLife): no later print is assumed.
//
// The state is valid until the end of the window sampled: the state's age bound plus the inclusion allowance after its
// block, never after the last second in which a fill certainly finds the sampled word standing and the sampled sides'
// depth unexpired (validUntil, from doorView and the front's clock() read at the same block). Its timestamp is the
// block's own time, read in the pinned probe, so its age runs from the state it holds.
//
// A front on a builder chain (Ethereum) is read through /state (Front.Titan): there a desk's update lands only in a
// block a builder built with it ahead of a fill, so at the head the door refuses every size. Each refresh fetches the
// front's urls' /state answers and reads the listing from one of them (readBuilderHead): of the highest block's, the
// first in the url order that writes the listing's own stamp slot in its desk's quote contract (its front's slot rule)
// with a stamp the front takes in the target block, none of the listing's slots split, and, on a listing that reads a
// print, holds the price contract it reads now. Every call the state is made of, the head read included, is pinned to
// the head's block and runs in the answer's target block (its number and second, as block overrides) on that answer's
// overrides of the quote contract and the listing's bound price contracts alone; the front's storage reads stay plain
// reads at the head, since a desk's update writes only those contracts. A listing is quoted only where the answer's
// block is the head's next, its slot is the head's slot plus one (its second the head's plus 12 s), and the front's
// clock is the block's second (SEAL), which a time override places; a listing bound to a guarded entry is not quoted
// there. The window is that block, and the next where the listing and the word both carry and the update is the
// target block's own (builderWindow). Every other rule above holds there.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidState
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	tokens := [2]common.Address{common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)}
	custody := common.HexToAddress(staticExtra.Custody)

	// each side's top: the prefix of the most the front takes that the listing fills now
	maxIn := new(big.Int).Sub(maxPointInput, big.NewInt(1))
	// a front read through /state (a builder chain's): its head read runs in the answer's target block, on its
	// state (o0), and the listing is quoted only where its desk's update is in that block (mapped)
	var h headRead
	var o0 override
	mapped := true
	f := t.cfg.front(staticExtra.Front)
	builder := f != nil && f.builder()
	var err error
	if builder {
		h, o0, mapped, err = t.readBuilderHead(ctx, f, p, staticExtra, tokens, maxIn)
	} else {
		h, err = t.readHead(ctx, p, staticExtra, tokens, custody, maxIn)
	}
	if err != nil {
		return p, err
	}
	view, balances, clock, tops := h.view, h.balances, h.clock, h.tops
	if len(h.result) != 6 || !h.result[0] || !h.result[3] || !clock.valid() {
		return p, ErrInvalidState
	}
	block := h.block
	v := view.View
	// a block-count clock reads the chain's millisecond time, the front's configured BaseTime; a front read through
	// /state reads none, and is quoted only on the block's second (below)
	if f := t.cfg.front(staticExtra.Front); f != nil && !builder {
		if clock.ClockMode.Uint64() == clockBlocks && f.BaseTime == "" {
			return p, fmt.Errorf("%w: front %s counts blocks and names no baseTime", ErrInvalidConfig, f.Address)
		}
		clock.baseTime = common.HexToAddress(f.BaseTime)
	}
	if h.multicall == nil {
		return p, ErrInvalidState
	}
	multicall := *h.multicall

	reserves := [2]*big.Int{orZero(balances[0], h.result[1]), orZero(balances[1], h.result[2])}
	// a pool that holds none of a token: a fill with that token in is charged the write of its balance from zero
	var noneHeld uint8
	for i := range 2 {
		if h.result[1+i] && balances[i] != nil && balances[i].Sign() == 0 {
			noneHeld |= 1 << i
		}
	}
	extra := Extra{TipCapWei: tipCapWei(v.TipCap), CreditMode: v.CreditMode, Allowance: t.cfg.inclusionAllowance(),
		NoneHeld:   noneHeld,
		BaseToken1: v.PairWord == nil || v.PairWord.Bit(8) == 0,
		Print:      printed(v.Scopes) || v.PairWord != nil && v.PairWord.Bit(pairAssetBit) == 1,
		Past:       v.BaseIndex != 0 || v.QuoteIndex != 1}
	// a listing with a credit mode only on a front read through /state that names it, with its gas
	var credit *CreditListing
	if f != nil {
		credit = f.creditListing(staticExtra.Listing)
	}
	creditOK := v.CreditMode == 0 || builder && credit != nil && v.CreditMode <= creditModeAlways
	// a pool keeps the fill mode it was listed under (StaticExtra.Mode): the adapter fills it through that entry, and
	// every gas figure its front's config names is the configured mode's. Under a front set to the other mode since,
	// it is not quoted until it is listed again
	modeOK := false
	if f != nil {
		m, ok := f.mode()
		modeOK = ok && m == staticExtra.Mode
	}
	quoted := v.Pool == custody && v.Token0 == tokens[0] && v.Token1 == tokens[1] && creditOK && modeOK &&
		t.cfg.tipCapOK(extra.TipCapWei) && (v.Scopes>>1)&15 != tiedScopes
	// the maker gate: only a listing whose desk's contract is approved
	if quoted {
		if quoted, err = t.deskApproved(ctx, f, v.Desk); err != nil {
			return p, err
		}
	}
	// a listing's own gas, where its front's config names it; never a credit listing's, whose figures are its credit
	// entry's
	if f != nil && v.CreditMode == 0 {
		if g := f.listingGas(staticExtra.Listing); g != nil {
			extra.Gas = &[2]int64{g.BaseIn, g.QuoteIn}
		}
	}
	// a listing of an asset-module pair (a tokenized stock's): its fill also runs its module's checks, whose cost no
	// class figure holds, so it is quoted only with gas of its own (a credit listing's is its credit entry's)
	if v.PairWord != nil && v.PairWord.Bit(pairAssetBit) == 1 && v.CreditMode == 0 && extra.Gas == nil {
		quoted = false
	}
	// idle: not quoted for a reason no new block or /state answer lifts (its desk not approved, its fill mode, its tip
	// cap, a credit mode its front does not name, a class with no gas figure), or not live
	extra.Idle = !quoted || v.Flags&flagLive == 0
	if builder {
		// its allowance is taken off validUntil (builderWindow); its front's clock must be the block's second, which
		// a time override places
		extra.Builder, extra.Allowance = true, 0
		extra.NoAsset, extra.AssetQuote = v.BaseIndex != 0 && v.QuoteIndex != 0, v.QuoteIndex == 0
		quoted = quoted && mapped && clock.ClockMode.Uint64() == clockSeconds
		if !mapped {
			o0 = override{}
		}
		// a class with no measured figure in the front's fill mode is quoted only with figures of its own
		if v.CreditMode == 0 && extra.Gas == nil {
			if _, _, ok := gasOfBuilder(staticExtra.Mode, extra.Print, false, extra.Past, extra.NoAsset,
				extra.AssetQuote); !ok {
				quoted, extra.Idle = false, true
			}
		}
	}
	door := common.HexToAddress(p.Address)
	front := common.HexToAddress(staticExtra.Front)
	var lanes []common.Address
	if quoted && printed(v.Scopes) {
		if quoted, lanes, err = t.sourcesApproved(ctx, front, staticExtra.Listing, block, h.sourceIds); err != nil {
			return p, err
		}
		// a builder build's sources are price contracts, never guarded: a listing bound to a guarded entry there is
		// not quoted
		if builder && len(lanes) > 0 {
			quoted, lanes = false, nil
		}
	}
	// a quoted listing with a credit mode: each size's class is read beside its quote
	var plan *planRead
	if quoted && v.CreditMode != 0 {
		plan = &planRead{front: front, listing: staticExtra.Listing}
	}
	var points [2][]*big.Int
	if quoted {
		points = [2][]*big.Int{t.probeSizes(p, 0, gridReserve(p, 1, reserves[1], plan != nil)),
			t.probeSizes(p, 1, gridReserve(p, 0, reserves[0], plan != nil))}
		for dir := range points {
			if top := tops[dir].FillableAmountIn; h.result[4+dir] && top != nil && top.Sign() > 0 &&
				top.Cmp(maxPointInput) < 0 {
				points[dir] = withSize(points[dir], top)
			}
		}
	}
	session := quoted && (v.Scopes>>1)&3 == fenceSession
	s, err := t.probe(ctx, multicall, door, front, staticExtra.Listing, tokens, points, block, quoted, session, clock,
		lanes, o0, plan)
	if err != nil {
		return p, err
	}
	// a node that ran the pinned read outside the target block gave no sample of it
	if s.unplaced {
		quoted = false
	}
	// the gas class follows the source the listing reads now; the hold is sampled where either bound entry is guarded
	extra.Guarded = quoted && s.liveGuarded
	if session && closedSession(s.session) {
		quoted = false
	}
	// a front upgraded to an implementation not listed may lay out its storage or age its prices otherwise
	if quoted {
		if f := t.cfg.front(staticExtra.Front); f == nil || !f.implements(s.implementation) {
			quoted = false
		}
	}
	// a block-count clock that may read earlier at a later block: no few samples bound the window
	if clock.pin(s.time, s.number, s.baseTimeMs) {
		quoted = false
	}
	if quoted {
		var w window
		if builder {
			w = builderWindow(v, s, o0, t.cfg.builderAllowance(), h.carried)
		} else {
			ends := t.ends(v, s, clock)
			var life printLife
			if printed(v.Scopes) {
				if life, err = t.printLife(ctx, multicall, door, tokens, points, block, s, clock, ends); err != nil {
					return p, err
				}
			}
			w = t.window(v, s, clock, ends, life)
		}
		outputs, caps, err := t.sampleWindow(ctx, multicall, door, tokens, points, block, s, w, plan)
		if err != nil {
			return p, err
		}
		var from [2]int
		for dir := range points {
			ok := make([]bool, len(outputs[dir]))
			for i, out := range outputs[dir] {
				ok[i] = out != nil
			}
			extra.Ladders[dir], from[dir] = collectClassedRun(points[dir], outputs[dir], ok, s.routed[dir])
		}
		if plan != nil {
			// a side whose route has no measured figure is quoted inside its custody alone
			extra.Credit = creditOf(credit, extra.BaseToken1, from, extra.Ladders)
			for dir := range extra.Credit.Ladders {
				if extra.Credit.Gas[dir][1] == 0 && from[dir] < len(extra.Credit.Ladders[dir]) {
					extra.Credit.Ladders[dir] = extra.Credit.Ladders[dir][:from[dir]]
				}
			}
			// a credit listing's ladders ride in Credit alone: a simulator that predates it quotes none
			extra.Ladders = [2][]Point{}
		}
		// valid to the end of the window sampled, never later: route finding checks no other bound once the
		// simulator is built
		for dir, l := range extra.ladders() {
			if len(l) == 0 {
				continue
			}
			until := w.end[dir]
			if caps[dir] != 0 {
				until = min(until, caps[dir])
			}
			if extra.ValidUntil == 0 || until < extra.ValidUntil {
				extra.ValidUntil = until
			}
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	// a direction with no ladder pays nothing: its output side is no liquidity
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	for dir, l := range extra.ladders() {
		if len(l) == 0 {
			p.Reserves[1-dir] = "0"
		}
	}
	p.BlockNumber = block.Uint64()
	// the state's age runs from its block, not from the end of the reads
	p.Timestamp = int64(s.time)
	return p, nil
}

// NextUpdateAt asks for the next refresh refreshEvery after the state's block time: a desk re-posts its word every few
// seconds, and the front's fill memory moves with every fill, which the logs the parser maps (EventParser) bring
// sooner where the service follows them. So no state is asked to stand past its ValidUntil but one whose word ends in
// its own block's second, which route finding never quotes (it stops the inclusion allowance before ValidUntil); asking
// for that state's refresh at once would repeat the same block's read.
//
// A state read through /state (Extra.Builder) holds a target block whose second is ahead of the wall clock, and a new
// answer, a higher block, comes with each block: its refresh is asked for refreshEvery from now, or at its
// ValidUntil where that is sooner (now where that has passed), so a new answer is quoted within a second of its
// publication and no state stands past its block.
//
// An idle state (Extra.Idle), one not quoted for a reason no new block lifts, is asked for idleRefreshEvery on
// instead, on either kind of front. A listing with no ladder for its block alone (no answer for the head's next block
// yet, a word that ended) is not idle, and keeps the schedule above: its next answer is a block away.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	var e Extra
	err := json.Unmarshal([]byte(p.Extra), &e)
	every := refreshEvery
	if err == nil && e.Idle {
		every = idleRefreshEvery
	}
	if err == nil && e.Builder {
		now := t.now().Unix()
		next := now + int64(every/time.Second)
		if e.ValidUntil != 0 && int64(e.ValidUntil) < next {
			next = max(now, int64(e.ValidUntil))
		}
		return next
	}
	return p.Timestamp + int64(every/time.Second)
}

// deskApproved is whether desk, a listing's desk on front f, is an approved one (approvedDesks): its contract is the
// CREATE2 child of the front under the front's CELL_CODEHASH, which the front's deskCellOf answers for it. A listing
// bound to no desk (desk 0) fills nothing, and is never approved.
func (t *PoolTracker) deskApproved(ctx context.Context, f *Front, desk uint32) (bool, error) {
	if f == nil || desk == 0 {
		return false, nil
	}
	cell, err := t.hashes.cell(ctx, *f)
	if err != nil {
		return false, err
	}
	return t.desks.approved(ctx, deskContractOf(common.HexToAddress(f.Address), cell, desk)), nil
}

// readHead is a listing's head read at the head, in one multicall: its block, the front's clock and the listing as
// it stands there.
func (t *PoolTracker) readHead(ctx context.Context, p entity.Pool, se StaticExtra, tokens [2]common.Address,
	custody common.Address, maxIn *big.Int) (h headRead, err error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: se.Front, Method: "doorView",
			Params: []any{big.NewInt(int64(se.Listing))}}, []any{&h.view})
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: token.Hex(), Method: "balanceOf",
			Params: []any{custody}}, []any{&h.balances[i]})
	}
	req.AddCall(&ethrpc.Call{ABI: frontABI, Target: se.Front, Method: "clock"}, []any{&h.clock})
	for dir := range h.tops {
		req.AddCall(&ethrpc.Call{ABI: doorABI, Target: p.Address, Method: "quoteFillable",
			Params: []any{tokens[dir], tokens[1-dir], maxIn}}, []any{&h.tops[dir]})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return h, err
	}
	if resp.BlockNumber == nil {
		return h, ErrNoBlockNumber
	}
	h.result, h.block = resp.Result, resp.BlockNumber
	if resp.Request != nil && resp.Request.RawCallMsg.To != nil {
		multicall := *resp.Request.RawCallMsg.To
		h.multicall = &multicall
	}
	return h, nil
}

var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)

// probeSizes is one direction's sizes: the shared ladder grid (a sweep by the input token's decimals at first,
// then a grid dense at the last ladder's knee), plus twice and four times the last ladder's top, so a listing whose
// depth grew is followed up. The input side's balance is no cap: the input is paid in.
func (t *PoolTracker) probeSizes(p entity.Pool, dir int, outputReserve *big.Int) []*big.Int {
	var prev Extra
	if p.Extra != "" {
		_ = json.Unmarshal([]byte(p.Extra), &prev)
	}
	var guide ladder.Extra
	for d, l := range prev.ladders() {
		for _, point := range l {
			if point[0] == nil || point[1] == nil {
				continue
			}
			guide.Ladders[d] = append(guide.Ladders[d], ladder.Point{point[0].Float64(), point[1].Float64()})
		}
	}
	guideBytes, _ := json.Marshal(guide)
	q := p
	q.Extra = string(guideBytes)
	sizes := ladder.SamplePoints(q, dir, nil, outputReserve)
	if l := prev.ladders()[dir]; len(l) > 0 && l[len(l)-1][0] != nil {
		top := l[len(l)-1][0].ToBig()
		sizes = append(sizes, new(big.Int).Lsh(top, 1), new(big.Int).Lsh(top, 2))
	}

	out := sizes[:0:0]
	for _, s := range sizes {
		if s != nil && s.Sign() > 0 && s.Cmp(maxPointInput) < 0 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Cmp(out[b]) < 0 })
	deduped := out[:0]
	for _, s := range out {
		if len(deduped) == 0 || s.Cmp(deduped[len(deduped)-1]) != 0 {
			deduped = append(deduped, s)
		}
	}
	return deduped
}

// pinned is the refresh's pinned read: the block's time and number, the moment the front's clock reads in it (ms,
// moment) and BaseTime's reading there, the door's quote at every size of both directions at it (nil where the door
// refused), the listing's controls word and the front proxy's implementation (where quoted; the zero address where
// the implementation could not be read), its session word (where asked; nil where it could not be read) and, where it
// is bound to a guarded source, the longest hold of its lanes, and whether the source it reads now is guarded.
type pinned struct {
	time       uint64
	number     uint64
	ms         uint64
	baseTimeMs *big.Int // BaseTime's answer, on a block-count clock
	outputs    [2][]*big.Int
	// routed is, where the sizes' classes were read (planRead), whether the front's plan routes each size (classify);
	// nil where they were not
	routed   [2][]bool
	controls *big.Int
	session  *big.Int
	// implementation is the front proxy's ERC-1967 implementation
	implementation common.Address
	// guarded is whether the listing is bound to a guarded source, and holdMs the longest HOLD_MS of those sources'
	// lanes, defaultHoldMs for one that does not answer
	guarded bool
	holdMs  uint64
	// liveGuarded is whether the source the listing reads now is guarded: the guard (bit 247) of its record word 2,
	// the live source's copy a fill reads alone, where that word was read and is set; else whether either bound entry
	// is (guarded)
	liveGuarded bool
	// unplaced: the read was to run in a block override's block (a /state answer's target block), and its block's
	// time or number was another
	unplaced bool
}

// probe reads the block's time, and on a block-count clock BaseTime's milliseconds, each guarded source's lane's hold
// (lanes, HOLD_MS), and asks the door's quote at every size of both directions, in one multicall pinned to block, and
// where quoted reads the listing's controls word and the front proxy's implementation at that block beside it, where
// session its session word, and where bound to a guarded source its live source word (record word 2), in one batch.
// Where plan is set (a listing with a credit mode), the front's plan of every size is asked beside its quote in the
// same multicall, and each size classed by it (classify).
func (t *PoolTracker) probe(ctx context.Context, multicall, door, front common.Address, listing uint32,
	tokens [2]common.Address, points [2][]*big.Int, block *big.Int, quoted, session bool,
	c frontClock, lanes []common.Address, o override, plan *planRead) (s pinned, err error) {
	calls := []call{{abi: &abi.Multicall3ABI, target: multicall, method: "getCurrentBlockTimestamp"},
		{abi: &baseTimeABI, target: c.baseTime, method: "timestampMs"}}
	if c.ClockMode.Uint64() != clockBlocks || c.baseTime == (common.Address{}) {
		calls = calls[:1]
	}
	timed := len(calls)
	// in a block override's block (o), its number too, so a sample run elsewhere is caught
	if o.time != 0 {
		calls = append(calls, call{abi: &abi.Multicall3ABI, target: multicall, method: "getBlockNumber"})
	}
	holds := len(calls)
	for _, lane := range lanes {
		calls = append(calls, call{abi: &laneABI, target: lane, method: "HOLD_MS"})
	}
	head := len(calls)
	calls = append(calls, quoteCalls(door, tokens, points)...)
	planned := len(calls)
	calls = append(calls, plan.calls(door, tokens, points)...)
	var raw hexutil.Bytes
	var ctl, ses, impl, live hexutil.Bytes
	elem, err := multicallElem(multicall, calls, block, o, &raw)
	if err != nil {
		return s, err
	}
	batch := []rpc.BatchElem{elem}
	if quoted {
		batch = append(batch, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{front, controlsSlot(listing), hexutil.EncodeBig(block)}, Result: &ctl})
	}
	reads := len(batch)
	if quoted {
		batch = append(batch, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{front, implementationSlot, hexutil.EncodeBig(block)}, Result: &impl})
	}
	liveAt := len(batch)
	if len(lanes) > 0 {
		batch = append(batch, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{front, liveSourceSlot(listing), hexutil.EncodeBig(block)}, Result: &live})
	}
	if session {
		batch = append(batch, rpc.BatchElem{Method: "eth_getStorageAt",
			Args: []any{front, sessionSlot(listing), hexutil.EncodeBig(block)}, Result: &ses})
	}
	if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), batch,
		eth.DefaultBatchRetry); err != nil {
		return s, err
	}
	for _, b := range batch[:reads] {
		if b.Error != nil {
			return s, b.Error
		}
	}
	// an implementation not read is the zero address, which no front lists
	if quoted && batch[reads].Error == nil && len(impl) == common.HashLength {
		s.implementation = common.BytesToAddress(impl)
	}
	// a session word not read is nil: the listing is not quoted (closedSession)
	if session && batch[len(batch)-1].Error == nil && len(ses) == common.HashLength {
		s.session = new(big.Int).SetBytes(ses)
	}
	rets, err := unpackTry(raw, len(calls))
	if err != nil {
		return s, err
	}
	ts := unpackUint(&abi.Multicall3ABI, "getCurrentBlockTimestamp", rets[0])
	if ts == nil || !ts.IsUint64() || ts.Sign() == 0 {
		return s, ErrInvalidState
	}
	s.time, s.number = ts.Uint64(), block.Uint64()
	if timed > 1 {
		s.baseTimeMs = unpackUint(&baseTimeABI, "timestampMs", rets[1])
	}
	if o.time != 0 {
		n := unpackUint(&abi.Multicall3ABI, "getBlockNumber", rets[timed])
		s.unplaced = s.time != o.time || n == nil || !n.IsUint64() || n.Uint64() != o.number
		s.number = o.number
	}
	// a lane whose hold is not read holds for the most any lane does
	s.guarded = len(lanes) > 0
	s.liveGuarded = s.guarded
	if s.guarded && batch[liveAt].Error == nil && len(live) == common.HashLength {
		if w := new(big.Int).SetBytes(live); w.Sign() != 0 {
			s.liveGuarded = w.Bit(entryGuardBit) == 1
		}
	}
	for _, ret := range rets[holds:head] {
		hold := uint64(defaultHoldMs)
		if v := unpackUint(&laneABI, "HOLD_MS", ret); v != nil && v.IsUint64() {
			hold = v.Uint64()
		}
		s.holdMs = max(s.holdMs, hold)
	}
	s.ms = c.moment(s.time, s.number, s.baseTimeMs)
	s.outputs = readQuotes(rets[head:planned], points)
	if plan != nil {
		s.outputs, s.routed = classify(s.outputs, readPlans(rets[planned:], points))
	}
	if quoted {
		if len(ctl) != common.HashLength {
			return s, ErrInvalidState
		}
		s.controls = new(big.Int).SetBytes(ctl)
	}
	return s, nil
}

// window is the blocks besides the pinned one, each a block override, in which each direction's ladder must hold, the
// last second a fill quoted from it lands (end), and the last second it may be quoted to where those samples cannot
// be read (0: no bound beyond end).
type window struct {
	at  [2][]override
	end [2]uint64
	cap [2]uint64
	// blind marks a direction whose samples no override places (the front's clock is one place does not follow): it
	// is quoted no later than its cap
	blind [2]bool
	// guarded: the listing is bound to a guarded source, and a direction any of whose samples is not read or placed
	// has no ladder
	guarded bool
	// keep marks a direction whose samples, where all refuse every size, leave the pinned block's outputs standing,
	// quoted no later than its cap (a carried block on a front read through /state, builderWindow)
	keep [2]bool
}

// ends is each direction's last second a fill quoted from the state lands in: the state's age bound plus the
// inclusion allowance after its block, never after the side's word and depth end.
func (t *PoolTracker) ends(v DoorView, s pinned, c frontClock) (ends [2]uint64) {
	for dir := range 2 {
		var only [2]bool
		only[dir] = true
		ends[dir] = min(validUntil(v, c.MaxTtlMs.Uint64(), c.AgeSlackMs.Uint64(), only),
			s.time+uint64(maxAge/time.Second)+t.cfg.inclusionAllowance())
	}
	return ends
}

// sourcesApproved is whether every price source a listing is bound to is one its front's config approves (see the
// README, How to approve a source), the binding read from the front's storage at block: the listing's source ids
// (record word 10, the primary and the backup where it has one, 32 bits each), each resolved to its entry (an entry is
// written once, never edited): the contract it names (bits 0-159) and its guard (bit 247). lanes are the contracts of
// the guarded entries among them, whose holds the window samples (window). Any source answers the front's call as it
// likes, so the door's quote, sampled by an eth_call, is what a fill pays only on a source trusted to answer every call
// alike. With no source approved nothing is read. A listing bound to none, one whose ids or entries the node does not
// answer, or one bound to a source not approved, is not quoted; a transport failure fails the refresh. ids is the
// listing's source ids word where the refresh has read it at block already (nil: it is read here).
func (t *PoolTracker) sourcesApproved(ctx context.Context, front common.Address, listing uint32,
	block *big.Int, ids *big.Int) (approved bool, lanes []common.Address, err error) {
	f := t.cfg.front(hexutil.Encode(front[:]))
	if f == nil || len(f.ApprovedSources) == 0 {
		return false, nil, nil
	}
	read := func(slots []common.Hash) ([]*big.Int, error) {
		words := make([]hexutil.Bytes, len(slots))
		batch := make([]rpc.BatchElem, len(slots))
		for k, slot := range slots {
			batch[k] = rpc.BatchElem{Method: "eth_getStorageAt", Args: []any{front, slot, hexutil.EncodeBig(block)},
				Result: &words[k]}
		}
		if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), batch,
			eth.DefaultBatchRetry); err != nil {
			return nil, err
		}
		out := make([]*big.Int, len(slots))
		for k, b := range batch {
			if b.Error == nil && len(words[k]) == common.HashLength {
				out[k] = new(big.Int).SetBytes(words[k])
			}
		}
		return out, nil
	}
	if ids == nil {
		words, err := read([]common.Hash{sourceIdsSlot(listing)})
		if err != nil || words[0] == nil {
			return false, nil, err
		}
		ids = words[0]
	}
	primary, backup := uint32(bits(ids, 0, 32)), uint32(bits(ids, 32, 32))
	if primary == 0 {
		return false, nil, nil
	}
	slots := []common.Hash{sourceSlot(primary)}
	if backup != 0 {
		slots = append(slots, sourceSlot(backup))
	}
	entries, err := read(slots)
	if err != nil {
		return false, nil, err
	}
	for _, e := range entries {
		if e == nil || !f.approves(common.BigToAddress(e)) {
			return false, nil, nil
		}
		if lane := common.BigToAddress(e); e.Bit(entryGuardBit) == 1 && !slices.Contains(lanes, lane) {
			lanes = append(lanes, lane)
		}
	}
	return true, lanes, nil
}

// pairAssetBit is the pair word's bit of an asset-module pair, whose every fill runs its module's checks.
const pairAssetBit = 15

// fenceSession is the fence kind SESSION (scopes bits 1-2).
const fenceSession = 3

// closedSession is whether a SESSION listing is left out, its session word (record word 13, read at the state's
// block) session: it has a closed phase (closedAfter, bits 0-15, nonzero), or its word was not read (nil). In
// the closed phase the front clamps each level to a band around the stale print that eases as the print ages, while
// the word's own levels tighten, so a quote can fall below its window's every sample; and the refused gap between
// the phases may be shorter than a second. A SESSION listing with no closed phase is quoted as a PRINT listing is.
func closedSession(session *big.Int) bool {
	return session == nil || bits(session, 0, 16) != 0
}

// printed is whether a listing of binding scopes reads a price print: its fence kind (bits 1-2) PRINT (0) or SESSION
// (3), a tied listing's included (which is not quoted).
func printed(scopes uint8) bool {
	k := (scopes >> 1) & 3
	return k == 0 || k == 3
}

// printLife bounds the window of a listing that reads a price print. read: it was asked.
type printLife struct {
	read bool
	// until is each direction's last second the door still fills in, on the pinned state
	until [2]uint64
	// blind marks a direction whose bound could not be read: it is quoted no later than the block's own second
	blind [2]bool
}

// printLife asks, for a listing that reads a price print, in which seconds up to each direction's end the door still
// fills the least size it filled at the block, each in the second's last block (place) on the pinned block's state, in
// one batch: on that state no later print lands, so the print ages, and past its source's window or the listing's
// max age the front refuses every size. A direction's bound is the last second of the run from the block's that
// fills, before its first refusal. Each sample reads its block's time and number beside its quotes, as the window's do
// (sampleWindow). A SESSION listing with a closed phase, which fills again once its print is stale for its
// closed-phase delay, is not quoted (closedSession), so no run here reaches a closed phase.
func (t *PoolTracker) printLife(ctx context.Context, multicall, door common.Address, tokens [2]common.Address,
	points [2][]*big.Int, block *big.Int, s pinned, c frontClock, ends [2]uint64) (life printLife, err error) {
	life.read = true
	var least [2]*big.Int
	for dir := range 2 {
		life.until[dir] = s.time
		for i, out := range s.outputs[dir] {
			if out != nil {
				least[dir] = points[dir][i]
				break
			}
		}
	}
	type probe struct {
		at    uint64
		o     override
		sized [2][]*big.Int
	}
	var probes []probe
	for at := s.time + 1; at <= max(ends[0], ends[1]); at++ {
		o, ok := c.place(at*1000 + 999)
		if !ok {
			// no override places the second's last block: the print's life cannot be read
			life.blind = [2]bool{true, true}
			return life, nil
		}
		pr := probe{at: at, o: o}
		for dir := range 2 {
			if least[dir] != nil && at <= ends[dir] {
				pr.sized[dir] = []*big.Int{least[dir]}
			}
		}
		probes = append(probes, pr)
	}
	if len(probes) == 0 {
		return life, nil
	}
	raws := make([]hexutil.Bytes, len(probes))
	batch := make([]rpc.BatchElem, len(probes))
	for k, pr := range probes {
		calls := append(blockCalls(multicall, pr.o), quoteCalls(door, tokens, pr.sized)...)
		if batch[k], err = multicallElem(multicall, calls, block, pr.o, &raws[k]); err != nil {
			return life, err
		}
	}
	if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), batch,
		eth.DefaultBatchRetry); err != nil {
		return life, err
	}
	var done [2]bool
	for k, pr := range probes {
		var rets [][]byte
		if batch[k].Error == nil {
			head := len(blockCalls(multicall, pr.o))
			if rets, err = unpackTry(raws[k], head+len(pr.sized[0])+len(pr.sized[1])); err != nil {
				return life, err
			}
		}
		number := pr.o.number
		if number == 0 {
			number = block.Uint64()
		}
		read := batch[k].Error == nil && readsBlock(rets, pr.o, number)
		var outs [2][]*big.Int
		if read {
			outs = readQuotes(rets[len(blockCalls(multicall, pr.o)):], pr.sized)
		}
		for dir := range 2 {
			if done[dir] || len(pr.sized[dir]) == 0 {
				continue
			}
			switch {
			case !read:
				life.blind[dir], done[dir] = true, true
			case outs[dir][0] == nil:
				done[dir] = true
			default:
				life.until[dir] = pr.at
			}
		}
	}
	return life, nil
}

// window plans the samples a direction's ladder needs to hold for every fill quoted from it. A fill quoted from the
// state lands by the state's age bound plus the inclusion allowance after the block, and never after the side's word
// and depth end, nor, on a listing that reads a price print, after that print's life (life): that second, end. Over
// [block time, end] the front's discount on every listing quoted (drift x age and widening x sqrt(age) past the
// controls' grace, the side's skew, and a premium while the clock is under the window after the latest of the word's
// maker time, the side's depth nonce and the controls' nonce) only rises with age but for the premium's end and, on a
// guarded source, its hold's end, so the least a size pays in the window is at the block's time, in end's last block,
// in the premium window's last block, or in a block of the hold:
// those are sampled, each in the block in which the front's clock reads that latest moment (place; where it reads
// BaseTime, at the moment itself, end's at its second's last millisecond), so a sample is never younger than a fill in
// its second, nor past the premium's end. Where the premium ends inside the window, the first block at or after its end
// (firstFrom) is sampled too: the side pays the most there, the premium lifted and the age barely moved, so a side
// bound by its output (a desk's bucket, the pool's room) refuses there a size it fills before and after, and that size
// is no point. A premium that ends by the state's own moment (pinned.ms), earlier in its second, is over in every block
// a fill can land in, and is not sampled. A guarded source prices a side at the worst print its lane still holds,
// and once the lane's hold (HOLD_MS, read from the lane at the state's block; defaultHoldMs where it cannot be read)
// has passed since the post that replaced a print, at the kept print alone, a better price: so every block from the
// state's to the first at or after the hold's end is sampled, the side's top refused there where its output bound
// binds. A listing whose discount may fall as its print ages in other ways (tied; a SESSION listing with a closed
// phase) is not quoted.
//
// Where the node refuses a block override or ignores one (its block, or BaseTime's state), or the front's clock is one
// no override places, a direction whose drift or widening is nonzero is quoted no later than its grace's last second,
// within which the age moves no level, and one whose premium ends inside the window no later than the last second
// wholly before that end (cap); so is one whose premium edge no override places. A direction whose print life could
// not be read is quoted no later than the block's time. On a guarded listing a direction any of whose samples is not
// placed or read is not quoted (sampleWindow), nor is any direction where the front's clock reads BaseTime: a block
// there may store any sub-second, so the hold's blocks are not a few steps from the state's.
func (t *PoolTracker) window(v DoorView, s pinned, c frontClock, ends [2]uint64, life printLife) window {
	var w window
	ageSlackMs := c.AgeSlackMs.Uint64()
	ctl := s.controls
	makerMs := bits(v.Word, 72, 48)
	baseIsToken0 := v.PairWord != nil && v.PairWord.Bit(8) == 1
	for dir := range 2 {
		end := ends[dir]
		if life.read {
			end = min(end, life.until[dir])
		}
		w.end[dir] = end
		if life.blind[dir] {
			w.cap[dir], w.blind[dir] = s.time, true
			continue
		}
		if end <= s.time {
			continue
		}
		d0 := v.Depth[2]
		if (dir == 0) == baseIsToken0 {
			d0 = v.Depth[0]
		}
		if bits(ctl, 136, 20) != 0 || bits(d0, 92, 20) != 0 {
			w.cap[dir] = s.time
			if lead := max(999, ageSlackMs); makerMs+bits(ctl, 192, 16) >= lead {
				w.cap[dir] = max(s.time, (makerMs+bits(ctl, 192, 16)-lead)/1000)
			}
		}

		last, ok := c.place(end*1000 + 999)
		if !ok {
			w.blind[dir] = true
		}
		if premium := bits(ctl, 156, 20); premium != 0 {
			start := max(makerMs, bits(d0, 12, 48), bits(ctl, 16, 48))
			// a premium that ends after the state's moment and by end's last millisecond ends inside the window
			if pend := start + bits(ctl, 176, 16); pend > s.ms && pend <= end*1000+999 {
				// the last second no block of which reads the premium's end
				before := s.time
				if pend >= 1000 {
					before = max(s.time, (pend-1000)/1000)
				}
				if w.cap[dir] == 0 || before < w.cap[dir] {
					w.cap[dir] = before
				}
				edges := 0
				if o, ok := c.place(pend - 1); ok && o.time >= s.time {
					edges++
					if o != last {
						w.at[dir] = append(w.at[dir], o)
					}
				}
				if o, ok := c.firstFrom(pend); ok && o.time <= end {
					edges++
					if o != last {
						w.at[dir] = append(w.at[dir], o)
					}
				} else if ok {
					edges++ // no block of the window reads it: no fill lands past the premium
				}
				if edges < 2 {
					w.blind[dir] = true
				}
			}
		}
		// a guarded source prices the side at the worst print its lane holds, and once the hold ends at the kept print
		// alone, a better price: every block from the state's to the first at or after the hold's end. Where the front
		// reads BaseTime a block may store any sub-second, so a moment just before the hold ends is no step from the
		// state's: the direction is not quoted (blind, on a guarded listing)
		if s.guarded && c.ClockMode.Uint64() == clockBlocks && c.reads == readsBaseTime {
			w.blind[dir] = true
		} else if s.guarded {
			step := uint64(1000)
			if c.ClockMode.Uint64() == clockBlocks {
				step = blockMs
			}
			for k := uint64(1); (k-1)*step < s.holdMs; k++ {
				o, ok := c.firstFrom(s.ms + k*step)
				if !ok {
					w.blind[dir] = true
					break
				}
				if o.time > end {
					break
				}
				w.at[dir] = append(w.at[dir], o)
			}
		}
		if !w.blind[dir] {
			w.at[dir] = append(w.at[dir], last)
		}
	}
	w.guarded = s.guarded
	return w
}

// sampleWindow asks the door's quote at every size again in each of the window's blocks, on the pinned block's
// state, in one batch, and keeps per size the least output of every sample (nil where any refused). Each sample reads
// the block's time and number first, beside its quotes: a sample in which they are not the override's, from a node
// that took the override and ignored it, is no sample. caps[dir] is the direction's quoting bound where a sample it
// needed was not read, or where a direction the window keeps (w.keep) was refused every size there (0: none); on a
// guarded listing such an unread direction has no ladder. Where plan is set (a listing with a credit mode), each sample
// asks the front's plan of every size beside its quote, and a size whose class there is none, or not the pinned
// block's (s.routed), is no point.
func (t *PoolTracker) sampleWindow(ctx context.Context, multicall, door common.Address, tokens [2]common.Address,
	points [2][]*big.Int, block *big.Int, s pinned, w window, plan *planRead) (outputs [2][]*big.Int, caps [2]uint64,
	err error) {
	outputs = s.outputs
	var drop [2]bool
	for dir := range 2 {
		if w.blind[dir] {
			caps[dir] = w.cap[dir]
			drop[dir] = w.guarded
		}
	}
	// a guarded listing's direction with a sample unread or unplaced is not quoted
	defer func() {
		for dir := range 2 {
			if drop[dir] {
				outputs[dir] = make([]*big.Int, len(points[dir]))
			}
		}
	}()
	var blocks []override
	for _, os := range w.at {
		for _, o := range os {
			if !slices.Contains(blocks, o) {
				blocks = append(blocks, o)
			}
		}
	}
	if len(blocks) == 0 {
		return outputs, caps, nil
	}
	raws := make([]hexutil.Bytes, len(blocks))
	masks := make([][2]bool, len(blocks))
	batch := make([]rpc.BatchElem, len(blocks))
	for k, o := range blocks {
		var sized [2][]*big.Int
		for dir := range 2 {
			if slices.Contains(w.at[dir], o) {
				masks[k][dir], sized[dir] = true, points[dir]
			}
		}
		calls := append(blockCalls(multicall, o), quoteCalls(door, tokens, sized)...)
		calls = append(calls, plan.calls(door, tokens, sized)...)
		if batch[k], err = multicallElem(multicall, calls, block, o, &raws[k]); err != nil {
			return outputs, caps, err
		}
	}
	if err := eth.BatchCallWithRetry(ctx, t.ethrpcClient.GetETHClient().Client(), batch,
		eth.DefaultBatchRetry); err != nil {
		return outputs, caps, err
	}
	perSize := 1
	if plan != nil {
		perSize = 2
	}
	for k := range blocks {
		var sized [2][]*big.Int
		for dir := range 2 {
			if masks[k][dir] {
				sized[dir] = points[dir]
			}
		}
		var rets [][]byte
		if batch[k].Error == nil {
			head := len(blockCalls(multicall, blocks[k]))
			if rets, err = unpackTry(raws[k], head+perSize*(len(sized[0])+len(sized[1]))); err != nil {
				return outputs, caps, err
			}
		}
		number := blocks[k].number
		if number == 0 {
			number = block.Uint64()
		}
		if batch[k].Error != nil || !readsBlock(rets, blocks[k], number) {
			// the node refused the override, or ignored it: no sample, and the direction quoted no later than its cap
			for dir := range 2 {
				if masks[k][dir] {
					caps[dir] = w.cap[dir]
					drop[dir] = drop[dir] || w.guarded
				}
			}
			continue
		}
		head := len(blockCalls(multicall, blocks[k]))
		quotes := head + len(sized[0]) + len(sized[1])
		later := readQuotes(rets[head:quotes], sized)
		// the sizes' classes there: a kept direction's refusal is the door's, before any size is classed
		classed, routed := later, [2][]bool{}
		if plan != nil {
			classed, routed = classify(later, readPlans(rets[quotes:], sized))
		}
		for dir := range 2 {
			// a kept direction refused every size there: its pinned outputs stand, to its cap
			if masks[k][dir] && w.keep[dir] && !slices.ContainsFunc(later[dir], func(o *big.Int) bool { return o != nil }) {
				caps[dir] = w.cap[dir]
				continue
			}
			for i, out := range classed[dir] {
				if plan != nil && out != nil && routed[dir][i] != s.routed[dir][i] {
					out = nil // a size the front routes in one block and fills inside custody in the other
				}
				if cur := outputs[dir][i]; cur != nil && (out == nil || out.Cmp(cur) < 0) {
					outputs[dir][i] = out
				}
			}
		}
	}
	return outputs, caps, nil
}

// blockCalls read the block's time and number, the block a sample in o ran in, and where o sets BaseTime, BaseTime.
func blockCalls(multicall common.Address, o override) []call {
	calls := []call{{abi: &abi.Multicall3ABI, target: multicall, method: "getCurrentBlockTimestamp"},
		{abi: &abi.Multicall3ABI, target: multicall, method: "getBlockNumber"}}
	if o.baseTimeMs != 0 {
		calls = append(calls, call{abi: &baseTimeABI, target: o.baseTime, method: "timestampMs"})
	}
	return calls
}

// readsBlock is whether blockCalls' answers, the first of rets, are o's block time, number (the pinned block's where
// o names none) and BaseTime's reading: a node that takes an override and ignores it, the block's or the state's,
// ran the sample elsewhere.
func readsBlock(rets [][]byte, o override, number uint64) bool {
	if len(rets) < len(blockCalls(common.Address{}, o)) {
		return false
	}
	ts := unpackUint(&abi.Multicall3ABI, "getCurrentBlockTimestamp", rets[0])
	n := unpackUint(&abi.Multicall3ABI, "getBlockNumber", rets[1])
	if ts == nil || n == nil || !ts.IsUint64() || !n.IsUint64() || ts.Uint64() != o.time || n.Uint64() != number {
		return false
	}
	if o.baseTimeMs == 0 {
		return true
	}
	ms := unpackUint(&baseTimeABI, "timestampMs", rets[2])
	return ms != nil && ms.IsUint64() && ms.Uint64() == o.baseTimeMs
}

// quoteCalls asks the door's quote at every size of both directions.
func quoteCalls(door common.Address, tokens [2]common.Address, points [2][]*big.Int) []call {
	var calls []call
	for dir := range points {
		for _, size := range points[dir] {
			calls = append(calls, call{abi: &doorABI, target: door, method: "quote",
				params: []any{tokens[dir], tokens[1-dir], size}})
		}
	}
	return calls
}

// readQuotes is quoteCalls' answers by direction and size: nil where the door refused.
func readQuotes(rets [][]byte, points [2][]*big.Int) (outputs [2][]*big.Int) {
	k := 0
	for dir := range points {
		outputs[dir] = make([]*big.Int, len(points[dir]))
		for i := range points[dir] {
			outputs[dir][i] = unpackUint(&doorABI, "quote", rets[k])
			k++
		}
	}
	return outputs
}

// withSize is sizes (ascending, distinct) with size in its place.
func withSize(sizes []*big.Int, size *big.Int) []*big.Int {
	i := sort.Search(len(sizes), func(i int) bool { return sizes[i].Cmp(size) >= 0 })
	if i < len(sizes) && sizes[i].Cmp(size) == 0 {
		return sizes
	}
	return slices.Insert(sizes, i, size)
}

// collectRun is a direction's ladder from its sizes (ascending) and the door's answers: the passing sizes from the
// smallest, up to the first refusal or falling output after it, at most maxLadderPoints. A size under the listing's
// minimum refuses below the run, a size past its depth above it; a refusal inside the run would leave a span the
// door may not fill, so the run ends there. On sizes the front's plan classes, collectClassedRun.
func collectRun(sizes, outputs []*big.Int, ok []bool) []Point {
	run, _ := collectClassedRun(sizes, outputs, ok, nil)
	return run
}

func orZero(v *big.Int, ok bool) *big.Int {
	if !ok || v == nil {
		return new(big.Int)
	}
	return v
}

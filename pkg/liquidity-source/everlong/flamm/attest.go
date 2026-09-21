package everlongflamm

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
)

// A refresh is quotable only when (1) the wiring it read is the listing's, as the registries resolve it, (2) the
// state built from its reads reproduces the deployed views read beside them, (3) the deployed views a deadline word
// decides answer at the clocks inside the snapshot window what the port answers there (deadlineClocks below), and
// (4) the pool's own previewSwap / previewLever and the Router's fundingCeiling, called at the same block on amounts
// the port picks -- a grid in each direction and the band edge the port locates by bisection -- answer exactly what
// the port answers at that block's timestamp, reverts included. Any difference, or anything the port cannot map,
// fails the refresh closed. The bisection binds one local edge per direction, not every one: the acceptance set is
// not an interval (edge), so a state can hold several, and each probe added to the aggregate is gas against the
// node's own eth_call cap (probeAggregateGas). Measured over every integer amount at 51470150, the deployed pool's
// buy direction holds 22 class transitions and the grid straddles one of them; the refusal class below the grid's
// first buy rung of 1,000 -- FillInvalid, which every buy up to 779 answers with -- is not probed at all, and
// neither is either leverage direction's NothingToFill, since each leverage grid carries one amount (on an armed
// pool at 51470153, lever-down reaches it at 1 and lever-up at 24,691,965, and that direction holds 41
// transitions). What a refresh binds per state is therefore the priced interior and one edge per direction; the
// refusal classes themselves are bound against the chain offline, by the core grids, which carry the chain's own
// answer from amount 0 upwards in both swap and both leverage directions at three blocks
// (testdata/core_e2e_grid_*.jsonl.gz, TestCoreE2EPreviewGrids).
//
// The probes bind the state's priced consequences at this snapshot, not every word: a word that moves no probe (a
// one-wei change of a book or Morpho total below every probed answer's resolution, a flag only another state reads)
// is bound by the tracker's decoding instead, which TestTrackerReplay pins field for field to the core replay's
// dump, and, for the MMRouter configuration, the FLAMMStore words and the factory's upgrade schedule, by the
// storage layout check (drift). A word whose only effect is at a later clock is bound by (3).

// probeAggregateGas is the gas the probe aggregate's eth_call is sent with. Its cost is state-dependent and far
// larger than the view aggregate's: the pool as deployed costs 2.87M for 17 probes, an armed pool 5.2M, a moved
// book 7.9M, a notional-capped state 9.8M, and a state the curator can reach with ordinary calls (setDials walking
// ltv down its on-chain envelope with the Router pin, a reserve target and an ordinary buy) 11.3M; a Router-only
// pin measured 13.0M. Sending the limit explicitly makes the refresh independent of whatever a node applies to a
// call that names none. A node whose own eth_call cap is lower caps it there instead, and what follows depends on
// where that cap falls: below the aggregate's own frame it fails the eth_call outright, a transport error that
// leaves pool-service on its last snapshot; between that and the aggregate's full cost it lets the aggregate run
// and starves the tail subcalls, which Multicall3 reports as unsuccessful with empty returndata and confirmRevert
// then re-calls on their own, so the pool publishes as refusing rather than attesting a fill the chain never
// refused. Either way the cap is an operator requirement (README, "Known limitations").
const probeAggregateGas uint64 = 30_000_000

// revertError maps revert data onto the port's error vocabulary: custom errors by selector, Solidity panics, and
// OpenZeppelin 4.8 Math.mulDiv's empty revert. nil means unmapped.
func revertError(data []byte) error {
	if len(data) == 0 {
		return errMulDivOverflow
	}
	if len(data) < 4 {
		return nil
	}
	var sel [4]byte
	copy(sel[:], data[:4])
	if sel == [4]byte{0x4e, 0x48, 0x7b, 0x71} && len(data) == 36 { // Panic(uint256)
		switch data[35] {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		case 0x32:
			return errPanicIndex
		}
		return nil
	}
	return revertSelectors[sel]
}

// drift reports the first difference between the wiring the refresh read and the pool's pinned identity, w (the
// listing as the registries resolve it), plus the venue set the refresh is publishing (empty: none). Each hook's
// binding is compared with its registry entry by its kind (hookKindSpec.bound). The venue set is the refresh's own,
// so "venue count" and "venue i" report a set that changed under the round, not one that changed since the
// listing: the tracker re-reads it and runs the round again before publishing drift (pool_tracker.go).
func (s *snapshot) drift(pool common.Address, se *StaticExtra, venueSet []StaticVenue, w *poolWiring,
	overrides map[common.Address]gethclient.OverrideAccount) string {
	if w == nil || w.pool != pool {
		return "listing"
	}
	venues := len(venueSet)
	checks := []struct {
		ok   bool
		what string
	}{
		{s.Asset == se.PoolAsset && s.LoanAsset == se.LoanAsset && s.Reads.Pool.Loans[0].Token == se.LoanAsset, "pair"},
		{s.LoanCount.Eq(uOne) && s.RouterLoanCount.Eq(uOne) && s.RouterLoanToken == se.LoanAsset, "loan set"},
		{s.Router == se.Router && s.PriceFeed == se.PriceFeed && s.Factory == se.Factory, "pool bindings"},
		{s.Implementation == se.Implementation, "implementation"},
		{s.Hooks == se.Hooks && s.Hooks == w.hooks.Addrs, "hook set"},
		{w.hooks.bound(roleSwap, &s.Bindings[roleSwap], pool), roleName[roleSwap] + " binding"},
		{w.hooks.bound(roleLeverage, &s.Bindings[roleLeverage], pool), roleName[roleLeverage] + " binding"},
		{w.hooks.bound(roleSpread, &s.Bindings[roleSpread], pool), roleName[roleSpread] + " binding"},
		{s.SequencerFeed == se.SequencerFeed && s.FeedAggregators == se.Aggregators &&
			s.Reads.Feed.Tokens[0].Known && s.Reads.Feed.Tokens[1].Known &&
			s.Reads.Feed.Tokens[0].Heartbeat.Uint64() == se.Heartbeats[0] &&
			s.Reads.Feed.Tokens[1].Heartbeat.Uint64() == se.Heartbeats[1], "price feed"},
		{s.VenueCount.IsUint64() && s.VenueCount.Uint64() == uint64(venues), "venue count"},
		{validVenues(w, venueSet) == nil, "venue profile"},
		{s.LayoutOk, "storage layout"},
	}
	for _, c := range checks {
		if !c.ok {
			return c.what
		}
	}
	for i := range venueSet {
		if s.VenueAccounts[i] != venueSet[i].Account || s.VenueIDs[i] != venueSet[i].MarketID {
			return fmt.Sprintf("venue %d", i)
		}
	}
	// A code override on any registered contract the pool is wired to (every codehash the listing pins, the Morpho
	// singleton) or on the AdaptiveCurveIrm the port prices replaces the code the port mirrors.
	mirrored := []common.Address{pool, se.Factory, se.Implementation, se.Router, se.PriceFeed, se.Morpho,
		se.Hooks[roleSlot[roleSwap]], se.Hooks[roleSlot[roleLeverage]], se.Hooks[roleSlot[roleSpread]], adaptiveCurveIrm}
	for i := range venueSet {
		mirrored = append(mirrored, venueSet[i].Account)
	}
	for _, a := range mirrored {
		if o, ok := overrides[a]; ok && o.Code != nil {
			return "code override " + a.Hex()
		}
	}
	return ""
}

func eqWords(a, b []uint256.Int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Eq(&b[i]) {
			return false
		}
	}
	return true
}

// attestViews recomputes the deployed views the round read beside the state (empty: all reproduced).
func attestViews(s *snapshot, st *flammState) string {
	now := s.Timestamp
	ok, price, ts := st.Feed.peekCross(&st.Feed.Asset, &st.Feed.Loans[0], now)
	if ok != s.PeekCrossOk || !price.Eq(&s.PeekCrossPrice) || ts != s.PeekCrossTs {
		return fmt.Sprintf("peekCross: chain (%v %s %d) port (%v %s %d)", s.PeekCrossOk, s.PeekCrossPrice.Dec(),
			s.PeekCrossTs, ok, price.Dec(), ts)
	}
	peg, err := st.pegOk(0, now)
	if (s.PegOk == nil) != (err != nil) || (s.PegOk != nil && *s.PegOk != peg) {
		return fmt.Sprintf("pegOk: port %v %v", peg, err)
	}
	pos, err := st.Router.positions(now)
	if (s.Positions == nil) != (err != nil) {
		return fmt.Sprintf("positions: port %v", err)
	}
	if s.Positions != nil {
		c := s.Positions
		if !eqWords(c.Coll, pos.Coll) || !eqWords(c.Sup, pos.Sup) || !eqWords(c.Debt, pos.Debt) ||
			!c.TotalColl.Eq(&pos.TotalColl) {
			return fmt.Sprintf("positions: chain %+v port %+v", *c, pos)
		}
		// The ledger identity: gross = physical + recognized collateral.
		var gross uint256.Int
		if _, overflow := gross.AddOverflow(&st.Pool.Physical, &pos.TotalColl); overflow ||
			!gross.Eq(&s.Reads.Pool.Gross) {
			return fmt.Sprintf("gross: chain %s port %s", s.Reads.Pool.Gross.Dec(), gross.Dec())
		}
	}
	_, sup, debt, err := st.Router.position(0, now)
	if err != nil || !st.Pool.Loans[0].Liquid.Eq(&s.LoanPosition[0]) || !sup.Eq(&s.LoanPosition[1]) ||
		!debt.Eq(&s.LoanPosition[2]) {
		return fmt.Sprintf("loanPosition: chain %v port (%s %s) %v", s.LoanPosition, sup.Dec(), debt.Dec(), err)
	}
	for i := range st.Router.Venues {
		vp, err := st.Router.venuePosition(uint16(i), now)
		if err != nil || !eqWords(vp[:], s.VenuePositions[i][:]) {
			return fmt.Sprintf("venuePosition(%d): chain %v port %v %v", i, s.VenuePositions[i], vp, err)
		}
		m := &st.Router.Venues[i].Morpho
		tp, err := m.tryPosition(now)
		want := s.TryPositions[i]
		if err != nil || tp.Readable != want.Readable || !tp.Collateral.Eq(&want.Collateral) ||
			!tp.SupplyShares.Eq(&want.SupplyShares) || !tp.Supplied.Eq(&want.Supplied) || !tp.Debt.Eq(&want.Debt) {
			return fmt.Sprintf("tryPosition(%d): chain %+v port %+v %v", i, want, tp, err)
		}
		rateOk, rate, err := m.tryBorrowRate(uZero, uZero, now)
		if err != nil || rateOk != m.IrmReadable || !rate.Eq(&s.BorrowRates[i]) {
			return fmt.Sprintf("borrowRateAfter(%d): chain %s port (%v %s) %v", i, s.BorrowRates[i].Dec(), rateOk,
				rate.Dec(), err)
		}
	}
	pool := st.Pool
	var nav uint256.Int
	b, err := st.priced(&pool, now)
	if err == nil {
		nav, err = gateNavAt(&b)
	}
	if s.TotalAssets != nil {
		if err != nil || !nav.Eq(s.TotalAssets) {
			return fmt.Sprintf("totalAssets: chain %s port %s %v", s.TotalAssets.Dec(), nav.Dec(), err)
		}
	} else if want := revertError(s.TotalAssetsErr); want == nil || !errors.Is(err, want) {
		return fmt.Sprintf("totalAssets: chain revert %x port %v", s.TotalAssetsErr, err)
	}
	return ""
}

// Deadline words -- a word whose only effect is to fix a future second at which something changes -- move no answer
// at the snapshot's own clock, so neither the views nor the probes bind them: LeverageSpreadHook's lastSetTs and
// maxSpreadAge (its post answers through lastSetTs + maxSpreadAge, LeverageSpreadHook.sol:75), each aggregator
// round's updatedAt (its quote is read through updatedAt + heartbeat, PriceFeed.sol:171) and the sequencer round's
// startedAt (the feed answers nothing until startedAt + SEQUENCER_GRACE, PriceFeed.sol:157-162). A value on the
// permissive side is fail-open: the simulator keeps quoting fills the pool reverts, up to the end of the window
// the policy admits quoting in.
//
// They are bound at the second the port believes each predicate flips. Every one of them is monotone in the clock
// -- a live post and a fresh round only ever go dead, a sequencer's grace only ever ends -- so, given that the
// chain and the port agree at the snapshot (attestViews' peekCross and the probes), agreeing at the two seconds
// either side of that flip pins it exactly: the chain must answer one way at the last second before it and the
// other way at the flip itself. A predicate whose flip is outside the window needs only the window's far end,
// which is why the deployed pool costs one extra round trip. A policy that declares no window -- an explicit
// maxSnapshotAgeSec of 0, which also lifts the staleness refusal -- has no end to bind against and sends none, as
// it publishes no oracle-window answer either.

// deadlineClocks are the clocks the deadline round reads at: the end of the window, which binds every predicate
// whose flip is past it, plus the two seconds either side of each flip that falls inside the window. Clocks are
// deduplicated and never before the snapshot's own, which the view attestation and the probes already bind.
func deadlineClocks(st *flammState, now, window uint64) []uint64 {
	if window == 0 || st == nil {
		return nil
	}
	end := now + window
	if end < now { // a window that overflows the clock binds nothing
		return nil
	}
	out := []uint64{end}
	seen := map[uint64]bool{now: true, end: true}
	add := func(t uint64) {
		if t > now && t <= end && !seen[t] {
			seen[t], out = true, append(out, t)
		}
	}
	for _, f := range flipSeconds(st, now) {
		if f <= now || f > end {
			continue // outside the window: the end, and the snapshot's own round, bind it
		}
		add(f - 1)
		add(f)
	}
	return out
}

// flipSeconds is the first second at which each clock-decided predicate the port evaluates answers differently
// from the second before it. Each is monotone in the clock, so that second is the whole of its clock dependence:
// the feed rounds and the keeper's post go dead at `written + window + 1`, and the sequencer's grace is the mirror
// -- it becomes usable at `startedAt + grace + 1` (PriceFeed.sol:157-162, a "not before" rather than a deadline).
// A predicate the state does not reach at all (a round the snapshot itself refuses, a post with no staleness
// window) has no flip and contributes nothing.
func flipSeconds(st *flammState, now uint64) []uint64 {
	var out []uint64
	add := func(base, span *uint256.Int) {
		var f uint256.Int
		if f.Add(base, span); f.IsUint64() && f.Uint64() != ^uint64(0) {
			out = append(out, f.Uint64()+1)
		}
	}
	f := &st.Feed
	tokens := []*feedToken{&f.Asset}
	if len(f.Loans) > 0 {
		tokens = append(tokens, &f.Loans[0])
	}
	for _, t := range tokens {
		if _, _, err := feedRead(t, now); err != nil {
			continue
		}
		add(&t.Round.UpdatedAt, &t.Heartbeat)
	}
	if sp, err := st.Hooks.Spread.port(); err == nil {
		if lastSet, maxAge, ok := sp.expiry(now); ok {
			add(&lastSet, &maxAge)
		}
	}
	if r := &f.Sequencer; f.HasSequencer && r.Ok && r.Answer.IsZero() && !r.StartedAt.IsZero() {
		add(&r.StartedAt, &f.SequencerGrace)
	}
	return out
}

// attestDeadlines reads the deadline round at each clock and compares it with what the port answers there, stopping
// at the first clock the port does not reproduce (empty: every clock reproduced, or the policy declares no window).
// A round the chain answered at the block with a revert or undecodable data fails the refresh closed like any
// other read of it; only a transport failure is returned.
func attestDeadlines(ctx context.Context, rpc *mcRPC, se *StaticExtra, s *snapshot, st *flammState,
	window uint64) (string, error) {
	clocks := deadlineClocks(st, s.Timestamp, window)
	if len(clocks) == 0 {
		return "", nil
	}
	block := new(big.Int).SetUint64(s.Block)
	for _, at := range clocks {
		answer, err := readClock(ctx, rpc, se, block, at)
		if err != nil {
			if !chainAnswered(err) {
				return "", err
			}
			return fmt.Sprintf("deadline at %d: %v", at, err), nil
		}
		if f := attestClock(st, &answer); f != "" {
			return f, nil
		}
	}
	return "", nil
}

func attestClock(st *flammState, c *clockAnswer) string {
	at := c.At
	f := &st.Feed
	ok, price, ts := f.peekCross(&f.Asset, &f.Loans[0], at)
	if ok != c.CrossOk || !price.Eq(&c.CrossPrice) || ts != c.CrossTs {
		return fmt.Sprintf("peekCross at %d: chain (%v %s %d) port (%v %s %d)", at, c.CrossOk, c.CrossPrice.Dec(),
			c.CrossTs, ok, price.Dec(), ts)
	}
	for i, t := range []*feedToken{&f.Asset, &f.Loans[0]} {
		ok, usd, ts := f.peekUsd(t, at)
		if ok != c.UsdOk[i] || !usd.Eq(&c.Usd[i]) || ts != c.UsdTs[i] {
			return fmt.Sprintf("peekUsd(%d) at %d: chain (%v %s %d) port (%v %s %d)", i, at, c.UsdOk[i],
				c.Usd[i].Dec(), c.UsdTs[i], ok, usd.Dec(), ts)
		}
	}
	peg, err := st.pegOk(0, at)
	if (c.PegOk == nil) != (err != nil) || (c.PegOk != nil && *c.PegOk != peg) {
		return fmt.Sprintf("pegOk at %d: port %v %v", at, peg, err)
	}
	if c.PegOk == nil {
		if want := revertError(c.PegErr); want == nil || !errors.Is(err, want) {
			return fmt.Sprintf("pegOk at %d: chain revert %x port %v", at, c.PegErr, err)
		}
	}
	if c.HasSpread != st.Hooks.hasSpread() {
		return fmt.Sprintf("spreadPpm at %d: chain hook %v port %v", at, c.HasSpread, st.Hooks.hasSpread())
	}
	if c.HasSpread {
		ok, ppm, err := st.Hooks.Spread.spreadPpm(at)
		if err != nil {
			return fmt.Sprintf("spreadPpm at %d: port %v", at, err)
		}
		if ok != c.SpreadOk || !ppm.Eq(&c.SpreadPpm) {
			return fmt.Sprintf("spreadPpm at %d: chain (%v %s) port (%v %s)", at, c.SpreadOk, c.SpreadPpm.Dec(), ok,
				ppm.Dec())
		}
	}
	return ""
}

// probe is one on-chain call the port must reproduce: its answer words, or its error.
type probe struct {
	call  mcCall
	words []uint256.Int
	err   error
}

func decWords(ws []uint256.Int) []string {
	out := make([]string, len(ws))
	for i := range ws {
		out[i] = ws[i].Dec()
	}
	return out
}

func (p *probe) check(res mcResult) string {
	if !res.Ok {
		want := revertError(res.Data)
		if want == nil || !errors.Is(p.err, want) {
			return fmt.Sprintf("%s: chain reverted %s, port %v %v", p.call.Name, hexutil.Encode(res.Data),
				decWords(p.words), p.err)
		}
		return ""
	}
	if p.err != nil {
		return fmt.Sprintf("%s: chain answered %s, port %v", p.call.Name, hexutil.Encode(res.Data), p.err)
	}
	vals, err := unpack(&p.call, res)
	if err != nil || len(vals) != len(p.words) {
		return fmt.Sprintf("%s: undecodable answer %x", p.call.Name, res.Data)
	}
	for i := range vals {
		w, err := wordOf(vals[i])
		if err != nil || !w.Eq(&p.words[i]) {
			return fmt.Sprintf("%s: word %d chain %v port %s", p.call.Name, i, vals[i], p.words[i].Dec())
		}
	}
	return ""
}

// Probe amounts: a log grid per direction and venue; bisection extends each grid to the first refusal.
var (
	probeSellGrid      = []uint64{1, 1_000, 10_000, 100_000, 1_000_000}
	probeBuyGrid       = []uint64{1_000, 1_000_000, 100_000_000, 10_000_000_000}
	probeLeverUpGrid   = []uint64{10_000}
	probeLeverDownGrid = []uint64{10_000_000}
)

// probeSwap / probeLever evaluate the preview the way the probe will see it.
func probeSwap(pool common.Address, st *flammState, sell bool, amount *uint256.Int, now uint64) probe {
	pr := probe{call: mcCall{Name: fmt.Sprintf("previewSwap(%v,%s)", sell, amount.Dec()), Target: pool,
		ABI: &flammABI, Method: "previewSwap", Args: []any{sell, amount.ToBig()}, Optional: true}}
	p, err := st.previewSwap(sell, amount, now)
	if err != nil {
		pr.err = err
	} else {
		pr.words = []uint256.Int{p.UsedNative, p.NetNative, p.FeeWad}
	}
	return pr
}

func probeLever(pool common.Address, st *flammState, up bool, amount *uint256.Int, now uint64) probe {
	pr := probe{call: mcCall{Name: fmt.Sprintf("previewLever(%v,%s)", up, amount.Dec()), Target: pool,
		ABI: &flammABI, Method: "previewLever", Args: []any{up, amount.ToBig()}, Optional: true}}
	r, err := st.previewLever(up, amount, now)
	if err != nil {
		pr.err = err
	} else {
		pr.words = []uint256.Int{r.AmountInUsed, r.AmountOut, r.SpreadPpm, r.CrAfterWad}
	}
	return pr
}

// edge locates, from a grid, an adjacent (accepted, refused) pair above the largest accepted grid amount: the
// acceptance set is not an interval (the band check saw-tooths on the output grid), so the pair is a local edge,
// which both sides must agree on.
func edge(grid []uint64, ok func(*uint256.Int) bool) (lo, hi *uint256.Int, found bool) {
	lo = nil
	for _, g := range grid {
		if a := uint256.NewInt(g); ok(a) {
			lo = a
		}
	}
	if lo == nil {
		return nil, nil, false
	}
	hi = new(uint256.Int).Set(lo)
	for i := 0; i < 32; i++ {
		hi.Mul(hi, uint256.NewInt(4))
		if !ok(hi) {
			break
		}
		lo.Set(hi)
		if i == 31 {
			return nil, nil, false
		}
	}
	var mid uint256.Int
	for new(uint256.Int).Sub(hi, lo).GtUint64(1) {
		mid.Add(lo, hi).Rsh(&mid, 1)
		if ok(&mid) {
			lo.Set(&mid)
		} else {
			hi.Set(&mid)
		}
	}
	return lo, hi, true
}

// probesFor builds the probe set for a state at its snapshot timestamp.
func probesFor(pool common.Address, se *StaticExtra, st *flammState, peekOk bool, peekPrice *uint256.Int,
	now uint64) []probe {
	var out []probe
	for _, dir := range []struct {
		sell bool
		grid []uint64
	}{{true, probeSellGrid}, {false, probeBuyGrid}} {
		for _, g := range dir.grid {
			out = append(out, probeSwap(pool, st, dir.sell, uint256.NewInt(g), now))
		}
		if lo, hi, ok := edge(dir.grid, func(a *uint256.Int) bool {
			_, err := st.previewSwap(dir.sell, a, now)
			return err == nil
		}); ok {
			out = append(out, probeSwap(pool, st, dir.sell, lo, now), probeSwap(pool, st, dir.sell, hi, now))
		}
	}
	if st.Hooks.hasLeverage() {
		for _, dir := range []struct {
			up   bool
			grid []uint64
		}{{true, probeLeverUpGrid}, {false, probeLeverDownGrid}} {
			for _, g := range dir.grid {
				out = append(out, probeLever(pool, st, dir.up, uint256.NewInt(g), now))
			}
			if lo, hi, ok := edge(dir.grid, func(a *uint256.Int) bool {
				_, err := st.previewLever(dir.up, a, now)
				return err == nil
			}); ok {
				out = append(out, probeLever(pool, st, dir.up, lo, now), probeLever(pool, st, dir.up, hi, now))
			}
		}
	}
	if peekOk {
		for _, extra := range []uint64{0, 100_000} {
			var coll uint256.Int
			coll.AddUint64(&st.Pool.Physical, extra)
			pr := probe{call: mcCall{Name: fmt.Sprintf("fundingCeiling(%s)", coll.Dec()), Target: se.Router,
				ABI: &routerABI, Method: "fundingCeiling", Args: []any{pool, uint8(0), coll.ToBig(), peekPrice.ToBig()},
				Optional: true}}
			w, err := st.Router.fundingCeiling(0, &coll, peekPrice, now)
			if err != nil {
				pr.err = err
			} else {
				pr.words = []uint256.Int{w}
			}
			out = append(out, pr)
		}
	}
	return out
}

// attestProbes runs the probes at the snapshot block: (probes run, first mismatch, transport error).
func attestProbes(ctx context.Context, rpc *mcRPC, pool common.Address, se *StaticExtra, s *snapshot,
	st *flammState) (int, string, error) {
	probes := probesFor(pool, se, st, s.PeekCrossOk, &s.PeekCrossPrice, s.Timestamp)
	calls := make([]mcCall, len(probes))
	for i := range probes {
		calls[i] = probes[i].call
	}
	block := new(big.Int).SetUint64(s.Block)
	_, res, err := rpc.aggregate(ctx, block, callOpts{gas: probeAggregateGas}, calls)
	if err != nil {
		return 0, "", err
	}
	for i := range probes {
		// Multicall3 reports a subcall that ran out of gas the way it reports a revert with no data, which is the
		// shape OpenZeppelin's mulDiv reverts with, so a probe the port refuses for that reason is confirmed on its
		// own before it is read as agreement (multicall.go confirmRevert).
		if !res[i].Ok && len(res[i].Data) == 0 {
			if err := rpc.confirmRevert(ctx, block, &probes[i].call, probeAggregateGas); err != nil {
				if !chainAnswered(err) {
					return len(probes), "", err // transport, as for the aggregate itself: the last entity stands
				}
				return len(probes), fmt.Sprintf("%s: empty revert unconfirmed: %v", probes[i].call.Name, err), nil
			}
		}
		if f := probes[i].check(res[i]); f != "" {
			return len(probes), f, nil
		}
	}
	return len(probes), "", nil
}

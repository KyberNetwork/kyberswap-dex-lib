package everlongflamm

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator quotes one FLAMM pool (tokens: pool asset, loan asset 0) over its two venues on one shared book:
// the swap venue in both directions and, when leverage routing is on, the leverage venue (a sell is a lever-up, a
// buy a lever-down). A quote runs the pool's whole settlement on a copy of the state -- the hook fill, the Router
// legs through Morpho, the gate -- at the quote's timestamp, with Morpho interest accrued exactly to it, and keeps
// the better venue: the swap, unless the leverage fill pays more than Policy.LeverMinEdgeBps above it, which is the
// edge its larger gas has to earn. The venue is picked on the settlements alone and only the picked one is put
// through the configured guards, so a margin can refuse a quote but never move it to the other venue.
// UpdateBalance adopts that settlement's post-state. Nothing is quoted on a snapshot older than the policy allows
// or across a scheduled change of the code or hooks the port mirrors (fresh).
type PoolSimulator struct {
	pool.Pool
	StaticExtra StaticExtra
	// Venues is the refresh's Router venue set (Extra.Venues), re-checked against the registry on every quote: it is
	// the one part of the wiring a curator can change on a live pool.
	Venues []StaticVenue
	// OracleAhead is what each venue's Morpho market oracle answers at the far end of the snapshot window
	// (Extra.OracleAhead), in venue order: the one input of the snapshot the clock alone moves.
	OracleAhead []OracleAnswer
	Policy      Policy
	// Attested is the refresh verdict NewPoolSimulator required; a decoded simulator must still carry it.
	Attested bool

	state *flammState
	// snapshotTs is the refresh's block timestamp (state.Timestamp moves with every adopted fill), scheduledAt
	// Extra.ScheduledChangeAt.
	snapshotTs, scheduledAt uint64
	// lineage identifies the adopted state: derived from the entity by NewPoolSimulator, folded with each fill by
	// UpdateBalance and kept by CloneState and msgpack, so a SwapInfo is accepted exactly by the copies of the
	// state it was quoted on -- including a second simulator built from the same entity (lineageOf).
	lineage uint64
	// seq counts the transitions this simulator adopted.
	seq    uint64
	broken bool
	// nowFn is the quote clock (unix seconds); nil is the wall clock. Never earlier than the snapshot.
	nowFn func() uint64 `msgpack:"-"`
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulatorFromParams)

// NewPoolSimulatorFromParams is the registered factory. It honours pool.FactoryOpts.StaleCheck, which route
// finding sets and indexing does not: a snapshot already past Policy.MaxSnapshotAgeSec (or inside the lead time of
// a scheduled change) is refused at construction rather than only at the quote. Every quote re-checks it anyway
// (fresh), so a simulator cached across the age still stops quoting.
func NewPoolSimulatorFromParams(params pool.FactoryParams) (*PoolSimulator, error) {
	sim, err := NewPoolSimulator(params.EntityPool)
	if err != nil {
		return nil, err
	}
	if params.Opts.StaleCheck {
		if err := sim.fresh(sim.now()); err != nil {
			return nil, err
		}
	}
	return sim, nil
}

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var se StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &se); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}
	w, err := validEntity(&p, &se)
	if err != nil {
		return nil, err
	}
	if extra.ProfileDrift != "" {
		return nil, ErrProfileDrift
	}
	if !extra.Attested || extra.AttestFailure != "" || extra.Reads == nil {
		return nil, ErrNotAttested
	}
	st, err := extra.Reads.state(&w.hooks)
	if err != nil {
		return nil, err
	}
	if st.Block != p.BlockNumber {
		return nil, ErrNotAttested
	}
	sim := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(p.Address),
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) string { return strings.ToLower(t.Address) }),
			Reserves:    lo.Map(p.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: p.BlockNumber,
		}},
		StaticExtra: se,
		Venues:      extra.Venues,
		OracleAhead: extra.OracleAhead,
		Policy:      extra.Policy.withDefaults(),
		Attested:    true,
		state:       st,
		snapshotTs:  st.Timestamp,
		scheduledAt: extra.ScheduledChangeAt,
	}
	sim.lineage = lineageOf(sim.Info.Address, p.Extra)
	if err = sim.usable(); err != nil {
		return nil, err
	}
	return sim, nil
}

// now is the quote timestamp: the clock, never before the snapshot.
func (p *PoolSimulator) now() uint64 {
	var t uint64
	if p.nowFn != nil {
		t = p.nowFn()
	} else if u := time.Now().Unix(); u > 0 {
		t = uint64(u)
	}
	return max(t, p.state.Timestamp)
}

// usable re-checks what a decoded simulator could have lost or had altered: the listing against the registries,
// the state's hook kinds against the listing's, the refresh verdict and the envelope the port is quoted in. It runs
// on every quote because msgpack restores a simulator without NewPoolSimulator.
func (p *PoolSimulator) usable() error {
	switch {
	case p.broken:
		return ErrSwapInfoMismatch
	case p.state == nil || !p.Attested:
		return ErrNotAttested
	}
	w, err := validStatic(&p.StaticExtra, p.Info.Address, p.Info.Type, p.Info.Tokens)
	if err != nil {
		return err
	}
	if err := validVenues(w, p.Venues); err != nil {
		return err
	}
	// The end-of-window oracle answers are read from the venue set itself (tracker_reads.go readOracleAhead), so a
	// refresh publishes either none of them -- a policy that declares no window -- or exactly one per venue. Any
	// other length is an Extra no refresh wrote, and it is refused here rather than left to oracleShifted, where a
	// mismatch skips the guard silently and quotes the snapshot answer alone.
	if n := len(p.OracleAhead); n != 0 && n != len(p.Venues) {
		return ErrPoolRefused
	}
	if err := p.state.Hooks.matches(&w.hooks); err != nil {
		return err
	}
	return p.state.envelope(&p.StaticExtra, len(p.Venues), p.Policy.QuoteDonatedVenues)
}

// envelope refuses the whole pool outside the states the port is quoted in: one loan asset (the listed pair), and
// every venue readable -- an IRM that answers (neither inside the account's grace nor quarantined past it), an
// oracle that answers -- with no Morpho collateral or supply shares beyond what the Router manages, unless
// donated is set.
//
// Morpho lets anyone supplyCollateral or supply on behalf of the venue account, and the Router never recognizes
// such a gift (MMRouter.recognizeCollateral only raises the tracker by what the pool's emergency lane posted,
// IMMRouter.sol:143), so the refusal lasts until the curator's emergency lane withdraws the excess (collateral
// only with no debt outstanding, MorphoBlueAccount.sol:272) and anyone can renew it for the gas of a one-wei
// donation. donated (Config.QuoteDonatedVenues) quotes the pool instead. The port prices a donation as the Router
// does: recognized = min(actual, managed) (MMRouterLib.read :560-562), the actual position where the Router reads
// it (_slice :607, freeCollateral :755, venueHealth MMRouter.sol:464) and Blue's own health and share math on the
// actual position; the donation sequences of testdata/edges/core_edge_seq_*.jsonl.gz replay exactly
// (TestSimulatorDonation).
func (s *flammState) envelope(se *StaticExtra, venues int, donated bool) error {
	if s.PoolAsset != se.PoolAsset || len(s.Pool.Loans) != 1 || len(s.Router.Loans) != 1 || len(s.Feed.Loans) != 1 ||
		s.Pool.Loans[0].Token != se.LoanAsset || len(s.Router.Venues) != venues {
		return ErrPoolRefused
	}
	for i := range s.Router.Venues {
		v := &s.Router.Venues[i]
		if m := &v.Morpho; (m.HasIrm && !m.IrmReadable) || !m.OracleOk {
			return ErrPoolRefused
		}
		if !donated && (v.Morpho.Position.Collateral.Gt(&v.ManagedCollateral) ||
			v.Morpho.Position.SupplyShares.Gt(&v.ManagedSupplyShares)) {
			return ErrPoolRefused
		}
	}
	return nil
}

// oracleShifted is a copy of the state with every venue's Morpho market oracle answering what the refresh read it
// will answer at the far end of the snapshot window (Extra.OracleAhead), or nil when no venue's answer moves
// there. That answer is not a state transition: the BTC/USD feed behind the market oracle is a Chainlink SVR
// DualAggregator, which withholds each primary round for a fixed delay, so the same state answers a different
// price once the clock passes the reveal -- invisible to the probes, which all run at the snapshot clock.
//
// Exactly two predicates read the price. MMRouterLib.bandOk accepts it inside a band around the PriceFeed cross
// (MMRouterLib.sol:780-788), an interval; Morpho's _isHealthy caps the borrow at collateral * price * lltv
// (morpho-blue Morpho.sol:527-539), which only rises with the price. A fill that settles identically at the
// snapshot answer and at this one therefore settles identically at every answer between them, which is every
// answer the reveal can produce inside the window. An entity whose refresh published
// no ahead answers -- a policy with maxSnapshotAgeSec 0, which declares no window at all, or an entity recorded
// before the forward round -- is quoted at the snapshot answer alone.
//
// A set of another length than the venue set never reaches here: usable refuses the pool over it, since it is an
// Extra no refresh wrote -- readOracleAhead allocates its answers from the very slice the state's venues are built
// from (pool_tracker.go readWindow, tracker_reads.go readOracleAhead) -- and a set that does not line up with the
// venues names no venue's answer to compare. The length test below is therefore the no-window case alone, and it
// stays because oracleShifted is also read outside a quote (TestSimulatorOracleWindow).
func (p *PoolSimulator) oracleShifted() *flammState {
	venues := p.state.Router.Venues
	if len(p.OracleAhead) != len(venues) {
		return nil
	}
	moved := false
	for i := range p.OracleAhead {
		a, m := &p.OracleAhead[i], &venues[i].Morpho
		if a.Ok != m.OracleOk || !a.Price.Eq(&m.OraclePrice) {
			moved = true
			break
		}
	}
	if !moved {
		return nil
	}
	c := p.state.clone()
	for i := range p.OracleAhead {
		a, m := &p.OracleAhead[i], &c.Router.Venues[i].Morpho
		// An oracle that stops answering is refused either way, so the branch oracleZero separates is not read.
		m.OracleOk, m.OraclePrice, m.OracleZero = a.Ok, a.Price, false
	}
	return c
}

// fill is one venue's settlement, before the policy guards. capEvals and passes are a swap sell's gas counts
// (swapResult); priceWad is the checked feed cross the fill's own gates ran at and payL18 a leverage fill's taker
// loan leg, both of which the guards re-check.
type fill struct {
	venue            uint8
	used, out        uint256.Int
	fee              uint256.Int
	spread           uint256.Int
	priceWad         uint256.Int
	payL18           uint256.Int
	capEvals, passes uint64
	next             *flammState
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(params, -1)
}

// calcAmountOut quotes on the venue quote picks, or on venue when it is not negative.
func (p *PoolSimulator) calcAmountOut(params pool.CalcAmountOutParams, venue int) (*pool.CalcAmountOutResult, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	idxIn, idxOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}
	var amountIn uint256.Int
	if a := params.TokenAmountIn.Amount; a == nil || a.Sign() <= 0 || amountIn.SetFromBig(a) {
		return nil, ErrInvalidAmount
	}
	now := p.now()
	if err := p.fresh(now); err != nil {
		return nil, err
	}
	if err := p.feedMargin(now); err != nil {
		return nil, err
	}
	sell := idxIn == 0
	var f *fill
	var err error
	switch venue {
	case int(VenueSwap):
		f, err = p.quoteSwap(sell, &amountIn, now)
	case int(VenueLever):
		f, err = p.quoteLever(sell, &amountIn, now)
	default:
		f, err = p.quote(sell, &amountIn, now)
	}
	if err != nil {
		return nil, err
	}
	f.next.Timestamp = now // the chain's clock cannot run back past a settled fill
	if f.out.IsZero() || f.used.IsZero() || f.used.Gt(&amountIn) {
		return nil, ErrZeroAmountOut
	}
	var remaining uint256.Int
	remaining.Sub(&amountIn, &f.used)
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: f.out.ToBig()},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: f.fee.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: remaining.ToBig()},
		Gas:                    p.gas(f, sell),
		SwapInfo: SwapInfo{Venue: f.venue, PoolAssetIn: sell, AmountInUsed: f.used, AmountOut: f.out,
			SpreadPpm: f.spread, lineage: p.lineage, amountIn: amountIn, seq: p.seq, next: f.next},
	}, nil
}

// quote picks the venue: the swap, and the leverage venue when routing is on.
//
// The pick runs on the two venues' settlements alone -- what the pool itself would fill -- and only the venue it
// picks is then put through the configured guards. A guard can therefore refuse the routed quote but never move it
// to the other venue: were the pick made on guarded quotes, a margin refusal of one venue would hand the route to
// the other at a different price, which is a margin changing an amount (README, "Margins never change an amount").
func (p *PoolSimulator) quote(sell bool, amountIn *uint256.Int, now uint64) (*fill, error) {
	sw, swErr := p.settleSwap(sell, amountIn, now)
	if !p.Policy.LeverRouting {
		if swErr != nil {
			return nil, swErr
		}
		return sw, p.guardSwap(sw, sell, amountIn, now)
	}
	lv, lvErr := p.settleLever(sell, amountIn, now)
	f, err := pickVenue(sw, swErr, lv, lvErr, p.Policy.LeverMinEdgeBps)
	if err != nil {
		return nil, err
	}
	if f.venue == VenueSwap {
		return f, p.guardSwap(f, sell, amountIn, now)
	}
	return f, p.guardLever(f, sell, amountIn, now)
}

// pickVenue keeps the swap fill unless the leverage fill pays more than minEdgeBps above it, which is the edge its
// larger gas has to earn (constant.go defaultLeverMinEdgeBps): CalcAmountOut returns one result per pool, so a
// router that sees the leverage quote never sees the swap quote it replaced. A swap venue that cannot settle at all
// leaves the leverage fill as the only quote. When both refuse, the swap's refusal is reported.
func pickVenue(sw *fill, swErr error, lv *fill, lvErr error, minEdgeBps uint64) (*fill, error) {
	switch {
	case swErr != nil && lvErr != nil:
		return nil, swErr
	case swErr != nil:
		return lv, nil
	case lvErr != nil || !beatsBy(&lv.out, &sw.out, minEdgeBps):
		return sw, nil
	}
	return lv, nil
}

// beatsBy reports out > base + base * edgeBps / 10000: an edge of 0 is the plain comparison, and an edge whose
// product is not representable keeps the base.
func beatsBy(out, base *uint256.Int, edgeBps uint64) bool {
	if edgeBps == 0 {
		return out.Gt(base)
	}
	var need uint256.Int
	if _, overflow := need.MulOverflow(base, uint256.NewInt(edgeBps)); overflow {
		return false
	}
	need.Div(&need, uBps)
	if _, overflow := need.AddOverflow(&need, base); overflow {
		return false
	}
	return out.Gt(&need)
}

// gas is the estimate for f's venue and direction; a swap sell adds its cap solves and extra funding passes
// (constant.go).
func (p *PoolSimulator) gas(f *fill, sell bool) int64 {
	g := p.Policy.withDefaults()
	venue := f.venue
	switch {
	case venue == VenueSwap && sell:
		gas := g.GasSwapSell + int64(f.capEvals)*g.GasSwapSellCapEval
		if f.passes > 1 {
			gas += int64(f.passes-1) * g.GasSwapSellPass
		}
		return gas
	case venue == VenueSwap:
		return g.GasSwapBuy
	case sell:
		return g.GasLeverUp
	}
	return g.GasLeverDown
}

// swapTokens is the pair in the direction of the fill.
func (p *PoolSimulator) swapTokens(sell bool) (common.Address, common.Address) {
	tokenIn, tokenOut := p.state.PoolAsset, p.state.Pool.Loans[0].Token
	if !sell {
		tokenIn, tokenOut = tokenOut, tokenIn
	}
	return tokenIn, tokenOut
}

// quoteSwap is the adapter's `swap(tokenIn, tokenOut, amountIn, 1, recipient, block.timestamp)` at now, guarded.
func (p *PoolSimulator) quoteSwap(sell bool, amountIn *uint256.Int, now uint64) (*fill, error) {
	f, err := p.settleSwap(sell, amountIn, now)
	if err != nil {
		return nil, err
	}
	return f, p.guardSwap(f, sell, amountIn, now)
}

// settleSwap is that settlement alone, with no guard: what the pool fills.
func (p *PoolSimulator) settleSwap(sell bool, amountIn *uint256.Int, now uint64) (*fill, error) {
	tokenIn, tokenOut := p.swapTokens(sell)
	r, post, err := p.state.executeSwap(tokenIn, tokenOut, amountIn, uOne, now, now)
	if err != nil {
		return nil, err
	}
	return &fill{venue: VenueSwap, used: r.AmountInUsed, out: r.AmountOut, fee: r.FeeOut, priceWad: r.PriceWad,
		capEvals: r.capEvals, passes: r.passes, next: post}, nil
}

// guardSwap is the configured policy on a settled swap: the price band less the margin, the margin's feed move,
// the market oracle's end-of-window answer and the accrual drift.
func (p *PoolSimulator) guardSwap(f *fill, sell bool, amountIn *uint256.Int, now uint64) error {
	s := p.state
	cfg := &s.Pool.Loans[0]
	if p.Policy.PriceBandMarginBps != 0 {
		var outValue, inValue uint256.Int
		var err error
		if sell {
			outValue, err = gateMul(&f.out, &cfg.Scale)
			if err == nil {
				inValue, err = gateMul(&f.used, &f.priceWad)
			}
		} else {
			outValue, err = gateMul(&f.out, &f.priceWad)
			if err == nil {
				inValue, err = gateMul(&f.used, &cfg.Scale)
			}
		}
		if err != nil {
			return err
		}
		band, ok := p.marginBand(&cfg.SwapPriceBandWad)
		if !ok || swapBand(&outValue, &inValue, &band) != nil {
			return ErrBandMargin
		}
		// A sell's size is priced at the feed as well: its ceiling is the gate room and the Router's funding, both
		// evaluated at the cross (FLAMMSwapLib.sol:164, FLAMMGateLib.sol:45-53, :248-252), so an unseen round moves
		// a clipped sell's amounts and can empty the room outright. A buy's ceiling is the gross poolAsset, which
		// carries no price, so the band above is the whole of its feed exposure.
		if sell {
			tokenIn, tokenOut := p.swapTokens(true)
			if err := p.feedMoved(f, func(alt *flammState) (uint256.Int, uint256.Int, error) {
				r, _, err := alt.executeSwap(tokenIn, tokenOut, amountIn, uOne, now, now)
				if err != nil {
					return uint256.Int{}, uint256.Int{}, err
				}
				return r.AmountInUsed, r.AmountOut, nil
			}); err != nil {
				return err
			}
		}
	}
	if alt := p.oracleShifted(); alt != nil {
		tokenIn, tokenOut := p.swapTokens(sell)
		r2, _, err := alt.executeSwap(tokenIn, tokenOut, amountIn, uOne, now, now)
		if err != nil || !r2.AmountInUsed.Eq(&f.used) || !r2.AmountOut.Eq(&f.out) {
			return ErrOracleDrift
		}
	}
	if d := p.Policy.DebtDriftSec; d != 0 {
		later := now + d
		tokenIn, tokenOut := p.swapTokens(sell)
		r2, _, err := s.executeSwap(tokenIn, tokenOut, amountIn, uOne, later, later)
		if err != nil || !r2.AmountInUsed.Eq(&f.used) || !r2.AmountOut.Eq(&f.out) {
			return ErrDebtDrift
		}
		// The same amounts can be reached through a different plan: charge the larger of each count.
		f.capEvals, f.passes = max(f.capEvals, r2.capEvals), max(f.passes, r2.passes)
	}
	return nil
}

// quoteLever is the adapter's `leverUp(amountIn, 1, recipient, block.timestamp)` for a sell and `leverDown` for a
// buy, at now, guarded.
func (p *PoolSimulator) quoteLever(up bool, amountIn *uint256.Int, now uint64) (*fill, error) {
	f, err := p.settleLever(up, amountIn, now)
	if err != nil {
		return nil, err
	}
	return f, p.guardLever(f, up, amountIn, now)
}

// settleLever is the leverage settlement alone. It is quoted only while the keeper's spread is live, in both
// directions: a lapsed spread refuses a lever-up and silently re-prices a lever-down at the stored degrade value,
// which is a fill this port will not quote. Whether it also has to stay live through SpreadAgeMarginSec is a
// guard, not part of the settlement.
func (p *PoolSimulator) settleLever(up bool, amountIn *uint256.Int, now uint64) (*fill, error) {
	s := p.state
	if amountIn.IsZero() {
		return nil, ErrInvalidAmount
	}
	if err := s.leverOpen(up, now); err != nil {
		return nil, err
	}
	if err := p.spreadLive(now, 0); err != nil {
		return nil, err
	}
	r, post, err := s.executeLever(up, amountIn, uOne, now, now)
	if err != nil {
		return nil, err
	}
	return &fill{venue: VenueLever, used: r.AmountInUsed, out: r.AmountOut, spread: r.SpreadPpm,
		priceWad: r.PriceWad, payL18: r.PayL18, next: post}, nil
}

// guardLever is the configured policy on a settled leverage fill.
//
// Three of its gates besides the taker band are priced at the feed and not at the curve's own reservation price --
// the lever-up value leak (FLAMMLeverLib.sol:108), the 1.82e18 physical CR floor valued at the cross (:112) and the
// lever-down concession ceiling (:138) -- so tightening the band alone leaves a fill sitting on one of them, and a
// single adverse round reverts it. The whole fill is therefore re-run at the pool asset's answer moved by the
// margin in both directions and refused unless it settles to the identical amounts.
func (p *PoolSimulator) guardLever(f *fill, up bool, amountIn *uint256.Int, now uint64) error {
	s := p.state
	if err := p.spreadLive(now, p.Policy.SpreadAgeMarginSec); err != nil {
		return err
	}
	if p.Policy.PriceBandMarginBps != 0 {
		cfg := &s.Pool.Loans[0]
		band, ok := p.marginBand(&cfg.SwapPriceBandWad)
		if !ok {
			return ErrBandMargin
		}
		var bandErr error
		if up {
			var worth uint256.Int
			if worth, bandErr = gateMul(&f.used, &f.priceWad); bandErr == nil {
				bandErr = leverBandFloor(&f.payL18, &worth, &band)
			}
		} else {
			var got uint256.Int
			if got, bandErr = gateMul(&f.out, &f.priceWad); bandErr == nil {
				bandErr = leverBandFloor(&got, &f.payL18, &band)
			}
		}
		if bandErr != nil {
			return ErrBandMargin
		}
		if err := p.feedMoved(f, func(alt *flammState) (uint256.Int, uint256.Int, error) {
			r, _, err := alt.executeLever(up, amountIn, uOne, now, now)
			if err != nil {
				return uint256.Int{}, uint256.Int{}, err
			}
			return r.AmountInUsed, r.AmountOut, nil
		}); err != nil {
			return err
		}
	}
	if alt := p.oracleShifted(); alt != nil {
		r2, _, err := alt.executeLever(up, amountIn, uOne, now, now)
		if err != nil || !r2.AmountInUsed.Eq(&f.used) || !r2.AmountOut.Eq(&f.out) {
			return ErrOracleDrift
		}
	}
	if d := p.Policy.DebtDriftSec; d != 0 {
		later := now + d
		if _, _, err := s.executeLever(up, amountIn, uOne, later, later); err != nil {
			return ErrDebtDrift
		}
	}
	return nil
}

// feedMoved re-runs settle on the state with the pool asset's Chainlink answer moved by PriceBandMarginBps, once
// up and once down, and refuses the fill (ErrFeedMoveMargin) unless both settle to the amounts already quoted.
// That is one ordinary feed round of the configured size, in its adverse half: a real round also writes a new
// roundId and updatedAt, which only make the round fresher and so can only relax PriceFeed._read, while the answer
// is what every gate the fill runs prices against through PriceFeed.cross at the quote's own clock.
func (p *PoolSimulator) feedMoved(f *fill, settle func(*flammState) (uint256.Int, uint256.Int, error)) error {
	for _, up := range []bool{false, true} {
		alt := p.feedShifted(up)
		if alt == nil {
			return ErrFeedMoveMargin
		}
		used, out, err := settle(alt)
		if err != nil || !used.Eq(&f.used) || !out.Eq(&f.out) {
			return ErrFeedMoveMargin
		}
	}
	return nil
}

// feedShifted is a copy of the state whose pool asset round answers PriceBandMarginBps higher or lower, the move
// rounded up so the copy covers at least the configured margin. nil when the move is not representable, which
// refuses the fill.
func (p *PoolSimulator) feedShifted(up bool) *flammState {
	c := p.state.clone()
	r := &c.Feed.Asset.Round
	var d uint256.Int
	if _, overflow := d.MulOverflow(&r.Answer, uint256.NewInt(p.Policy.PriceBandMarginBps)); overflow {
		return nil
	}
	moved, err := ozCeilDiv(&d, uBps)
	if err != nil {
		return nil
	}
	if up {
		if _, overflow := r.Answer.AddOverflow(&r.Answer, &moved); overflow {
			return nil
		}
		return c
	}
	if moved.Gt(&r.Answer) {
		return nil
	}
	r.Answer.Sub(&r.Answer, &moved)
	return c
}

// spreadLive: the pool's spread hook answers through now + margin with a ppm below PPM (spreadHookPort.liveThrough).
// A pool with no spread hook has no answer.
func (p *PoolSimulator) spreadLive(now, margin uint64) error {
	sp, err := p.state.Hooks.Spread.port()
	if err != nil || !sp.liveThrough(now, margin) {
		return ErrSpreadNotLive
	}
	return nil
}

// fresh refuses a snapshot older than MaxSnapshotAgeSec at now, and every quote from scheduledChangeLeadSec before
// a scheduled implementation, hook-set, venue or loan-asset change becomes executable (scheduledChangeAt).
func (p *PoolSimulator) fresh(now uint64) error {
	if m := p.Policy.MaxSnapshotAgeSec; m != 0 && now > p.snapshotTs && now-p.snapshotTs > m {
		return ErrSnapshotStale
	}
	if p.scheduledAt != 0 && now+scheduledChangeLeadSec >= p.scheduledAt {
		return ErrScheduledChange
	}
	return nil
}

// feedMargin refuses once either feed round the fill reads (pool asset, loan asset 0) is stale at
// now + PriceAgeMarginSec: PriceFeed._read's `block.timestamp - updatedAt > heartbeat`.
func (p *PoolSimulator) feedMargin(now uint64) error {
	margin := p.Policy.PriceAgeMarginSec
	if margin == 0 {
		return nil
	}
	f := &p.state.Feed
	for _, t := range []*feedToken{&f.Asset, &f.Loans[0]} {
		if !t.Known || !t.Round.Ok {
			continue // the fill itself refuses
		}
		var deadline, at uint256.Int
		deadline.Add(&t.Round.UpdatedAt, &t.Heartbeat)
		at.SetUint64(now)
		if _, overflow := at.AddOverflow(&at, uint256.NewInt(margin)); overflow || at.Gt(&deadline) {
			return ErrFeedAgeMargin
		}
	}
	return nil
}

// marginBand is the loan asset's band less PriceBandMarginBps (1 bp = 1e14 WAD); false when nothing is left.
func (p *PoolSimulator) marginBand(band *uint256.Int) (uint256.Int, bool) {
	var m, out uint256.Int
	if _, overflow := m.MulOverflow(uint256.NewInt(p.Policy.PriceBandMarginBps), uint256.NewInt(1e14)); overflow ||
		!m.Lt(band) {
		return out, false
	}
	return *out.Sub(band, &m), true
}

// lineageOf is the token of a freshly built snapshot: the pool's address and the entity's whole Extra, which is
// the refresh's published state -- its reads, venue set, oracle answers and policy. It is a value, not an
// identity, so two simulators built from the same entity -- pool-service's and router-service's, or one rebuilt
// after a restart -- agree on it and accept each other's SwapInfo, which a per-construction random token did not,
// while any entity that differs in a single byte is a different state. It is computed once per construction, next
// to the two JSON decodes of the same bytes.
func lineageOf(address, extra string) uint64 {
	h := fnvOffset64
	for _, s := range [2]string{address, extra} {
		for i := 0; i < len(s); i++ {
			h = (h ^ uint64(s[i])) * fnvPrime64
		}
	}
	return h
}

// adoptLineage is the token of the state a fill leaves: the token of the state it was quoted on folded with the
// fill itself -- everything the settlement that produced the post-state was a function of besides the prior
// state: venue, direction, requested input and clock, plus the amounts it settled to -- so the chain of adopted
// fills is what a SwapInfo is matched against. Two simulators that adopted the same fills from the same entity
// agree; a simulator that adopted any other fill does not.
func adoptLineage(prev uint64, si *SwapInfo) uint64 {
	dir := uint64(0)
	if si.PoolAssetIn {
		dir = 1
	}
	h := fnvWord(prev, uint64(si.Venue)|dir<<8)
	h = fnvWords(h, &si.amountIn)
	h = fnvWords(h, &si.AmountInUsed)
	h = fnvWords(h, &si.AmountOut)
	return fnvWord(h, si.next.Timestamp)
}

// UpdateBalance adopts the quoted fill's post-state (a copy of it: one SwapInfo may be applied to several clones).
// A SwapInfo identifies the state it was quoted on by that state's lineage token, so it is accepted by every copy
// of that state (clones, a msgpack round trip, a second simulator built from the same entity) and by nothing else:
// not by a state that adopted another fill since, even at the same count of transitions. One that was not quoted on
// this state leaves the simulator refusing (ErrSwapInfoMismatch), since adopting or ignoring it would quote
// liquidity the route already spent. The route finder applies UpdateBalance only to its per-request clones, so the
// refusal ends with that request; the tracked simulator is replaced by the next refresh.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok || si.next == nil || p.state == nil || si.lineage != p.lineage || si.next.Block != p.state.Block {
		p.broken = true
		return
	}
	p.state = si.next.clone()
	p.lineage = adoptLineage(p.lineage, &si)
	p.seq++
	// Both published reserves follow the adopted state: the gross poolAsset, and the sell capacity recomputed from
	// it exactly as the refresh computes it (pool_tracker.go sellCapacity), since a fill moves the room, the liquid
	// and the funding the capacity is built from. The clock is the adopted state's own -- the one the fill settled
	// at (fill.next.Timestamp) -- and not the wall clock, so what a simulator publishes is a function of the state
	// it adopted alone: two clones handed the same SwapInfo publish the same reserves.
	now := p.state.Timestamp
	if pos, err := p.state.Router.positions(now); err == nil && len(p.Info.Reserves) == 2 {
		var gross uint256.Int
		if _, overflow := gross.AddOverflow(&p.state.Pool.Physical, &pos.TotalColl); !overflow {
			capacity := sellCapacity(p.state, now)
			p.Info.Reserves = []*big.Int{gross.ToBig(), capacity.ToBig()}
		}
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	c := *p
	if p.state != nil {
		c.state = p.state.clone()
	}
	c.Info.Reserves = lo.Map(p.Info.Reserves, func(r *big.Int, _ int) *big.Int {
		if r == nil {
			return nil
		}
		return new(big.Int).Set(r)
	})
	return &c
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{ApprovalAddress: p.Info.Address, BlockNumber: p.Info.BlockNumber}
}

// GetApprovalAddress: the pool pulls the input with transferFrom.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Info.Address
}

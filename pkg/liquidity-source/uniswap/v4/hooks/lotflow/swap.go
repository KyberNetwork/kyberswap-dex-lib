package lotflow

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// The whole venue swap (A4): exact-input only, as Kyber's executor sends it (PRD §4.1).
//
//	buy  (USDG in):  NAV -> inventory -> per constituent pick+leg (exact-out) -> mint -> place JIT ->
//	                 user swap -> remove -> band -> float -> redeem leftover -> memo/probed sells -> profit
//	sell (LOT in):   NAV -> price/inventory -> place JIT -> user swap -> remove -> band -> float ->
//	                 redeem -> probed sells (exact-in) -> profit
//
// Every leg runs on the state the previous legs left: a constituent's buy leg moves its pool, and the
// leftover (memo) sell of that constituent then trades the moved pool.

// Hook callback phases (the selector WrappedError carries).
const (
	PhaseBeforeSwap = "0x575e24b4"
	PhaseAfterSwap  = "0xb47b2fb1"
)

// SwapError is a simulated revert: the hook error (Reason) and the callback it reverts in. The
// PoolManager wraps it as WrappedError(hook, Phase, reason, HookCallFailed).
type SwapError struct {
	Phase  string
	Reason error
}

func (e *SwapError) Error() string {
	ph := "beforeSwap"
	if e.Phase == PhaseAfterSwap {
		ph = "afterSwap"
	}
	return fmt.Sprintf("lotflow: %s: %v", ph, e.Reason)
}
func (e *SwapError) Unwrap() error { return e.Reason }

// errorSignatures: the Solidity signature of every NamedError the simulator returns.
var errorSignatures = map[string]string{
	"TradeTooSmall":      "TradeTooSmall()",
	"ZeroNav":            "ZeroNav()",
	"BadFeedAnswer":      "BadFeedAnswer(address)",
	"OutsideBand":        "OutsideBand(uint256,uint256)",
	"NotProfitable":      "NotProfitable(int256)",
	"SellExceedsFloat":   "SellExceedsFloat(uint256,uint256)",
	"NavMismatch":        "NavMismatch(uint256,uint256)",
	"FeedCoverageTooLow": "FeedCoverageTooLow(uint256,uint256)",
	"NoExecutableSource": "NoExecutableSource(address)",
	"LegShort":           "LegShort(address,uint256,uint256)",
}

// Selector is the 4-byte custom-error selector of a NamedError ("" if unknown).
func Selector(err error) string {
	var n NamedError
	if !errors.As(err, &n) {
		return ""
	}
	if s, ok := n.(interface{ ErrorSignature() string }); ok {
		return fmt.Sprintf("0x%x", crypto.Keccak256([]byte(s.ErrorSignature()))[:4])
	}
	sig, ok := errorSignatures[n.ErrorName()]
	if !ok {
		return ""
	}
	return fmt.Sprintf("0x%x", crypto.Keccak256([]byte(sig))[:4])
}

// MarketConstituent is one basket constituent with everything its legs need.
type MarketConstituent struct {
	Stock    string
	Unit     *big.Int
	Nav      NavConstituent
	Registry RegistryEntry
	VenuesOK bool
	Venues   []BookVenueView
	// Pools: state of every source the admission may return, by Source.ID (lowercase).
	Pools map[string]*LegPool
	// StableDecimals of the registry stable; when set, spot NAV is recomputed from the registry price
	// pool's CURRENT state if that pool is in Pools (a constituent leg or plain swap moved it).
	StableDecimals int
	// Mq: set for a WETH-quoted token (registry stable = WETH) on an MQ venue.
	Mq *MqToken
}

// Market is the pre-swap state of one LOT venue and everything behind it.
type Market struct {
	Lot          string
	Venue        *Venue
	Constituents []MarketConstituent // Basket.constituents() order
	NavParams    NavGuardParams
	Timestamp    uint64   // block.timestamp
	PmLotBalance *big.Int // PoolManager's LOT ERC-20 balance before the swap (the sell float)
	// Mq: the MQ route and pricing state (nil on a roster venue).
	Mq *MqMarket
}

// SwapInput is one exact-input swap on the venue.
type SwapInput struct {
	BuyLot   bool
	AmountIn *big.Int
	// SqrtPriceLimitX96 of the venue swap; nil = the extreme Kyber sends (MIN+1 / MAX-1).
	SqrtPriceLimitX96 *big.Int
	// Prefund: the router settles the input before the swap (Kyber's executor). On a sell that LOT
	// is in the PoolManager float by unwind time; pay-after it is not.
	Prefund bool
}

// ExecutedLeg mirrors one LegExecuted event.
type ExecutedLeg struct {
	Stock            string
	ConstituentIndex int
	CandidateIndex   int // index into the admitted list at the time the leg ran
	Source           Source
	Buy              bool
	Paid, Got        *big.Int
	Mode             string // "probed" | "single" | "memo"
	// Gas features (SwapGasFor): candidates probed (0 = executed unprobed), the probes' summed leg-gas
	// upper bounds (each capped at PROBE_GAS, the stipend), the probes that may exhaust the stipend
	// (StipendProbes: GasSkip, or an envelope reaching PROBE_GAS - InexactGasMargin), and the executed
	// leg's ticks and bound.
	Probes        int
	ProbeGasHi    uint64
	StipendProbes int
	CrossedTicks  int
	GasHi         uint64
}

// SwapResult is the simulated outcome.
type SwapResult struct {
	AmountOut   *big.Int
	Treasury    *big.Int // hook USDG surplus taken to the treasury
	Nav         *big.Int
	Inventory   *big.Int
	Price18     *big.Int
	Position    *Position
	Venue       *VenueSwapResult
	Legs        []ExecutedLeg
	LotRedeemed *big.Int
	// Standing: what the swap did to the venue's standing position (nil = none).
	Standing *StandingResult
	// PostSqrtPriceX96 / PostTick: venue slot0 after the swap (the standing restore's centre when
	// there is a standing position, else where the user's swap ended).
	PostSqrtPriceX96 *big.Int
	PostTick         int
	// PmLotAfter: the PoolManager's LOT balance after the swap (settles in, takes out).
	PmLotAfter *big.Int
	// Inexact: some probe verdict rested on the gas envelope or an unknown input, not exact maths.
	Inexact  bool
	post     []map[string]*LegPool // constituent pool states after the swap
	convPost *LegPool              // shared USDG/WETH conversion pool after an MQ swap
}

// marketState is the mutable per-swap copy of the constituents' pool states.
type marketState struct {
	m     *Market
	pools []map[string]*LegPool
	conv  *LegPool // MQ conversion pool, shared by every WETH-token leg
}

func newMarketState(m *Market) *marketState {
	s := &marketState{m: m, pools: make([]map[string]*LegPool, len(m.Constituents))}
	for i, c := range m.Constituents {
		s.pools[i] = make(map[string]*LegPool, len(c.Pools))
		for id, p := range c.Pools {
			cp := *p
			s.pools[i][strings.ToLower(id)] = &cp
		}
	}
	if m.Mq != nil && m.Mq.Conv != nil {
		cp := *m.Mq.Conv
		s.conv = &cp
	}
	return s
}

// sources is NavJitLegs.sources() at the current state: the pre-state admission, re-checked against
// the live price/liquidity of any pool this swap has already traded (only our own legs move them).
func (s *marketState) sources(cfg *Config, i int) ([]Source, []*LegPool, error) {
	c := &s.m.Constituents[i]
	if c.Mq != nil {
		q := s.m.Mq
		pool := s.pools[i][v3SourceID(c.Mq.Pool)]
		srcs := mqSources(cfg, c.Stock, c.Registry.Stable, q, c.Mq, pool, s.conv)
		pools := make([]*LegPool, len(srcs))
		for j := range srcs {
			cp := *pool
			cp.Conv, cp.WethIs0, cp.PmWeth = s.conv, q.wethIs0(), q.PmWeth
			pools[j] = &cp
		}
		return srcs, pools, nil
	}
	reg := c.Registry
	reg.Pools = append([]RegistryPoolView(nil), c.Registry.Pools...)
	for j := range reg.Pools {
		if p, ok := s.pools[i][strings.ToLower(reg.Pools[j].PoolID)]; ok {
			reg.Pools[j].SqrtPriceX96, reg.Pools[j].Liquidity = p.SqrtPriceX96, p.Liquidity
		}
	}
	venues := append([]BookVenueView(nil), c.Venues...)
	for j := range venues {
		if p, ok := s.pools[i][v3SourceID(venues[j].Pool)]; ok && venues[j].SqrtOK && venues[j].LiqOK {
			venues[j].SqrtPriceX96, venues[j].Liquidity = p.SqrtPriceX96, p.Liquidity
		}
	}
	srcs, _, _ := Sources(cfg, c.Stock, reg, c.VenuesOK, venues)
	pools := make([]*LegPool, len(srcs))
	for j, src := range srcs {
		p, ok := s.pools[i][strings.ToLower(src.ID)]
		if !ok {
			return nil, nil, fmt.Errorf("lotflow: no pool state for source %s of %s", src.ID, c.Stock)
		}
		pools[j] = p
	}
	return srcs, pools, nil
}

// apply moves the executed source's pool to its post-leg state (initialized ticks are unchanged by
// a swap; only slot0 and in-range liquidity move).
func (s *marketState) apply(i int, src Source, r *LegResult) {
	p := s.pools[i][strings.ToLower(src.ID)]
	cp := *p
	cp.SqrtPriceX96, cp.Tick, cp.Liquidity = r.SqrtPriceX96, r.Tick, r.Liquidity
	cp.Conv = nil
	s.pools[i][strings.ToLower(src.ID)] = &cp
	if r.ConvPost != nil && s.conv != nil {
		cv := *s.conv
		cv.SqrtPriceX96, cv.Tick, cv.Liquidity = r.ConvPost.SqrtPriceX96, r.ConvPost.Tick, r.ConvPost.Liquidity
		s.conv = &cv
	}
}

// Quote simulates one exact-input swap. A revert comes back as *SwapError wrapping the hook's
// NamedError (use Selector / errors.As); the partial SwapResult is returned alongside it.
func Quote(cfg *Config, m *Market, in SwapInput) (*SwapResult, error) {
	res := &SwapResult{}
	before := func(err error) (*SwapResult, error) { return res, &SwapError{Phase: PhaseBeforeSwap, Reason: err} }
	after := func(err error) (*SwapResult, error) { return res, &SwapError{Phase: PhaseAfterSwap, Reason: err} }
	v := m.Venue
	st := newMarketState(m)

	units := make([]*big.Int, len(m.Constituents))
	for i, c := range m.Constituents {
		units[i] = c.Unit
	}
	nav, err := st.checkedNav()
	if err != nil {
		return before(err)
	}
	res.Nav = nav
	// _liftStanding precedes _quote: the user never trades standing liquidity
	if res.Standing, err = v.liftStanding(); err != nil {
		return res, err
	}
	a := in.AmountIn
	memo := map[int]string{} // constituent -> source id the buy leg used (R5)
	cost := new(big.Int)

	// -------- beforeSwap: _quote (+ mintLot on a buy) and _place
	if in.BuyLot {
		inv, err := v.BuyInventory(a, nav)
		if err != nil {
			return before(err)
		}
		res.Inventory = inv
		amts := MintAmounts(units, inv)
		for i := range m.Constituents {
			if amts[i].Sign() == 0 {
				continue
			}
			leg, r, inexact, err := st.runLeg(cfg, i, true, amts[i], "")
			res.Inexact = res.Inexact || inexact
			if err != nil {
				return before(err)
			}
			st.apply(i, leg.Source, r)
			memo[i] = strings.ToLower(leg.Source.ID)
			cost.Add(cost, leg.Paid)
			res.Legs = append(res.Legs, *leg)
		}
		res.Price18 = v.BuyPrice18(cost, inv)
	} else {
		res.Price18 = v.SellPrice18(nav)
		inv, err := v.SellInventory(a, res.Price18)
		if err != nil {
			return before(err)
		}
		res.Inventory = inv
	}
	pos, err := v.Place(res.Price18, res.Inventory, in.BuyLot)
	if err != nil {
		return before(err)
	}
	res.Position = pos

	// -------- the user's swap
	limit := in.SqrtPriceLimitX96
	if limit == nil {
		if v.ZeroForOne(in.BuyLot) {
			limit = new(big.Int).Add(MinSqrtPrice, big.NewInt(1))
		} else {
			limit = new(big.Int).Sub(MaxSqrtPrice, big.NewInt(1))
		}
	}
	sw, err := v.Swap(pos, in.BuyLot, a, limit)
	if err != nil {
		return res, err
	}
	res.Venue = sw
	res.AmountOut = sw.AmountOut

	// -------- afterSwap: _removeJit, _requireInBand, _unwind
	if err := v.RequireInBand(sw.SqrtPriceX96, nav); err != nil {
		return after(err)
	}
	res.PostSqrtPriceX96, res.PostTick = sw.SqrtPriceX96, sw.Tick
	pmLot := new(big.Int).Set(m.PmLotBalance)
	if res.Standing != nil {
		// _restoreStanding runs before _unwind; its leftovers leave the PoolManager to owner()
		if err := v.restoreStanding(res.Standing, nav, sw.SqrtPriceX96, sw.Tick); err != nil {
			return res, err
		}
		res.PostSqrtPriceX96, res.PostTick = res.Standing.SqrtPriceX96, res.Standing.Tick
		_, ownerLot := sw.usdgLot(res.Standing.ToOwner0, res.Standing.ToOwner1)
		pmLot.Sub(pmLot, ownerLot)
	}
	lotLeft := sw.LotLeftOnHook(in.BuyLot, res.Inventory)
	_, lotRemoved := sw.usdgLot(sw.Removed0, sw.Removed1)
	if lotRemoved.Sign() > 0 {
		float := new(big.Int).Set(pmLot) // a copy: pmLot feeds PmLotAfter below
		if in.BuyLot {
			_, lotAdded := sw.usdgLot(pos.Added0, pos.Added1)
			float.Add(float, lotAdded) // the minted LOT the hook settled into the position
		} else if in.Prefund {
			float.Add(float, a)
		}
		if err := RequireFloat(float, lotRemoved); err != nil {
			return after(err)
		}
	}
	res.LotRedeemed = lotLeft
	proceeds := new(big.Int)
	if lotLeft.Sign() > 0 {
		sells := RedeemAmounts(units, lotLeft)
		for i := range m.Constituents {
			if sells[i].Sign() == 0 {
				continue
			}
			leg, r, inexact, err := st.runLeg(cfg, i, false, sells[i], memo[i])
			res.Inexact = res.Inexact || inexact
			if err != nil {
				return after(err)
			}
			st.apply(i, leg.Source, r)
			proceeds.Add(proceeds, leg.Got)
			res.Legs = append(res.Legs, *leg)
		}
	}
	delta := sw.HookUsdgDelta(in.BuyLot)
	delta.Sub(delta, cost)
	delta.Add(delta, proceeds)
	if err := RequireProfitable(delta); err != nil {
		return after(err)
	}
	res.Treasury = delta
	// PoolManager LOT: buy = + minted LOT settled into the position - user's take - hook's take of the
	// removed LOT; sell = + user's LOT - hook's take; both - the standing leftover taken to owner().
	pmAfter := new(big.Int).Set(pmLot) // already net of the owner take
	if in.BuyLot {
		_, lotAdded := sw.usdgLot(pos.Added0, pos.Added1)
		pmAfter.Add(pmAfter, lotAdded)
		pmAfter.Sub(pmAfter, sw.AmountOut)
	} else {
		pmAfter.Add(pmAfter, sw.AmountIn)
	}
	pmAfter.Sub(pmAfter, lotRemoved)
	res.PmLotAfter = pmAfter
	res.post = st.pools
	res.convPost = st.conv
	return res, nil
}

// checkedNav is shared by the swap and its executor price limit. It uses the current dependency
// state, including changes applied by UpdateBalance, and the snapshot's timestamp for MQ TWAPs.
func (s *marketState) checkedNav() (*big.Int, error) {
	m := s.m
	cs := make([]NavConstituent, len(m.Constituents))
	for i, c := range m.Constituents {
		cs[i] = c.Nav
		if c.Mq != nil {
			sp, err := MqTokenUsdg18(m.Mq, c.Mq, m.Timestamp) // MqFactory.spotUsd18
			if err != nil {
				return nil, err
			}
			cs[i].SpotUsd18 = sp
		} else if sp, ok := s.spot(i); ok {
			cs[i].SpotUsd18 = sp
		}
	}
	return CheckedNavPerUnit18(cs, m.NavParams, m.Timestamp)
}

// spot is constituent i's spot NAV price from its registry price pool's current state, when that
// pool is one this swap's state carries.
func (s *marketState) spot(i int) (*big.Int, bool) {
	c := &s.m.Constituents[i]
	if c.StableDecimals == 0 || c.Registry.PriceIndex >= len(c.Registry.Pools) {
		return nil, false
	}
	p, ok := s.pools[i][strings.ToLower(c.Registry.Pools[c.Registry.PriceIndex].PoolID)]
	if !ok {
		return nil, false
	}
	sp, err := SpotUsd18(p.SqrtPriceX96, c.Registry.StockIsCurrency0, c.Nav.StockDecimals, c.StableDecimals)
	return sp, err == nil
}

// runLeg is one constituent leg: sources at the current state, then the memo source (leftover sell
// of a buy, unprobed, if still admitted) or _pick, then the leg itself.
func (s *marketState) runLeg(cfg *Config, i int, buy bool, amount *big.Int, memoID string) (*ExecutedLeg,
	*LegResult, bool, error) {
	stock := s.m.Constituents[i].Stock
	srcs, pools, err := s.sources(cfg, i)
	if err != nil {
		return nil, nil, false, err
	}
	if len(srcs) == 0 {
		return nil, nil, false, &NoExecutableSourceError{Stock: stock}
	}
	leg := &ExecutedLeg{Stock: stock, ConstituentIndex: i, Buy: buy}
	idx := -1
	if memoID != "" {
		for j, src := range srcs {
			if strings.ToLower(src.ID) == memoID {
				idx = j
				break
			}
		}
	}
	var r *LegResult
	inexact := false
	if idx >= 0 {
		// memo: executed unprobed on the buy's source; a short leg reverts with its own error
		r = SimulateLeg(cfg, srcs[idx], pools[idx], buy, amount)
		inexact = errors.Is(r.Err, ErrUnknownDynamicFee)
		if r.Err != nil || !r.Fill {
			return nil, r, inexact, &LegShortError{Stock: stock, Wanted: new(big.Int).Set(amount), Got: r.Paid,
				Underlying: r.Err}
		}
		leg.Mode = "memo"
	} else {
		pick, err := Pick(cfg, stock, srcs, pools, buy, amount)
		if pick != nil {
			inexact = pick.Inexact
		}
		if err != nil {
			return nil, nil, inexact, err
		}
		idx, r = pick.Index, pick.Results[pick.Index]
		leg.Mode = "single"
		if pick.Probed {
			leg.Mode = "probed"
			leg.Probes = len(pick.Results)
			for _, c := range pick.Results {
				leg.ProbeGasHi += min(c.GasHi, cfg.ProbeGas)
				if cfg.mayExhaustStipend(c) {
					leg.StipendProbes++
				}
			}
		}
	}
	leg.CandidateIndex, leg.Source, leg.Paid, leg.Got = idx, srcs[idx], r.Paid, r.Got
	leg.CrossedTicks, leg.GasHi = r.CrossedTicks, r.GasHi
	return leg, r, inexact, nil
}

// QuoteConservative never promises more than the chain can deliver: an exact quote is returned as
// is; an Inexact one is also quoted with near-stipend probes skipped, and the worse of the two wins
// (a revert in either is a refusal).
func QuoteConservative(cfg *Config, m *Market, in SwapInput) (*SwapResult, error) {
	res, err := Quote(cfg, m, in)
	if err != nil || !res.Inexact {
		return res, err
	}
	skip := *cfg
	skip.NearStipend = NearStipendSkip
	alt, altErr := Quote(&skip, m, in)
	if altErr != nil {
		return alt, altErr
	}
	if alt.AmountOut.Cmp(res.AmountOut) < 0 {
		return alt, nil
	}
	return res, nil
}

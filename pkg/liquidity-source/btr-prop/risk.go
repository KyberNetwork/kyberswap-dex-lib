package btrprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// getSwapQuote omits checks swap_qe runs at settle: the feed gate, the depeg band, the route's
// feed-distinctness rule and the liquid-reserve floor. They are mirrored from state the tracker
// reads in the same multicall, so a pair that would revert is refused, never quoted. The leg
// word layout is the pool's published oracle word: obs 32..63 | sigma 64..90 | conf 91..106 |
// ttl 107..122 | halt 134 | INTERNAL 232 | depeg code 234..237.

const (
	depegStepBps = 250 // depeg band step per code
	bps          = 10_000
	pbps         = 1_000_000
	// maxConfidenceBps is the confidence interval above which the pool refuses a feed.
	maxConfidenceBps = 1000
	// Staleness premium constants. They mirror pool parameters: re-check them on a pool upgrade.
	// premium = staleZ*sigma*sqrt(excess)/bps in PBPS, excess = seconds the mark is older than
	// min(ttl/2, maxStaleAllowance). The base and INTERNAL legs pay none.
	staleZ            = 472
	maxStaleAllowance = 3
)

// rpcLegMark mirrors Pool.getLegMark's three outputs.
type rpcLegMark struct {
	Word   *big.Int `abi:"word"`
	Priced bool     `abi:"priced"`
	PxWad  *big.Int `abi:"pxWad"`
}

// markLeg is a decoded leg word.
type markLeg struct {
	obs      uint64
	ttl      uint64
	sigma    uint64 // PBPS
	conf     uint64 // mark confidence interval, bps
	mark     uint256.Int
	halted   bool
	internal bool
	depegBps uint64 // code*DEPEG_STEP_BPS, 0 = no depeg code
	// base is the pool's numeraire: exempt from the depeg breaker and the staleness premium.
	base bool
}

// field is the width-bit field of w at shift.
func field(w *uint256.Int, shift uint, width uint) uint64 {
	var t uint256.Int
	return t.Rsh(w, shift).Uint64() & (1<<width - 1)
}

// decodeMark decodes a leg word; false when the read failed or is empty.
func decodeMark(word *big.Int) (*markLeg, bool) {
	var w uint256.Int
	if word == nil || w.SetFromBig(word) || w.IsZero() {
		return nil, false
	}
	f := field(&w, 0, 32) // the lane float: mant u25 | exp7, mark = (mant << exp) >> 16
	l := &markLeg{
		obs:      field(&w, 32, 32),
		sigma:    field(&w, 64, 27),
		conf:     field(&w, 91, 16),
		ttl:      field(&w, 107, 16),
		halted:   field(&w, 134, 1) != 0,
		internal: field(&w, 232, 1) != 0,
		depegBps: field(&w, 234, 4) * depegStepBps,
	}
	l.mark.Lsh(uint256.NewInt(f&(1<<25-1)), uint(f>>25&0x7f)).Rsh(&l.mark, 16)
	// pegTested: INTERNAL or a depeg code; INTERNAL without a code reads the first step.
	if l.internal && l.depegBps == 0 {
		l.depegBps = depegStepBps
	}
	return l, true
}

// ageAt is the leg's mark age at unix time t.
func (l *markLeg) ageAt(t uint64) uint64 {
	if t > l.obs {
		return t - l.obs
	}
	return 0
}

// stale is the leg's staleness premium inputs. The base and INTERNAL legs pay none: sigma 0.
func (l *markLeg) stale() StaleLeg {
	s := StaleLeg{Obs: l.obs, Grace: min(l.ttl/2, maxStaleAllowance)}
	if !l.base && !l.internal {
		s.Sigma = l.sigma
	}
	return s
}

// staleTermAt is the leg's staleness premium in PBPS at unix time t.
func (l *markLeg) staleTermAt(t uint64) uint64 { return l.stale().termAt(t) }

// StaleLeg replays one leg's staleness premium at any block time: the mark's observation time,
// its sigma in PBPS (0 when the leg pays no premium) and the keeper's allowance in seconds.
type StaleLeg struct {
	Obs   uint64 `json:"o"`
	Sigma uint64 `json:"s,omitempty"`
	Grace uint64 `json:"g,omitempty"`
}

// excessAt is the seconds the mark is stale past the keeper's allowance at unix time t.
func (l StaleLeg) excessAt(t uint64) uint64 {
	if t > l.Obs && t-l.Obs > l.Grace {
		return t - l.Obs - l.Grace
	}
	return 0
}

// termAt is the premium in PBPS at unix time t: staleZ*sigma*isqrt(excess)/bps.
func (l StaleLeg) termAt(t uint64) uint64 {
	if l.Sigma == 0 {
		return 0
	}
	return staleZ * l.Sigma * new(uint256.Int).Sqrt(uint256.NewInt(l.excessAt(t))).Uint64() / bps
}

// PairCut sizes a pair's haircut at quote time: Legs are the token indices the pool prices the
// pair through, Den is 2e6 less the largest spread sampled on the pair.
type PairCut struct {
	Legs []uint16 `json:"m,omitempty"`
	Den  uint64   `json:"d,omitempty"`
}

// factor returns num/den, the share of a quote sampled at block time t0 that the pool still pays
// w seconds later with no push in between: the pool charges half the spread on the output,
// out = gross*(2e6 - spread)/2e6, so a spread that grows by d PBPS scales the output by
// (2e6 - spread - d)/(2e6 - spread). d is the path's premium growth over w, plus 1 PBPS for the
// pool rounding its fee up. ok is false when the growth reaches the whole spread.
func (c PairCut) factor(legs []StaleLeg, t0, w uint64) (num, den uint64, ok bool) {
	d := uint64(1)
	for _, i := range c.Legs {
		if int(i) < len(legs) {
			d += legs[i].termAt(t0+w) - legs[i].termAt(t0)
		}
	}
	if d >= c.Den {
		return 0, 0, false
	}
	return c.Den - d, c.Den, true
}

// pegTested reports whether the depeg breaker applies to the leg.
func (l *markLeg) pegTested() bool { return l.internal || l.depegBps != 0 }

// tradableAt mirrors the pool's feed gate and depeg band for a swap executed at unix time t: a
// halted, dead, uncertain or TTL-expired feed reverts (INTERNAL legs included, their lane word
// is gated at settle), and so does a depeg-coded (or INTERNAL) non-base leg whose mark sits
// outside its band.
func (l *markLeg) tradableAt(t uint64) bool {
	if l.halted || l.mark.IsZero() || l.conf > maxConfidenceBps || l.ageAt(t) > l.ttl {
		return false
	}
	if l.base || !l.pegTested() {
		return true
	}
	// |mark - 1.0| * BPS <= WAD * band; mark < 2^152 so the products fit.
	var dev uint256.Int
	if l.mark.Gt(covWad) {
		dev.Sub(&l.mark, covWad)
	} else {
		dev.Sub(covWad, &l.mark)
	}
	var lhs, rhs uint256.Int
	return !lhs.Mul(&dev, uint256.NewInt(bps)).Gt(rhs.Mul(covWad, uint256.NewInt(l.depegBps)))
}

// pathLegs returns the distinct pool-token indices a swap prices through: the two endpoints and
// every quoted route hop that is a pool token.
func pathLegs(tokens []common.Address, from, to int, hops []common.Address) []int {
	legs := []int{from, to}
	for _, h := range hops {
		for i, t := range tokens {
			if t == h && i != from && i != to && !containsInt(legs, i) {
				legs = append(legs, i)
			}
		}
	}
	return legs
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// liquidRoom is the most the pool can pay out of a token at settle: liquid reserves less the
// asset's minLiquidity floor (settle needs liquid >= out + minLiquidity).
func liquidRoom(liquid *big.Int, a *rpcAsset) *uint256.Int {
	var liq, floor uint256.Int
	if liquid == nil || a == nil || a.MinLiquidity == nil || liq.SetFromBig(liquid) || floor.SetFromBig(a.MinLiquidity) {
		return nil
	}
	if !liq.Gt(&floor) {
		return new(uint256.Int)
	}
	return liq.Sub(&liq, &floor)
}

// cutAbove drops the trailing samples whose output exceeds room.
func cutAbove(pts []Point, room *uint256.Int) []Point {
	if room == nil {
		return pts
	}
	for len(pts) > 0 && pts[len(pts)-1].Out.Gt(room) {
		pts = pts[:len(pts)-1]
	}
	return pts
}

// guard holds the state the settle-time checks read, for one refresh at block time now. A swap
// quoted from the refresh may execute until now+window (MaxAge plus InclusionAllowance).
type guard struct {
	tokens []common.Address
	legs   []*markLeg // nil = unreadable word
	// assets[i] is the token's book (anchor, oracle lane), nil = unreadable.
	assets []*rpcAssetResult
	base   common.Address
	// rooms[i] is the settle-time output room of token i (liquidRoom), nil = unknown.
	rooms       []*uint256.Int
	now, window uint64
}

// legClosed reports whether swap_qe would revert on leg i by the time a quote may execute: its
// word is unreadable, or the leg is halted, dead, uncertain, TTL-expired or outside its depeg band.
func (g *guard) legClosed(i int) bool {
	return g.legs[i] == nil || !g.legs[i].tradableAt(g.now+g.window)
}

// distinctFeeds mirrors the pool's route rule: walking the route up the anchor tree to its turn
// node, no ascending chain and no descending chain may price two of its legs off one oracle lane.
// hops are token indices in route order. False also when a book is unreadable.
func (g *guard) distinctFeeds(hops []int) bool {
	n := len(hops)
	anchor := func(i int) (common.Address, bool) {
		if g.assets[hops[i]] == nil {
			return common.Address{}, false
		}
		return g.assets[hops[i]].Asset.Anchor, true
	}
	turn := 0
	for turn+1 < n && g.tokens[hops[turn]] != g.base {
		a, ok := anchor(turn)
		if !ok {
			return false
		}
		if a != g.tokens[hops[turn+1]] {
			break
		}
		turn++
	}
	chain := func(from, to int) bool {
		var seen uint64
		for i := from; i < to; i++ {
			a := g.assets[hops[i]]
			if a == nil {
				return false
			}
			bit := uint64(1) << (a.Asset.OracleBits & 63)
			if seen&bit != 0 {
				return false
			}
			seen |= bit
		}
		return true
	}
	return chain(0, turn) && chain(turn+1, n)
}

// routeIndices maps the quoted route to token indices; false when a hop is not a pool token.
func routeIndices(tokens []common.Address, hops []common.Address) ([]int, bool) {
	res := make([]int, len(hops))
	for k, h := range hops {
		res[k] = -1
		for i, t := range tokens {
			if t == h {
				res[k] = i
			}
		}
		if res[k] < 0 {
			return nil, false
		}
	}
	return res, true
}

// refine returns the ladder of from -> to restricted to what swap_qe would settle and pay within
// the window, with the pair's haircut inputs, or nil when every size would revert: a leg on the
// path (other than the output leg, whose reserve the tracker zeroes so the ladder survives) is
// halted, dead, uncertain, TTL-expired by now+window or outside its depeg band, its word is
// unreadable, or the route prices two legs off one feed. Samples whose output exceeds the output
// leg's liquid room are cut.
//
// The pool's quote drifts with block time: each non-base leg's staleness premium, part of the
// quoted spread, grows with the square root of its mark's age until the next push. The ladder is
// stored as sampled; the simulator scales it by the premium's growth from the sample to the
// quote's age plus InclusionAllowance (PairCut.factor), the worst case with no push in between,
// since a push only lowers the premium. spread is the largest SpreadPbps over the pair's samples
// (the ratio falls as spread grows). A pair whose growth over the whole window would reach the
// spread is refused.
func (g *guard) refine(from, to int, hops []common.Address, spread uint64, pts []Point) ([]Point, PairCut) {
	legs := pathLegs(g.tokens, from, to, hops)
	cut := PairCut{Legs: make([]uint16, 0, len(legs)), Den: 2*pbps - min(spread, 2*pbps)}
	stale := make([]StaleLeg, len(g.tokens))
	for _, i := range legs {
		if g.legClosed(i) {
			if i == to {
				continue // the output leg's reserve is zeroed instead: the ladder is kept
			}
			return nil, PairCut{}
		}
		stale[i] = g.legs[i].stale()
		cut.Legs = append(cut.Legs, uint16(i))
	}
	if len(hops) > 0 {
		route, ok := routeIndices(g.tokens, hops)
		if !ok || !g.distinctFeeds(route) {
			return nil, PairCut{}
		}
	}
	if _, _, ok := cut.factor(stale, g.now, g.window); !ok {
		return nil, PairCut{}
	}
	return cutAbove(pts, g.rooms[to]), cut
}

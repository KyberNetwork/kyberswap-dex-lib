package btrprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// getSwapQuote omits the guards swapExactIn runs at settle: the feed gate and depeg breaker
// (PoolIOLib.pegGuardAll) and the liquid-reserve floor (PoolIOLib.settle). They are mirrored from
// state the tracker reads in the same multicall, so a pair that would revert is refused, never
// quoted. The leg word is MarkWordLib's: obs 32..63 | sigma 64..90 | ttl 107..122 | halt 134 |
// INTERNAL 232 | depeg code 234..237.

const (
	depegStepBps = 250 // PoolConstantsLib.DEPEG_STEP_BPS
	bps          = 10_000
	pbps         = 1_000_000
	// PricingLib staleness premium: STALE_Z*sigma*sqrt(excess)/BPS in PBPS, excess = seconds the
	// mark is older than the keeper grace min(ttl/2, 3 s). The base and INTERNAL legs pay none.
	staleZ        = 472
	maxStaleGrace = 3
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

// decodeMark decodes a leg word (MarkWordLib); false when the read failed or is empty.
func decodeMark(word *big.Int) (*markLeg, bool) {
	var w uint256.Int
	if word == nil || w.SetFromBig(word) || w.IsZero() {
		return nil, false
	}
	f := field(&w, 0, 32) // the lane float: mant u25 | exp7, mark = (mant << exp) >> 16
	l := &markLeg{
		obs:      field(&w, 32, 32),
		sigma:    field(&w, 64, 27),
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

// staleExcessAt is PricingLib._staleExcessOf at unix time t.
func (l *markLeg) staleExcessAt(t uint64) uint64 {
	grace := min(l.ttl/2, maxStaleGrace)
	if age := l.ageAt(t); age > grace {
		return age - grace
	}
	return 0
}

// staleTermAt is the leg's staleness premium in PBPS at unix time t (PricingLib._staleTerm).
func (l *markLeg) staleTermAt(t uint64) uint64 {
	if l.base || l.internal {
		return 0
	}
	return staleZ * l.sigma * new(uint256.Int).Sqrt(uint256.NewInt(l.staleExcessAt(t))).Uint64() / bps
}

// pegTested reports whether the depeg breaker applies to the leg.
func (l *markLeg) pegTested() bool { return l.internal || l.depegBps != 0 }

// tradableAt mirrors FeedMathLib.gate and PoolIOLib.pegGuard for a swap executed at unix time t:
// a halted, dead or TTL-expired feed reverts, and so does a depeg-coded (or INTERNAL) non-base leg
// whose mark sits outside its band.
func (l *markLeg) tradableAt(t uint64) bool {
	if l.halted || l.mark.IsZero() || l.ageAt(t) > l.ttl {
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
// asset's minLiquidity floor (PoolIOLib.settle needs liquid >= out + minLiquidity).
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

// callsPerToken is the per-token call count of the tracker's phase-1 multicall: getRiskFlags,
// balanceOf, getAsset, getLegMark, getLiquidReserves.
const callsPerToken = 5

// guard holds the state the settle-time guards read, for one refresh at block time now. A swap
// quoted from the refresh may execute until now+window (MaxAge).
type guard struct {
	tokens []common.Address
	legs   []*markLeg // nil = unreadable word
	// rooms[i] is the settle-time output room of token i (liquidRoom), nil = unknown.
	rooms       []*uint256.Int
	now, window uint64
}

// refine returns the ladder of from -> to restricted to what swapExactIn would settle and pay
// within the window, or nil when every size would revert: a leg on the path is halted, dead,
// TTL-expired by now+window or outside its depeg band, or its word is unreadable. Samples whose
// output exceeds the output leg's liquid room are cut.
//
// The pool's quote drifts with block time: each non-base leg's staleness premium grows with
// sqrt(mark age) until the next push lands. The outputs are scaled by the ratio of the path's
// premium at now+window to the one baked into the sample (both read off the legs' own words),
// the worst case with no push in between; a push only lowers the premium.
func (g *guard) refine(from, to int, hops []common.Address, pts []Point) []Point {
	legs := pathLegs(g.tokens, from, to, hops)
	var s0, s1 uint64
	for _, i := range legs {
		l := g.legs[i]
		if l == nil || !l.tradableAt(g.now+g.window) {
			return nil
		}
		s0 += l.staleTermAt(g.now)
		s1 += l.staleTermAt(g.now + g.window)
	}
	// +1 PBPS covers the pool's flooring of each fee term.
	s1++
	if s1 >= pbps {
		return nil
	}
	return haircut(cutAbove(pts, g.rooms[to]), pbps-s1, pbps-s0)
}

// haircut scales every output by num/den (rounded down) and keeps the ladder strictly increasing.
func haircut(pts []Point, num, den uint64) []Point {
	if num >= den {
		return pts
	}
	res := make([]Point, 0, len(pts))
	for _, p := range pts {
		out := new(uint256.Int).Mul(&p.Out, uint256.NewInt(num))
		out.Div(out, uint256.NewInt(den))
		if out.IsZero() || (len(res) > 0 && !out.Gt(&res[len(res)-1].Out)) {
			continue
		}
		res = append(res, Point{In: p.In, Out: *out})
	}
	return res
}

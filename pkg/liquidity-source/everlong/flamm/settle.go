package everlongflamm

import (
	"math/big"
	"slices"
)

// probeStatus is how one previewSwap answered.
type probeStatus uint8

const (
	statusPriced    probeStatus = 1 // a whole fill, or a fill cut by no more than a rounding
	statusClipped   probeStatus = 2 // the input used falls short of the ask by more than 1 part in 10,000
	statusRefused   probeStatus = 3 // a revert
	statusStarved   probeStatus = 4 // the probe ran out of its gas: it says nothing about the pool
	statusMalformed probeStatus = 5 // an answer that cannot be a fill
)

// settleRule is how a direction's answers become a ladder. ruleEverlong is the shape of the hook family that states
// it (HookFamily.Shape): an output the better of two routes, a curve's and a leverage row's, each labelled by the
// fee it reports. ruleStep takes an output to rise with its input and nothing more, so every span quotes the point
// below it: the shape of a hook of no listed family, which the shared settle vectors pin, though the tracker routes
// no such hook, as no fill of one was measured (gas.go).
type settleRule uint8

const (
	ruleEverlong settleRule = iota
	ruleStep
)

// sampled is one probe of a direction as settle reads it.
type sampled struct {
	ask, used, out, fee *big.Int
	status              probeStatus
}

// pairWords are the words of a direction's pool request (`pairRequest`) that settle reads, and on a buy the
// lever-up's top bid (word 6 of the hook's `levState` on that request). Zero: not read.
type pairWords struct {
	band, usdIn, usdOut *big.Int
	// legs: heldIn, suppliedIn, postedIn, owedIn, heldOut, suppliedOut, postedOut, owedOut
	legs [8]*big.Int
	bid  *big.Int
}

// settleIn is one direction to settle.
type settleIn struct {
	probes []sampled // ascending by ask
	words  pairWords
	rule   settleRule
	// tiers are the hook row's spreadWad and spreadDnWad, the labels a leverage fill carries; empty: no row.
	tiers         []*big.Int
	rowBand       *big.Int // the row's bandWad
	sale          bool     // token 0 is the input
	decIn, decOut uint8
	// trusted: the row passes rowOk at the live band and the pool has exactly two tokens.
	trusted  bool
	curveFee *big.Int // the curve route's label at this state; zero: not read

	// roomTest, where set, takes the room test's place, and skipAbove leaves the probe-above test out: controls for
	// the tests that show which of the two keeps a state safe. Unset, settle runs its rule.
	roomTest  func(r *room, p sampled) bool
	skipAbove bool
}

// settled is a direction's ladder: point k pays y[k] for an input of x[k], and flat[k] says the span from point k-1
// to point k quotes y[k-1] where a chord could pass what the pool pays. capped: a larger input fills the top and
// returns the rest.
type settled struct {
	x, y, f []*big.Int
	flat    []bool
	capped  bool
}

var (
	wad  = big.NewInt(1e18)
	e36  = new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	e16  = big.NewInt(1e16)
	one  = big.NewInt(1)
	dust = big.NewInt(dustOut)
)

func orZero(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}

func live(p sampled) bool { return p.status == statusPriced || p.status == statusClipped }

// settle turns a direction's probes into the published ladder, on the raw outputs. Every point is a probe the pool
// priced, and each step only drops a probe or lowers an output, so the ladder never says more than its probes. In
// order:
//
//   - Dust: once a price pays at least dustOut, every price paying less is forgotten. The pool's band test rounds a
//     whole-unit output, so a fill of n units can miss the band by 1/n and small sizes alternate between a price
//     and a refusal.
//   - Room (rule everlong, a row): a tier-labelled probe goes where its output sits within `margin` of either limit
//     of the band at its input. Within its own resolution of an edge the leverage route answers by lottery, and the
//     curve, answering on the pool's wider band, pays less at the sizes between.
//   - Above (rule everlong, a row): the points are taken once from the probes left, and a tier-labelled probe
//     stands only where one of them is a priced probe at least 1/1024 of its input above it, read or not. Where the
//     leverage route stops answering inside its span (its floor, a ceiling on its net between two sizes), it does so
//     in a band far narrower than that. A price past where those points end backs nothing, as no ray of it lowers
//     the points under it, so a tier-labelled probe that ends them, by a fall or as a third label, goes, and the
//     points then run past it.
//   - Points: from the first price, up to a refusal after a price over dust, a size the pool clipped, an output
//     that falls with one label, or a third label.
//   - Cuts: before the first tier-labelled point where the row fails its gate, the pool has another token than the
//     pair, the request did not read, a buy's top bid reads zero (the lever-down's output can jump there), or the
//     labels run tier, curve, tier; with no row, at the first point whose label differs from the first point's.
//   - Spans: across a label change, and over a ladder of one label that may be either route, a span is flat; a
//     flat span's left end is lowered to the ray of its right end where the left end is a tier's, as the curve
//     pays at least that ray below a point it priced. Every point is then lowered to the one above it.
func settle(in settleIn) settled {
	tiers := in.tiers
	isTier := func(f *big.Int) bool {
		return slices.ContainsFunc(tiers, func(t *big.Int) bool { return f.Cmp(t) == 0 })
	}
	probes := slices.Clone(in.probes)
	for k := range probes {
		probes[k].fee = orZero(probes[k].fee)
	}

	// dust
	if slices.ContainsFunc(probes, func(p sampled) bool { return live(p) && p.out.Cmp(dust) >= 0 }) {
		probes = slices.DeleteFunc(probes, func(p sampled) bool { return live(p) && p.out.Cmp(dust) < 0 })
	}

	rowed := in.rule == ruleEverlong && len(tiers) != 0
	var unread bool
	if rowed {
		// room
		r := newRoom(in)
		unread = r == nil
		if r != nil {
			passes := func(p sampled) bool { return r.passes(p.used, p.out) }
			if in.roomTest != nil {
				passes = func(p sampled) bool { return in.roomTest(r, p) }
			}
			probes = slices.DeleteFunc(probes, func(p sampled) bool { return live(p) && isTier(p.fee) && !passes(p) })
		}
		// above: a tier point stands only on a priced point of the probes left, which the lowering then reaches
		_, _, top := points(probes, isTier)
		probes = slices.DeleteFunc(probes, func(p sampled) bool {
			if in.skipAbove || !live(p) || !isTier(p.fee) {
				return false
			}
			need := new(big.Int).Rsh(p.used, aboveShift)
			return top.Cmp(need.Add(need, p.used)) < 0
		})
	}

	// points, again on the probes the drops leave
	pts, capped, _ := points(probes, isTier)

	// cuts
	n0 := len(pts)
	if in.rule == ruleEverlong && len(pts) != 0 {
		if len(tiers) == 0 {
			if k := slices.IndexFunc(pts, func(q point) bool { return q.f.Cmp(pts[0].f) != 0 }); k >= 0 {
				pts = pts[:k]
			}
		} else {
			var runs []*big.Int
			for _, q := range pts {
				if len(runs) == 0 || runs[len(runs)-1].Cmp(q.f) != 0 {
					runs = append(runs, q.f)
				}
			}
			aba := false
			for k := 0; k+2 < len(runs); k++ {
				aba = aba || runs[k].Cmp(runs[k+2]) == 0 && isTier(runs[k])
			}
			if !in.trusted || unread || !in.sale && orZero(in.words.bid).Sign() == 0 || aba {
				if k := slices.IndexFunc(pts, func(q point) bool { return isTier(q.f) }); k >= 0 {
					pts = pts[:k]
				}
			}
		}
	}
	if len(pts) != n0 {
		capped = false
	}

	// spans
	m := len(pts)
	out := settled{x: make([]*big.Int, m), y: make([]*big.Int, m), f: make([]*big.Int, m), flat: make([]bool, m),
		capped: capped}
	for k, q := range pts {
		out.x[k], out.y[k], out.f[k] = q.x, q.y, q.f
	}
	labels := distinct(pts, func(q point) *big.Int { return q.f })
	curveFee := orZero(in.curveFee)
	for k := 1; k < m; k++ {
		switch {
		case in.rule == ruleStep:
			out.flat[k] = true
		case len(tiers) == 0:
			out.flat[k] = false
		case labels == 1:
			l := out.f[0]
			out.flat[k] = isTier(l) && (curveFee.Sign() == 0 || curveFee.Cmp(l) == 0)
		default:
			out.flat[k] = out.f[k-1].Cmp(out.f[k]) != 0
		}
	}

	// lowering, from the top
	for k := m - 1; k >= 1; k-- {
		if in.rule != ruleStep && out.flat[k] && isTier(out.f[k-1]) {
			ray := new(big.Int).Mul(out.y[k], out.x[k-1])
			if ray.Quo(ray, out.x[k]); ray.Cmp(out.y[k-1]) < 0 {
				out.y[k-1] = ray
			}
		}
		if out.y[k].Cmp(out.y[k-1]) < 0 {
			out.y[k-1] = out.y[k]
		}
	}
	s := 0
	for s < m && out.y[s].Sign() == 0 {
		s++
	}
	out.x, out.y, out.f, out.flat = out.x[s:], out.y[s:], out.f[s:], out.flat[s:]
	if len(out.flat) != 0 {
		out.flat[0] = false // the first point covers no span
	}
	return out
}

// point is one point of a ladder as settle takes it: an input, its output and the label it carries.
type point struct{ x, y, f *big.Int }

// points takes a direction's points from its probes in order: from the first price, up to a refusal after a price
// over dust, a size the pool clipped, an output that falls with one label (a fall from a tier-labelled point to
// another label is a change of route), or a third label. A refusal before any price over dust starts them again.
// capped: a curve-labelled clip ends them, so a larger input fills there and returns the rest. top is the largest
// input of a priced probe they take, zero where they take none.
func points(probes []sampled, isTier func(*big.Int) bool) (pts []point, capped bool, top *big.Int) {
	top = new(big.Int)
	for _, p := range probes {
		if p.status == statusRefused && (len(pts) == 0 || pts[len(pts)-1].y.Cmp(dust) < 0) {
			pts, capped, top = pts[:0], false, new(big.Int)
			continue
		}
		if !live(p) {
			break
		}
		if len(pts) != 0 {
			l := pts[len(pts)-1]
			if p.used.Cmp(l.x) <= 0 {
				capped = p.used.Cmp(l.x) == 0 && p.fee.Cmp(l.f) == 0 && !isTier(l.f)
				break
			}
			// a fall ends the points, but a drop: a fall from a tier point to another label
			if p.out.Cmp(l.y) < 0 && (!isTier(l.f) || p.fee.Cmp(l.f) == 0) {
				break
			}
			if !slices.ContainsFunc(pts, func(q point) bool { return q.f.Cmp(p.fee) == 0 }) && distinct(pts,
				func(q point) *big.Int { return q.f }) >= 2 {
				break
			}
		}
		pts = append(pts, point{new(big.Int).Set(p.used), new(big.Int).Set(p.out), p.fee})
		if p.status == statusPriced && p.used.Cmp(top) > 0 {
			top = p.used
		}
		if p.status == statusClipped {
			capped = !isTier(p.fee)
			break
		}
	}
	return pts, capped, top
}

func distinct[T any](v []T, key func(T) *big.Int) int {
	var seen []*big.Int
	for _, e := range v {
		if k := key(e); !slices.ContainsFunc(seen, func(s *big.Int) bool { return s.Cmp(k) == 0 }) {
			seen = append(seen, k)
		}
	}
	return len(seen)
}

// room is the band test a tier-labelled point must pass with `margin` to spare on both sides, in output units.
type room struct {
	sale          bool
	p36, b        *big.Int // token 0's price in token 1 base units per token 0 base unit, 1e36; the band
	need, v, n    *big.Int
	decIn, decOut uint8
}

// newRoom reads the request's words for the room test; nil: they read as not read.
func newRoom(in settleIn) *room {
	w := in.words
	usdIn, usdOut, band := orZero(w.usdIn), orZero(w.usdOut), orZero(w.band)
	usd0, usd1, d0, d1 := usdIn, usdOut, in.decIn, in.decOut
	if !in.sale {
		usd0, usd1, d0, d1 = usdOut, usdIn, in.decOut, in.decIn
	}
	leg := func(k int) *big.Int { return orZero(w.legs[k]) }
	r := &room{sale: in.sale, p36: new(big.Int), decIn: in.decIn, decOut: in.decOut}
	if usd1.Sign() != 0 {
		num := new(big.Int).Mul(usd0, pow10(int(d1)))
		num.Mul(num, e36)
		r.p36.Div(num, new(big.Int).Mul(usd1, pow10(int(d0))))
	}
	r.v = new(big.Int).Add(leg(4), leg(5))
	r.v.Add(r.v, leg(6))
	if band.Sign() == 0 || usdIn.Sign() == 0 || usdOut.Sign() == 0 || r.p36.Sign() == 0 || !in.sale && r.v.Sign() == 0 {
		return nil
	}
	r.b = minBig(orZero(in.rowBand), band)
	u := big.NewInt(1)
	if in.sale {
		u.Quo(r.p36, e36).Add(u, one)
	}
	r.need = new(big.Int).Mul(u, big.NewInt(3))
	r.need.Add(r.need, big.NewInt(2))
	idle := new(big.Int).Add(leg(0), leg(1))
	idle.Sub(idle, leg(3))
	if idle.Sign() < 0 {
		idle.SetInt64(0)
	}
	r.n = idle.Mul(idle, e36).Div(idle, r.p36)
	return r
}

// passes reports whether a fill of `used` paying `out` sits `margin` inside both limits of the band.
func (r *room) passes(used, out *big.Int) bool {
	lo, hi := r.limits(used)
	m := r.margin(used, out)
	return out.Cmp(new(big.Int).Add(lo, m)) >= 0 && new(big.Int).Add(out, m).Cmp(hi) <= 0
}

// limits are the least and the most output the band allows at `used`: (1 -/+ b) times its value at the feed.
func (r *room) limits(used *big.Int) (lo, hi *big.Int) {
	down, up := new(big.Int).Sub(wad, r.b), new(big.Int).Add(wad, r.b)
	value, scale := new(big.Int).Mul(used, r.p36), new(big.Int).Mul(e36, wad)
	if !r.sale {
		value.Mul(used, e36)
		scale.Mul(r.p36, wad)
	}
	lo = new(big.Int).Mul(value, down)
	lo.Div(lo, scale).Add(lo, one)
	hi = new(big.Int).Mul(value, up)
	return lo, hi.Div(hi, scale)
}

// margin is the room a tier-labelled point needs on each side: the leverage route's own floors (`need`, three base
// units of the pool asset and two units), the feed's rounding bracket, which grows with the size on a sale and
// with the output on a buy, on a buy the lever-down's resolution in the book, and a unit for the floors.
func (r *room) margin(used, out *big.Int) *big.Int {
	m := new(big.Int).Set(r.need)
	if r.sale {
		bracket := new(big.Int).Mul(used, pow10(int(r.decOut)))
		m.Add(m, bracket.Div(bracket, e36))
		return m.Add(m, one)
	}
	book := new(big.Int).Add(r.v, r.n)
	m.Add(m, book.Div(book, e16))
	bracket := new(big.Int).Mul(out, pow10(int(r.decIn)))
	m.Add(m, bracket.Div(bracket, r.p36))
	return m.Add(m, one)
}

// hookRow is a hook's leverage row, words 27-34 of its `params()`: fixed at construction, so read once per hook.
type hookRow struct {
	spread, spreadDn, upCap, dnCap, ceiling, band, feedGate, upFloor *big.Int
}

// tiers are the labels a leverage fill carries; none where the row is zero, a hook with no leverage.
func (r hookRow) tiers() []*big.Int {
	if r.spread == nil || r.spread.Sign() == 0 {
		return nil
	}
	return []*big.Int{r.spread, orZero(r.spreadDn)}
}

// rowOk is the gate a hook's row passes before its tier-labelled points may stand: a band, both caps on, a lower
// spread of at least minSpreadWad, and each cap inside the band by capMarginWad, at the tighter of the row's band and
// the pool's live one. A row that fails it is routed on the curve's points only.
func rowOk(r hookRow, liveBand *big.Int) bool {
	b := minBig(orZero(r.band), orZero(liveBand))
	if b.Sign() == 0 || orZero(r.dnCap).Sign() == 0 || orZero(r.upCap).Sign() == 0 ||
		orZero(r.spreadDn).Cmp(big.NewInt(minSpreadWad)) < 0 {
		return false
	}
	for _, c := range []*big.Int{r.dnCap, r.upCap} {
		edge := new(big.Int).Mul(c, new(big.Int).Add(wad, b))
		edge.Quo(edge, wad).Add(edge, big.NewInt(capMarginWad))
		if edge.Cmp(b) > 0 {
			return false
		}
	}
	return true
}

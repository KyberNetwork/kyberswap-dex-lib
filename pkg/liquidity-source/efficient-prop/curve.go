package efficientprop

import (
	"errors"
	"sort"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// ErrBelowFirstPoint: the amount is under the smallest size the listing filled at the refresh, and the listing may
// refuse any less (its minimum fill).
var ErrBelowFirstPoint = errors.New("efficient-prop: amount below the smallest sampled size")

// ErrCreditStep: the amount is between a credit listing's last size filled inside its pool's custody and its first
// routed size, where neither point bounds the fill (Credit).
var ErrCreditStep = errors.New("efficient-prop: amount between the custody and the credit route")

// curve is one direction's ladder as the simulator walks it: point k is what the door's quote paid, y[k], for an
// input of x[k] at the refresh block.
//
// The front fills a listing by walking its levels from the side's fill memory, each level's price no better than
// the one before (a depth post with a better later level is refused), and floors the output once, so the output is
// a floored concave function of the input, and a chord between two sampled points never passes it. The simulator
// still checks the points: a span whose neighbours are not consistent with a concave output (each span's slope at
// most the one before it, up to the floors of the outputs) quotes the point below it, which only ever lowers the
// quote. Under the first point it quotes nothing (the listing's minimum fill), and past the last nothing (a fill is
// all or nothing, and the door refused the next size sampled or none was sampled).
//
// On a listing with a credit mode, points from `from` on are sizes the front routes to its pool's own swap, and those
// before it sizes it fills inside the custody: the span between the two (span from, where 0 < from < m) quotes
// nothing, since the route's output need not continue the custody's. Its slope still judges the spans beside it,
// which only turns their chords off.
type curve struct {
	x, y []uint256.Int
	// chord[k] is span k's verdict, from point k-1 to point k; chord[0], under the first point, is never on.
	chord []bool
	// from is the index of the first routed point; m where none routes
	from int
}

// newCurve is the curve of points, the first routed one at from (len(points) where none routes).
func newCurve(points []Point, from int) (curve, error) {
	m := len(points)
	if m > maxLadderPoints || from < 0 || from > m {
		return curve{}, ErrInvalidState
	}
	c := curve{x: make([]uint256.Int, m), y: make([]uint256.Int, m), chord: make([]bool, m), from: from}
	for k, p := range points {
		if p[0] == nil || p[1] == nil || p[0].IsZero() || p[1].IsZero() ||
			p[0].BitLen() > maxPointBits || p[1].BitLen() > maxPointBits {
			return curve{}, ErrInvalidState
		}
		if k > 0 && (!p[0].Gt(&c.x[k-1]) || p[1].Lt(&c.y[k-1])) {
			return curve{}, ErrInvalidState
		}
		c.x[k].Set(p[0])
		c.y[k].Set(p[1])
	}
	for k := 1; k < m; k++ {
		// span 1 is judged against the slope from the origin too: a start a fixed cost holds down is no concave one
		c.chord[k] = !c.steeper(k, k-1) && (k == m-1 || !c.steeper(k+1, k))
	}
	return c, nil
}

// steeper reports whether span a, the later, is steeper than span b by more than the floors of the outputs explain:
// each sampled output is floored, so of a concave output the slopes may rise by up to 1/dx_a + 1/dx_b. That is
// (dy_a - 1) * dx_b > (dy_b + 1) * dx_a. Every amount is below 2^maxPointBits, so neither product leaves 256 bits.
func (c *curve) steeper(a, b int) bool {
	var dxa, dya, dxb, dyb, one uint256.Int
	c.segment(a, &dxa, &dya)
	c.segment(b, &dxb, &dyb)
	if dya.IsZero() {
		return false
	}
	one.SetOne()
	dya.Sub(&dya, &one)
	dyb.Add(&dyb, &one)
	return dya.Mul(&dya, &dxb).Gt(dyb.Mul(&dyb, &dxa))
}

// segment is span k's run and rise: from the origin for k = 0.
func (c *curve) segment(k int, dx, dy *uint256.Int) {
	dx.Set(&c.x[k])
	dy.Set(&c.y[k])
	if k > 0 {
		dx.Sub(dx, &c.x[k-1])
		dy.Sub(dy, &c.y[k-1])
	}
}

// quote sets out to what a fill of amount pays: exact at a sampled point, the floored chord or the point below
// between two, nothing outside the sampled range nor across a credit listing's step.
func (c *curve) quote(amount, out *uint256.Int) error {
	m := len(c.x)
	if m == 0 {
		return ladder.ErrNoQuote
	}
	if amount.Gt(&c.x[m-1]) {
		return ladder.ErrAmountInTooLarge
	}
	k := sort.Search(m, func(i int) bool { return !c.x[i].Lt(amount) })
	if c.x[k].Eq(amount) {
		out.Set(&c.y[k])
		return nil
	}
	if k == 0 {
		return ErrBelowFirstPoint
	}
	if k == c.from {
		return ErrCreditStep
	}
	x0, y0 := &c.x[k-1], &c.y[k-1]
	if !c.chord[k] {
		out.Set(y0)
		return nil
	}
	var dx, dy, t uint256.Int
	dx.Sub(&c.x[k], x0)
	dy.Sub(&c.y[k], y0)
	t.Sub(amount, x0)
	t.Mul(&t, &dy) // both below 2^maxPointBits
	t.Div(&t, &dx)
	out.Add(&t, y0)
	return nil
}

// quoteAfter sets out to no more than a fill of amount pays once fills of the route took used of the direction's input
// off its rungs (each fill's input rounded up to the board unit, as the front's fill memory counts it). The front
// pays that fill F(used + amount) - F(used), floored, F being the scaled output of the walk the ladder sampled:
// concave, never falling, zero at zero, with y[k] <= F(x[k]) < y[k] + 1 in a block that answered the ladder itself.
// The quote is the ladder's own answer at used + amount, which F is never under, less a bound F(used) is never over.
// With i the first point at or past used:
//
//   - from the right: F(used) <= F(x[i]) - (x[i] - used) * slope, the slope that of a chord from point i to a later
//     point, which a concave F is never under before point i (slopeAfter). Where the fill ends before point i, it
//     pays at least that slope throughout.
//   - from the left (own, every block's samples the ladder itself): F(used) <= F(x[i-1]) + (used - x[i-1]) * slope,
//     the slope that of a chord from an earlier point, or from nothing, to point i-1, which a concave F is never
//     over after it (slopeBefore), and F(used) <= F(x[i]).
//
// The floors cost each bound a unit or two, taken against the taker. On a ladder that keeps the least of blocks that
// answered differently (own false) only the first holds, and only where the tracker found the ladder steady (Repeat):
// every block pays at least the ladder's rise between any two points, less a unit, which the bound gives up as well.
// A fill is quoted only at or above the ladder's first point on both legs, since the listing's minimum fill binds
// each fill, and only where used + amount is inside the ladder: a size the door filled whole at the refresh.
func (c *curve) quoteAfter(used, amount *uint256.Int, own bool, out *uint256.Int) error {
	m := len(c.x)
	if m == 0 || c.from != m {
		return ErrSecondFill
	}
	if amount.Lt(&c.x[0]) {
		return ErrBelowFirstPoint
	}
	var z, yz uint256.Int
	if _, overflow := z.AddOverflow(used, amount); overflow {
		return ladder.ErrAmountInTooLarge
	}
	if err := c.quote(&z, &yz); err != nil {
		return err
	}
	// one is: used < used + amount <= x[m-1]
	i := sort.Search(m, func(k int) bool { return !c.x[k].Lt(used) })
	var rise, run, t, bound uint256.Int
	out.Clear()

	// from the right
	c.slopeAfter(i, &rise, &run)
	if !c.x[i].Gt(&z) {
		t.Sub(&c.x[i], used)
		t.Mul(&t, &rise) // both below 2^maxPointBits
		t.Div(&t, &run)
		t.Add(&t, &yz)
		bound.AddUint64(&c.y[i], 2)
		if t.Gt(&bound) {
			out.Sub(&t, &bound)
		}
	} else {
		t.Mul(amount, &rise)
		out.Div(&t, &run)
	}

	// from the left, where every block sampled answered the ladder itself
	if own {
		bound.AddUint64(&c.y[i], 1)
		if i > 0 {
			c.slopeBefore(i-1, &rise, &run)
			t.Sub(used, &c.x[i-1])
			t.Mul(&t, &rise)
			t.Add(&t, &run)
			t.SubUint64(&t, 1)
			t.Div(&t, &run) // rounded up
			t.Add(&t, &c.y[i-1])
			t.AddUint64(&t, 1)
			if t.Lt(&bound) {
				bound.Set(&t)
			}
		}
		if yz.Gt(&bound) {
			if t.Sub(&yz, &bound); t.Gt(out) {
				out.Set(&t)
			}
		}
	}
	if out.Lt(&c.y[0]) {
		return ErrSecondFill
	}
	return nil
}

// slopeAfter sets rise/run to the steepest slope the ladder proves F keeps before point i: over the later points j,
// the most of (y[j] - y[i] - 2) / (x[j] - x[i]), the chord's slope less what the floors of its two ends and a steady
// ladder's slack may hide; 0/1 where no later point proves any. A short chord's floors hide most of its slope, so
// every later point is tried.
func (c *curve) slopeAfter(i int, rise, run *uint256.Int) {
	rise.Clear()
	run.SetOne()
	var dx, dy, l, r uint256.Int
	for j := i + 1; j < len(c.x); j++ {
		dy.Sub(&c.y[j], &c.y[i])
		if dy.CmpUint64(2) <= 0 {
			continue
		}
		dy.SubUint64(&dy, 2)
		dx.Sub(&c.x[j], &c.x[i])
		if l.Mul(&dy, run).Gt(r.Mul(rise, &dx)) { // every factor below 2^maxPointBits
			rise.Set(&dy)
			run.Set(&dx)
		}
	}
}

// slopeBefore sets rise/run to the flattest slope the ladder proves F stays under after point k: over the earlier
// points j, and nothing (0 in, 0 out), the least of (y[k] - y[j] + 1) / (x[k] - x[j]), the chord's slope plus what
// the floors of its two ends may hide.
func (c *curve) slopeBefore(k int, rise, run *uint256.Int) {
	rise.AddUint64(&c.y[k], 1)
	run.Set(&c.x[k])
	var dx, dy, l, r uint256.Int
	for j := range k {
		dy.Sub(&c.y[k], &c.y[j])
		dy.AddUint64(&dy, 1)
		dx.Sub(&c.x[k], &c.x[j])
		if l.Mul(&dy, run).Lt(r.Mul(rise, &dx)) { // every factor at most 2^maxPointBits
			rise.Set(&dy)
			run.Set(&dx)
		}
	}
}

// routes is whether a fill of amount, one quote answers, is a routed one: at or past the first routed point.
func (c *curve) routes(amount *uint256.Int) bool {
	return c.from < len(c.x) && !amount.Lt(&c.x[c.from])
}

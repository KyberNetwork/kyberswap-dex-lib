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

// routes is whether a fill of amount, one quote answers, is a routed one: at or past the first routed point.
func (c *curve) routes(amount *uint256.Int) bool {
	return c.from < len(c.x) && !amount.Lt(&c.x[c.from])
}

package everlongflamm

import (
	"math"
	"slices"
	"sort"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// curve is one direction's ladder as the simulator walks it: point k pays y[k] for an input of x[k], what the pool
// paid there at the refresh block unless the tracker lowered it (settle).
//
// Between two points the pool's output is unknown. A span the tracker marked flat (Ladder.Flat) quotes the point
// below it: there the ladder changes route, or its one label may be either route, and the tracker has lowered that
// point to what the route above pays at least. A span whose neighbours are not consistent with an output concave in
// the input (each span's slope at most the one before it, but for the floors of the outputs: a fixed cost, a size
// tier, a branch that pays more past a size) quotes the point below it too, which only ever lowers the quote. Every
// other span quotes the chord less the haircut, never less than the point below it. Under the first point it quotes
// nothing: that point is the smallest size priced, and the pool may refuse any less, as the floor of a dust output
// falls past its price band.
type curve struct {
	x, y []uint256.Int
	// chord[k] is span k's verdict, from point k-1 to point k; chord[0], under the first point, is never on.
	chord []bool
	// flat: a span is flat (Ladder.Flat), so a point may be lowered and no longer say what the pool paid there: the
	// direction takes no second fill.
	flat   bool
	capped bool

	// The charge (gas.go): the ladder's cells, their least, the second-fill charge (never above the least), and the
	// reads that narrow the cells. heldRead and freedRead: held, the output's custody, and freedTop, what the pool
	// frees before a draw with its slack, were read. after are the cells' paths at their gas after another pool of
	// the source filled, each at most its cell, and leastAfter their least: the cells themselves where the ladder
	// carries none.
	cells, after                []Cell
	least, second, leastAfter   int64
	held, freedTop              uint256.Int
	heldRead, freedRead         bool
	owedOut, owedIn             uint8
	firstOwing, postedIn, repay bool
}

func newCurve(l Ladder) (curve, error) {
	m := len(l.Points)
	if m > maxLadderPoints {
		return curve{}, ErrInvalidState
	}
	c := curve{x: make([]uint256.Int, m), y: make([]uint256.Int, m), chord: make([]bool, m), capped: l.Capped,
		cells: l.Cells, least: math.MaxInt64, owedOut: l.OwedOut, owedIn: l.OwedIn, firstOwing: l.FirstOwing,
		postedIn: l.PostedIn, repay: l.Repay}
	// a ladder without cells or a second-fill charge (one saved before either, or of a kind with no measured fill) is
	// not quoted
	if len(l.Cells) == 0 || l.Second <= 0 || l.OwedOut > owedSome || l.OwedIn > owedSome {
		return curve{}, ErrInvalidState
	}
	for _, cell := range l.Cells {
		if cell.Gas <= 0 {
			return curve{}, ErrInvalidState
		}
		c.least = min(c.least, cell.Gas)
	}
	c.second = min(l.Second, c.least)
	c.after, c.leastAfter = c.cells, c.least
	if len(l.After) != 0 {
		c.after = slices.Clone(l.Cells)
		for _, cell := range l.After {
			if cell.Gas <= 0 {
				return curve{}, ErrInvalidState
			}
			if at := slices.IndexFunc(c.after, func(e Cell) bool { return e.Path == cell.Path }); at >= 0 {
				c.after[at].Gas = min(c.after[at].Gas, cell.Gas)
			}
		}
		for _, cell := range c.after {
			c.leastAfter = min(c.leastAfter, cell.Gas)
		}
	}
	if l.Held != nil {
		c.held.Set(l.Held)
		c.heldRead = true
	}
	if l.Freed != nil {
		var slack, d uint256.Int
		slack.Div(l.Freed, d.SetUint64(freedSlack))
		if _, overflow := c.freedTop.AddOverflow(l.Freed, &slack); overflow {
			c.freedTop.SetAllOne()
		}
		c.freedRead = true
	}
	for k, p := range l.Points {
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
		if l.Flat>>k&1 == 1 {
			c.flat = true
			continue
		}
		// the slope from the origin still judges span 1: a start that a fixed cost holds down is no concave one
		c.chord[k] = !c.steeper(k, k-1) && (k == m-1 || !c.steeper(k+1, k))
	}
	return c, nil
}

// steeper reports whether span a, the later, is steeper than span b by more than the floors of the outputs
// explain: each probe's output is the pool's floored, so of a concave output the slopes may rise by up to
// 1/dx_a + 1/dx_b. That is (dy_a - 1) * dx_b > (dy_b + 1) * dx_a. Every amount is below 2^maxPointBits, so
// neither product leaves 256 bits.
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

func (c *curve) segment(k int, dx, dy *uint256.Int) {
	dx.Set(&c.x[k])
	dy.Set(&c.y[k])
	if k > 0 {
		dx.Sub(dx, &c.x[k-1])
		dy.Sub(dy, &c.y[k-1])
	}
}

// quote returns the input a fill of `amount`, counted from the direction's start, uses and the output it pays, and
// whether the pool's input ceiling cut it.
func (c *curve) quote(amount *uint256.Int, haircutBps uint64, used, out *uint256.Int) (bool, error) {
	m := len(c.x)
	if m == 0 {
		return false, ladder.ErrNoQuote
	}
	if amount.Gt(&c.x[m-1]) {
		if !c.capped {
			return false, ladder.ErrAmountInTooLarge
		}
		used.Set(&c.x[m-1])
		out.Set(&c.y[m-1])
		return true, nil
	}
	k := sort.Search(m, func(i int) bool { return !c.x[i].Lt(amount) })
	used.Set(amount)
	if c.x[k].Eq(amount) {
		out.Set(&c.y[k])
		return false, nil
	}
	if k == 0 {
		return false, ladder.ErrNoQuote
	}
	x0, y0 := &c.x[k-1], &c.y[k-1]
	if !c.chord[k] {
		out.Set(y0)
		return false, nil
	}
	var dx, dy, t uint256.Int
	dx.Sub(&c.x[k], x0)
	dy.Sub(&c.y[k], y0)
	t.Sub(amount, x0)
	t.Mul(&t, &dy) // both below 2^maxPointBits
	t.Div(&t, &dx)
	t.Add(&t, y0)
	if haircutBps != 0 {
		var h uint256.Int
		t.Mul(&t, h.SetUint64(10_000-haircutBps)) // below 2^135
		t.Div(&t, h.SetUint64(10_000))
	}
	if t.Lt(y0) {
		t.Set(y0)
	}
	out.Set(&t)
	return false, nil
}

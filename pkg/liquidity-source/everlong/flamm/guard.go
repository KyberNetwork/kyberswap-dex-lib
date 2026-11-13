package everlongflamm

import (
	"math/big"
)

// A venue that holds a minimum position (CAP_HELD_REPAY, Lista Moolah's) is passed over by the Router's plan where a
// draw would leave its debt under that minimum, or within releaseSlack units over it while it owes, and the pool's
// quote then refuses (FLAMMPairQuoteLib). With F what the pool frees of the output before a draw (its held, supplied
// and posted output), a payout past custody and under E = F + minimum - owed (plus releaseSlack where it owes) can
// refuse, while a payout within custody, or at or over E, passes the minimum. Neither a chord nor a flat span knows
// where that window falls between two probed sizes, so a direction whose output such a venue lends keeps one piece of
// its ladder: the sizes paid from custody, or those paying at least E (keepHeld).
const (
	// capHeldRepay is the capabilities() bit of a venue's account that holds a minimum position
	// (IFinancingAccount.CAP_HELD_REPAY): its minimum is IHeldRepay.minimumDebt.
	capHeldRepay = 512
	// releaseSlack is how far over its minimum an owing venue's debt must stay (MMRouterLib.RELEASE_SLACK).
	releaseSlack = 4
)

var (
	// minPartsBandWad is the least band the protocol's part is bounded by (FLAMMStore.MIN_FEE_BAND_WAD, 1 %).
	minPartsBandWad = big.NewInt(1e16)
	e19             = new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)
)

// heldRepay is what a refresh read of the live venue holding a minimum position that lends a token: its minimum
// (minimumDebt), nil where that read failed or answered not ok, as the venue then lends nothing; and whether it is the
// only live venue lending the token.
type heldRepay struct {
	floor *big.Int
	sole  bool
}

// withMargin is a minimum with the margin a fill landing after the read needs, as the venue's minimum moves with its
// oracle: 1 % over, rounded up.
func withMargin(floor *big.Int) *big.Int {
	m := new(big.Int).Add(floor, big.NewInt(99))
	m.Quo(m, big.NewInt(100))
	return m.Add(m, floor)
}

// heldParts bounds what the protocol's part and the operator's share add to a payout of y labelled f in a band of
// `band`: the larger of the label's fee, the band's bound at the highest spread rate (three tenths of the band, at
// least 1 %) and the notional floor (a thousandth of y), each rounded up, plus one. False where the label is no fee.
func heldParts(y, f, band *big.Int) (*big.Int, bool) {
	if f.Cmp(wad) >= 0 {
		return nil, false
	}
	fee := ceilDiv(new(big.Int).Mul(y, f), new(big.Int).Sub(wad, f))
	b := band
	if b.Cmp(minPartsBandWad) < 0 {
		b = minPartsBandWad
	}
	bound := ceilDiv(new(big.Int).Mul(new(big.Int).Mul(y, b), big.NewInt(3)), e19)
	floor := ceilDiv(y, big.NewInt(1_000))
	parts := fee
	if bound.Cmp(parts) > 0 {
		parts = bound
	}
	if floor.Cmp(parts) > 0 {
		parts = floor
	}
	return parts.Add(parts, one), true
}

func ceilDiv(a, b *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	if r.Sign() != 0 {
		q.Add(q, one)
	}
	return q
}

// keepHeld is the range of a settled ladder's points (x, y: the pool's outputs, each at most what it paid there) that
// a direction whose output g's venue lends may publish: all of them, none, or one piece. legs are the pool's
// positions as its request read them at the refresh block, the direction's input first, nil where no read of it
// answered (its band word zero), and band that read's band word; probes give each point's label (the largest fee
// priced or clipped at or under its input) and the pool's own answer at its input.
//
// Where the legs did not read nothing is published. The low piece is the points whose
// payout and parts fit the output's custody, with the spans between them; the high piece runs from the first point
// paying at least E, with nothing quoted under it. Where the minimum did not read, or another live venue lends the
// output too, the low piece alone is kept, as a draw split between venues can leave the minimum's slice under it.
// Where E is not past F every draw passes the minimum, and the ladder stands. Otherwise the piece whose top pays more
// is kept: the high one where it has a point, as every point of it pays past custody.
func keepHeld(x, y []*big.Int, probes []sampled, legs *[8]*big.Int, band *big.Int, g heldRepay) (lo, hi int) {
	if legs == nil {
		return 0, 0
	}
	held := legs[4]
	freed := new(big.Int).Add(held, legs[5])
	freed.Add(freed, legs[6])
	low := 0
	for k := range y {
		label, paid := new(big.Int), y[k]
		for _, p := range probes {
			if p.used == nil || p.used.Cmp(x[k]) > 0 {
				continue
			}
			if (p.status == statusPriced || p.status == statusClipped) && p.fee != nil && p.fee.Cmp(label) > 0 {
				label = p.fee
			}
			if p.used.Cmp(x[k]) == 0 && p.out != nil && p.out.Cmp(paid) > 0 {
				paid = p.out
			}
		}
		parts, ok := heldParts(paid, label, band)
		if !ok || parts.Add(parts, paid).Cmp(held) > 0 {
			break
		}
		low++
	}
	if g.floor == nil || !g.sole || low == len(y) {
		return 0, low
	}
	owed := legs[7]
	edge := new(big.Int).Add(freed, withMargin(g.floor))
	if owed.Sign() > 0 {
		edge.Add(edge, big.NewInt(releaseSlack))
	}
	edge.Sub(edge, owed)
	if edge.Cmp(freed) <= 0 {
		return 0, len(y)
	}
	for k := range y {
		if y[k].Cmp(edge) >= 0 {
			return k, len(y)
		}
	}
	return 0, low
}

// cutHeld keeps points [lo, hi) of a settled ladder: a high piece's first point quotes nothing under it, and a ladder
// cut under its top no longer ends at the pool's input ceiling.
func cutHeld(s settled, lo, hi int) settled {
	if lo == 0 && hi == len(s.x) {
		return s
	}
	out := settled{x: s.x[lo:hi], y: s.y[lo:hi], flat: append([]bool(nil), s.flat[lo:hi]...),
		capped: s.capped && hi == len(s.x)}
	if len(s.f) == len(s.x) {
		out.f = s.f[lo:hi]
	}
	if len(out.flat) != 0 {
		out.flat[0] = false
	}
	return out
}

package btrprop

import "github.com/holiman/uint256"

// BTR pool v2 prices a leg's coverage c = reserves/liabilities with a barrier law (dex-evm
// CovLib). A swap whose input or output exceeds the leg's cap reverts
// ThresholdViolation(amount, room): the size is refused, never clamped. coverageRoom ports
// CovLib._cap so the tracker sizes the ladder inside both caps.

var (
	covWad = uint256.NewInt(1e18)
	covBps = uint256.NewInt(10_000)
)

// kappaBps decodes a 6-bit kappa code (CovLib.kappaBps).
func kappaBps(code uint64) uint64 {
	if code>>2 == 0 {
		return code
	}
	return (4 + code&3) << ((code >> 2) - 1)
}

// coverageRoom is CovLib.cap: the most a leg can pay out (neg) or take in, given its covBits,
// kneeBits (zero unless the leg carries KNEE_BIT), reserves r and liabilities l. kNeg is the
// outflow kappa in bps. A leg with no liabilities pays its whole reserve and takes nothing.
func coverageRoom(bits uint32, knee uint16, neg bool, kNeg uint64, r, l *uint256.Int) *uint256.Int {
	if l.IsZero() {
		if neg {
			return new(uint256.Int).Set(r)
		}
		return new(uint256.Int)
	}
	u := func(v uint64) *uint256.Int { return uint256.NewInt(v) }
	t := u(uint64((bits>>6)&63)*5e14 + 1e18)
	var pole, span *uint256.Int
	var kb, kb2, bk uint64
	if neg {
		pole = u(uint64(bits&63) * 2e16)
		span = new(uint256.Int).Sub(t, pole)
		kb = kNeg
		if kb2 = kappaBps(uint64(knee >> 6)); kb2 != 0 {
			bk = uint64(knee&63)*2e16 - uint64(bits&63)*2e16
		}
	} else {
		pole = u(105e16 + uint64((bits>>15)&31)*5e16)
		span = new(uint256.Int).Sub(pole, t)
		span.Sub(span, u(uint64((bits>>12)&7)*5e16))
		kb = kappaBps(uint64(bits >> 26))
	}

	// y* = ceil(sqrt(q)): the distance from the pole where the marginal toll reaches 1.
	span2 := new(uint256.Int).Mul(span, span)
	q := divUp(new(uint256.Int).Mul(u(kb), span2), u(10_000+kb))
	if kb2 != 0 && bk > 0 {
		b2 := new(uint256.Int).Mul(u(bk-1), u(bk-1))
		if !q.Gt(b2) {
			num := new(uint256.Int).Mul(u(kb), span2)
			num.Add(num, new(uint256.Int).Mul(u(kb2), new(uint256.Int).Mul(u(bk), u(bk))))
			q = divUp(num, u(10_000+kb+kb2))
		}
	}
	y := new(uint256.Int).Sqrt(q)
	if sq := new(uint256.Int).Mul(y, y); sq.Lt(q) {
		y.AddUint64(y, 1)
	}

	// l < 2^128 and pole < 2^62: the products fit a word.
	if neg {
		floorR := divUp(new(uint256.Int).Mul(l, new(uint256.Int).Add(pole, y)), covWad)
		if r.Gt(floorR) {
			return floorR.Sub(r, floorR)
		}
		return new(uint256.Int)
	}
	capR := new(uint256.Int).Mul(l, pole.Sub(pole, y))
	capR.Div(capR.SubUint64(capR, 1), covWad)
	if capR.Gt(r) {
		return capR.Sub(capR, r)
	}
	return new(uint256.Int)
}

// divUp is ceil(a/b), b > 0.
func divUp(a, b *uint256.Int) *uint256.Int {
	res, rem := new(uint256.Int), new(uint256.Int)
	res.DivMod(a, b, rem)
	if !rem.IsZero() {
		res.AddUint64(res, 1)
	}
	return res
}

// legRoom is the cap of one asset as read from getAsset, nil when the book is unreadable. The
// outflow kappa is the leg's own: a class row can override it per pair family (not read here),
// so the outflow room is exact only for the pairs that keep the leg's kappa; a tighter row
// still ends the ladder at its first revert.
func legRoom(a *rpcAsset, neg bool) *uint256.Int {
	if a == nil || a.Reserves == nil || a.Liabilities == nil {
		return nil
	}
	var r, l uint256.Int
	if r.SetFromBig(a.Reserves) || l.SetFromBig(a.Liabilities) {
		return nil
	}
	var knee uint16
	if a.Flags&KneeBit != 0 {
		knee = a.KneeBits
	}
	return coverageRoom(a.CovBits, knee, neg, kappaBps(uint64((a.CovBits>>20)&63)), &r, &l)
}

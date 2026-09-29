package everlongflamm

import "github.com/holiman/uint256"

// Shared integer helpers. Every division floors unless the name says otherwise, matching
// Solidity's `/` and OpenZeppelin Math.mulDiv; the *Ceil variants match Math.Rounding.Ceil.
// Module files must not redefine these; module-private helpers carry the module prefix
// (alm, fee, hook, gate, mm, lev, swap).

// mulDiv returns floor(x*y/d) over 512-bit intermediates; overflow reports true when the
// quotient does not fit 256 bits (Math.mulDiv reverts there).
func mulDiv(x, y, d *uint256.Int) (*uint256.Int, bool) {
	var z uint256.Int
	_, overflow := z.MulDivOverflow(x, y, d)
	return &z, overflow
}

// mulDivCeil returns ceil(x*y/d); overflow reports true when the rounded quotient does not
// fit 256 bits.
func mulDivCeil(x, y, d *uint256.Int) (*uint256.Int, bool) {
	z, overflow := mulDiv(x, y, d)
	if overflow {
		return z, true
	}
	var rem uint256.Int
	if !rem.MulMod(x, y, d).IsZero() {
		if z.Eq(maxUint256) {
			return z, true
		}
		z.AddUint64(z, 1)
	}
	return z, false
}

// divCeil returns ceil(x/y) for y != 0.
func divCeil(x, y *uint256.Int) *uint256.Int {
	var q, r uint256.Int
	q.DivMod(x, y, &r)
	if !r.IsZero() {
		q.AddUint64(&q, 1)
	}
	return &q
}

// minU returns the smaller of a and b (b on a tie). The result IS one of the arguments: read it or copy it,
// never write through it.
func minU(a, b *uint256.Int) *uint256.Int {
	if a.Lt(b) {
		return a
	}
	return b
}

var maxUint256 = new(uint256.Int).SetAllOne()

// FNV-1a 64: the mix behind the simulator's lineage token (pool_simulator.go lineageOf). A hash and not a counter
// so that two simulators built from the same entity, and two that adopted the same fills, agree on it without
// sharing any state; FNV because the token identifies a state to this process, never to a caller that could choose
// a collision.
const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

// fnvWord folds one 64-bit word, low byte first.
func fnvWord(h, w uint64) uint64 {
	for i := 0; i < 8; i++ {
		h = (h ^ (w & 0xff)) * fnvPrime64
		w >>= 8
	}
	return h
}

// fnvWords folds a 256-bit word, limb by limb.
func fnvWords(h uint64, v *uint256.Int) uint64 {
	for i := 0; i < 4; i++ {
		h = fnvWord(h, v[i])
	}
	return h
}

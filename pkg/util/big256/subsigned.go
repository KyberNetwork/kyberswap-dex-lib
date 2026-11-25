package big256

import "github.com/holiman/uint256"

// SubSigned sets (mag, neg) -= d, where (mag, neg) is a sign-magnitude value and d a two's-complement int256,
// matching a big.Int subtraction of ToBig(d). overflow reports a magnitude past 256 bits.
func SubSigned(mag *uint256.Int, neg bool, d *uint256.Int) (newNeg, overflow bool) {
	dNeg := d.Sign() < 0
	var dMag uint256.Int
	dMag.Set(d)
	if dNeg {
		dMag.Neg(&dMag)
	}
	// mag - d == mag + (-d), and -d has sign !dNeg.
	if neg == !dNeg {
		_, overflow = mag.AddOverflow(mag, &dMag)
		return neg, overflow
	}
	if mag.Lt(&dMag) {
		mag.Sub(&dMag, mag)
		return !neg, false
	}
	mag.Sub(mag, &dMag)
	return neg && !mag.IsZero(), false
}

package math

import "github.com/holiman/uint256"

// MaxDecayProduct is the last Q32.32 product supported by Core's tick helper.
const MaxDecayProduct uint64 = (uint64(MaxTick)+1)<<32 - 1

// DecayedLaunchFee mirrors feeDecay.sol, including the compact sqrt ratio and
// the saturation check before multiplication. Callers validate final <= initial.
func DecayedLaunchFee(initial, final, rateX32, elapsed uint64) uint64 {
	if rateX32 == 0 {
		return initial
	}
	if elapsed > MaxDecayProduct/rateX32 {
		return final
	}
	magnitude := (rateX32 * elapsed) >> 32 // guarded product is below 2**59
	factor := ToSqrtRatio(-int32(magnitude))
	var excess uint256.Int
	excess.SetUint64(initial - final)
	excess.Mul(&excess, factor).Rsh(&excess, 128) // product is below 2**192
	return final + excess.Uint64()
}

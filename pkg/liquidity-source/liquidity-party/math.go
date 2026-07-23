package liquidityparty

// math.go is the wei-exact Go port of the swap-relevant LMSR kernel and the PartyPoolHelpers
// external-wei ↔ internal-Q64.64 conversions. It reproduces, bit-for-bit:
//   - LMSRKernel.swapAmountsForExactInput  (../lmsr-amm/src/LMSRKernel.sol:183)
//   - LMSRKernel.amountInForExactOutput    (../lmsr-amm/src/LMSRKernel.sol:329)
//   - PartyPoolHelpers._ceilFee / _internalToUintFloorPure / _internalToUintCeilPure
//     (../lmsr-amm/src/PartyPoolHelpers.sol:26,84,88)
//   - PartyInfo._internalCeilFromUint      (../lmsr-amm/src/PartyInfo.sol:321)
//
// All fixed-point arithmetic delegates to the reusable abdkmath64x64 port so every rounding mode
// (truncation, ceilings, the exp_2/log_2 ladders) matches on-chain to the wei. Q64.64 signed values
// (kappa, effectiveSigmaQ, qInternal) are int256.Int; external wei quantities (amounts, bases) are
// uint256.Int.

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	abdk "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abdkmath64x64"
)

var (
	u1e6      = uint256.NewInt(1_000_000)
	uFeeRound = uint256.NewInt(999_999) // 1e6 - 1, the ceil addend in _ceilFee
	uOne      = uint256.NewInt(1)
	iOne      = int256.NewInt(1)
)

// swapAmountsForExactInput mirrors LMSRKernel.swapAmountsForExactInput (the 6-arg cached-sigma
// overload the pool's hot swap path uses). Frozen-b Hanson LMSR:
//
//	b = κ·effectiveSigmaQ;  invB = 1/b;  aOverB = a·invB (must be ≤ EXP_LIMIT);
//	inner = 1 + exp((q_j−q_i)·invB)·(1 − exp(−aOverB));
//	if inner ≤ 0 → output caps to q_j;  y = b·ln(inner);  if y ≤ 0 → 0.
//
// Returns the gross output in internal Q64.64 (amountOutInternal). effectiveSigmaQ must be > 0.
func swapAmountsForExactInput(kappa, effectiveSigmaQ *int256.Int, q []*int256.Int, i, j int, a *int256.Int) (int256.Int, error) {
	if effectiveSigmaQ.Sign() <= 0 {
		return int256.Int{}, ErrUninitialized
	}

	b, err := abdk.Mul(kappa, effectiveSigmaQ)
	if err != nil {
		return int256.Int{}, err
	}
	invB, err := abdk.Div(abdk.ONE, &b)
	if err != nil {
		return int256.Int{}, err
	}
	aOverB, err := abdk.Mul(a, &invB)
	if err != nil {
		return int256.Int{}, err
	}
	if aOverB.Cmp(abdk.EXP_LIMIT) > 0 {
		return int256.Int{}, ErrTooLarge
	}

	qDiff, err := abdk.Sub(q[j], q[i])
	if err != nil {
		return int256.Int{}, err
	}
	qDiffOverB, err := abdk.Mul(&qDiff, &invB)
	if err != nil {
		return int256.Int{}, err
	}
	r0, err := abdk.Exp(&qDiffOverB) // exp((q_j−q_i)/b)
	if err != nil {
		return int256.Int{}, err
	}
	negAOverB, err := abdk.Neg(&aOverB)
	if err != nil {
		return int256.Int{}, err
	}
	expNeg, err := abdk.Exp(&negAOverB) // exp(−a/b)
	if err != nil {
		return int256.Int{}, err
	}
	oneMinus, err := abdk.Sub(abdk.ONE, &expNeg) // 1 − exp(−a/b)
	if err != nil {
		return int256.Int{}, err
	}
	prod, err := abdk.Mul(&r0, &oneMinus)
	if err != nil {
		return int256.Int{}, err
	}
	inner, err := abdk.Add(abdk.ONE, &prod)
	if err != nil {
		return int256.Int{}, err
	}
	if inner.Sign() <= 0 {
		// Kernel caps output to q_j; a real swap would then revert in applySwap ("pool drained"),
		// so the caller treats this as capacity-exceeded rather than routing it.
		var capped int256.Int
		capped.Set(q[j])
		return capped, nil
	}

	lnInner, err := abdk.Ln(&inner)
	if err != nil {
		return int256.Int{}, err
	}
	y, err := abdk.Mul(&b, &lnInner)
	if err != nil {
		return int256.Int{}, err
	}
	if y.Sign() <= 0 {
		return int256.Int{}, nil // amountOut 0 -> caller rejects as "too small" via grossOut>0
	}
	return y, nil
}

// amountInForExactOutput mirrors LMSRKernel.amountInForExactOutput: the closed-form inverse of
// swapAmountsForExactInput. a = b·ln(r0 / (r0 + 1 − exp(y/b))), reverting on infeasible output.
// y (yInternal) must be > 0. Returns the required input in internal Q64.64.
func amountInForExactOutput(kappa, effectiveSigmaQ *int256.Int, q []*int256.Int, i, j int, y *int256.Int) (int256.Int, error) {
	if effectiveSigmaQ.Sign() <= 0 {
		return int256.Int{}, ErrUninitialized
	}

	b, err := abdk.Mul(kappa, effectiveSigmaQ)
	if err != nil {
		return int256.Int{}, err
	}
	invB, err := abdk.Div(abdk.ONE, &b)
	if err != nil {
		return int256.Int{}, err
	}

	qDiff, err := abdk.Sub(q[j], q[i])
	if err != nil {
		return int256.Int{}, err
	}
	qDiffOverB, err := abdk.Mul(&qDiff, &invB)
	if err != nil {
		return int256.Int{}, err
	}
	r0, err := abdk.Exp(&qDiffOverB)
	if err != nil {
		return int256.Int{}, err
	}

	expArg, err := abdk.Mul(y, &invB)
	if err != nil {
		return int256.Int{}, err
	}
	if expArg.Cmp(abdk.EXP_LIMIT) > 0 {
		return int256.Int{}, ErrTooLarge
	}
	e, err := abdk.Exp(&expArg) // exp(y/b)
	if err != nil {
		return int256.Int{}, err
	}

	// rhs = r0 + 1 − exp(y/b); must be > 0 for the pool to be able to deliver y ("too large").
	r0Plus1, err := abdk.Add(&r0, abdk.ONE)
	if err != nil {
		return int256.Int{}, err
	}
	rhs, err := abdk.Sub(&r0Plus1, &e)
	if err != nil {
		return int256.Int{}, err
	}
	if rhs.Sign() <= 0 {
		return int256.Int{}, ErrTooLarge
	}

	numer, err := abdk.Div(&r0, &rhs)
	if err != nil {
		return int256.Int{}, err
	}
	if numer.Sign() <= 0 {
		return int256.Int{}, ErrTooSmall
	}
	lnNumer, err := abdk.Ln(&numer)
	if err != nil {
		return int256.Int{}, err
	}
	amountIn, err := abdk.Mul(&b, &lnNumer)
	if err != nil {
		return int256.Int{}, err
	}
	if amountIn.Sign() <= 0 {
		return int256.Int{}, ErrTooSmall
	}
	return amountIn, nil
}

// ceilFee mirrors PartyPoolHelpers._ceilFee: ceil(x · feePpm / 1e6), rounded up to favor the pool.
// Per-asset fees are < 10_000 ppm so the pair feePpm is < 20_000 (< 1e6); for any realistic wei
// amount the multiply and the +999_999 addend stay well inside 256 bits, but we check explicitly so
// a pathological input rejects rather than silently wrapping (on-chain this runs in checked math).
func ceilFee(x *uint256.Int, feePpm uint64) (uint256.Int, error) {
	if feePpm == 0 {
		return uint256.Int{}, nil
	}
	var r uint256.Int
	if _, over := r.MulOverflow(x, uint256.NewInt(feePpm)); over {
		return uint256.Int{}, ErrOverflow
	}
	if _, over := r.AddOverflow(&r, uFeeRound); over {
		return uint256.Int{}, ErrOverflow
	}
	r.Div(&r, u1e6)
	return r, nil
}

// internalToUintFloor mirrors PartyPoolHelpers._internalToUintFloorPure: mulu(amount, base), the
// LP-favorable floor of the internal→external-wei conversion.
func internalToUintFloor(amount *int256.Int, base *uint256.Int) (uint256.Int, error) {
	return abdk.MulU(amount, base)
}

// internalToUintCeil mirrors PartyPoolHelpers._internalToUintCeilPure: mulu(amount, base) rounded
// up whenever the sub-ulp fractional remainder is non-zero. Truncating both operands to their low
// 64 bits is exact for the ceiling decision (only (frac·base) mod 2^64 determines a remainder).
func internalToUintCeil(amount *int256.Int, base *uint256.Int) (uint256.Int, error) {
	floored, err := abdk.MulU(amount, base)
	if err != nil {
		return uint256.Int{}, err
	}
	frac := amount[0] // low 64 bits of the Q64.64 value (amount >= 0 here)
	if frac == 0 {
		return floored, nil
	}
	baseL := base[0]     // low 64 bits of base
	if frac*baseL != 0 { // (frac·base) mod 2^64 != 0 -> a fractional remainder exists
		floored.Add(&floored, uOne)
	}
	return floored, nil
}

// internalCeilFromUint mirrors PartyInfo._internalCeilFromUint: the smallest Q64.64 ≥ n/base.
// floor(n/base) via divu, bumped by one ulp when re-multiplying floors below n.
func internalCeilFromUint(n, base *uint256.Int) (int256.Int, error) {
	if n.IsZero() || base.IsZero() {
		return int256.Int{}, nil
	}
	floorQ, err := abdk.DivU(n, base)
	if err != nil {
		return int256.Int{}, err
	}
	reproduced, err := abdk.MulU(&floorQ, base)
	if err != nil {
		return int256.Int{}, err
	}
	if reproduced.Cmp(n) < 0 {
		floorQ.Add(&floorQ, iOne) // +1 raw ulp (2^-64), matching int128 `floorQ + 1`
	}
	return floorQ, nil
}

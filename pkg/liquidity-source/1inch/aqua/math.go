package oneinchaqua

import (
	"slices"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	// uOne is the 1e18 fixed-point base of the concentrate math.
	uOne = big256.BONE
	uBps = uint256.NewInt(feeBpsDenominator)
	// uMaxUint248 bounds an Aqua balance.
	uMaxUint248 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(big256.U1, 248), 1)
	// uUnlimitedAllowance is the allowance from which a pull is not expected to change it.
	uUnlimitedAllowance = big256.U2Pow128
)

// quoteResult is what the router's quote computes for a program, before any wallet or fee collection
// checks.
type quoteResult struct {
	// AmountIn is the taker-defined input, restored after the fee wrappers have run.
	AmountIn  uint256.Int
	AmountOut uint256.Int
	// FeeIns are the protocol fee amounts in tokenIn, in the order the router pulls them: the wrapper
	// that is innermost in the program first. The caller may point it at a buffer it owns.
	FeeIns []uint256.Int
}

// runProgram executes the whitelisted program on the strategy's balances into res. Every intermediate
// that would overflow uint256 is an error, as the router's checked arithmetic reverts. res.FeeIns is
// appended to, so a caller that hands in a zero-length slice with capacity runs without allocating.
func runProgram(res *quoteResult, prog []Instruction, balanceIn, balanceOut, amountIn *uint256.Int, tokenInLt bool) error {
	return runProgramFrom(res, prog, balanceIn, balanceOut, amountIn, tokenInLt, false)
}

// runProgramFrom is runProgram; with concentrated set the balances are already the effective ones (see
// effectiveBalances) and the concentrate instructions are skipped. A concentrate changes only the balances,
// so the result is the same as running them again, without the square root.
func runProgramFrom(res *quoteResult, prog []Instruction, balanceIn, balanceOut, amountIn *uint256.Int, tokenInLt, concentrated bool) error {
	var (
		balIn, balOut, amtIn = *balanceIn, *balanceOut, *amountIn
		amtOut, bps          uint256.Int
		feeStart             = len(res.FeeIns)
		swapped              bool
	)
	for i := range prog {
		ins := &prog[i]
		if swapped && ins.Op != opSalt {
			return ErrBadProgram
		}
		switch ins.Op {
		case opSalt:
		case opXYCConcentrate:
			if concentrated {
				continue
			}
			if err := concentrate(&balIn, &balOut, &ins.SqrtPMin, &ins.SqrtPMax, tokenInLt); err != nil {
				return err
			}
		case opFlatFeeIn, opAquaProtocolFee:
			if ins.Bps > feeBpsDenominator {
				return ErrBadProgram
			}
			var fee uint256.Int
			bps.SetUint64(uint64(ins.Bps))
			if _, overflow := fee.MulOverflow(&amtIn, &bps); overflow {
				return ErrOverflow
			}
			protocol := ins.Op == opAquaProtocolFee
			if protocol {
				fee.Div(&fee, uBps) // floor
			} else {
				ceilDiv(&fee, &fee, uBps) // the flat fee rounds up
			}
			amtIn.Sub(&amtIn, &fee)
			if protocol {
				res.FeeIns = append(res.FeeIns, fee)
			}
		case opXYCSwap:
			if balIn.IsZero() || balOut.IsZero() {
				return ErrZeroBalance
			}
			var num, den uint256.Int
			if _, overflow := num.MulOverflow(&amtIn, &balOut); overflow {
				return ErrOverflow
			}
			if _, overflow := den.AddOverflow(&balIn, &amtIn); overflow {
				return ErrOverflow
			}
			amtOut.Div(&num, &den)
			swapped = true
		default:
			return ErrBadProgram
		}
	}
	if !swapped {
		return ErrBadProgram
	}
	// The router pulls the protocol fees from the innermost wrapper out, and restores the taker amountIn.
	slices.Reverse(res.FeeIns[feeStart:])
	res.AmountIn, res.AmountOut = *amountIn, amtOut
	return nil
}

// effectiveBalances applies the concentrate instructions of prog, in program order, to the balances. It
// stops at the swap: a concentrate after it is a program runProgram refuses.
func effectiveBalances(prog []Instruction, balIn, balOut *uint256.Int, tokenInLt bool) error {
	for i := range prog {
		switch ins := &prog[i]; ins.Op {
		case opXYCSwap:
			return nil
		case opXYCConcentrate:
			if err := concentrate(balIn, balOut, &ins.SqrtPMin, &ins.SqrtPMax, tokenInLt); err != nil {
				return err
			}
		}
	}
	return nil
}

// invertFees sets x to the smallest amountIn for which the fee wrappers of prog leave at least net for the
// swap, walking from the wrapper nearest the swap outwards. net must be above zero. With d = 1e9 - bps:
//
//	flat fee, which takes ceil(x*bps/1e9):      x - ceil(x*bps/1e9)  = floor(x*d/1e9), so x = ceil(net*1e9/d)
//	protocol fee, which takes floor(x*bps/1e9): x - floor(x*bps/1e9) = ceil(x*d/1e9),  so x = floor((net-1)*1e9/d) + 1
//
// Each wrapper maps x to a value that never grows by more than 1 per unit of x, so the smallest preimages
// compose. A 100% fee leaves nothing: no amountIn reaches net.
func invertFees(x *uint256.Int, prog []Instruction, net *uint256.Int) error {
	*x = *net
	for i := len(prog) - 1; i >= 0; i-- {
		ins := &prog[i]
		if ins.Op != opFlatFeeIn && ins.Op != opAquaProtocolFee {
			continue
		}
		if ins.Bps >= feeBpsDenominator {
			return ErrNoAmountIn
		}
		var d uint256.Int
		d.SetUint64(feeBpsDenominator - uint64(ins.Bps))
		if ins.Op == opFlatFeeIn {
			if !mulDivCeil(x, x, uBps, &d) {
				return ErrOverflow
			}
			continue
		}
		var t uint256.Int
		t.SubUint64(x, 1) // x is at least 1: it is net, or a smaller preimage that is also at least 1
		if !mulDiv(&t, &t, uBps, &d) {
			return ErrOverflow
		}
		if _, overflow := x.AddOverflow(&t, big256.U1); overflow {
			return ErrOverflow
		}
	}
	return nil
}

// concentrate adds the virtual liquidity of a price-bounded range to both balances.
func concentrate(balIn, balOut, sqrtPMin, sqrtPMax *uint256.Int, tokenInLt bool) error {
	bLt, bGt := balOut, balIn
	if tokenInLt {
		bLt, bGt = balIn, balOut
	}
	var l uint256.Int
	if err := computeL(&l, bLt, bGt, sqrtPMin, sqrtPMax); err != nil {
		return err
	}
	// L is below 2^189 (see computeL), so L*1e18 and L*1e18/sqrtPmax always fit; only the products
	// with sqrtPmin can overflow.
	var addIn, addOut, tmp uint256.Int
	if tokenInLt {
		// balanceIn += ceilDiv(L * 1e18, sqrtPmax); balanceOut += floor(L * sqrtPmin / 1e18)
		tmp.Mul(&l, uOne)
		ceilDiv(&addIn, &tmp, sqrtPMax)
		if !mulDiv(&addOut, &l, sqrtPMin, uOne) {
			return ErrOverflow
		}
	} else {
		// balanceIn += ceilDiv(L * sqrtPmin, 1e18); balanceOut += floor(L * 1e18 / sqrtPmax)
		if _, overflow := tmp.MulOverflow(&l, sqrtPMin); overflow {
			return ErrOverflow
		}
		ceilDiv(&addIn, &tmp, uOne)
		addOut.Mul(&l, uOne)
		addOut.Div(&addOut, sqrtPMax)
	}
	_, overflowIn := balIn.AddOverflow(balIn, &addIn)
	_, overflowOut := balOut.AddOverflow(balOut, &addOut)
	if overflowIn || overflowOut {
		return ErrOverflow
	}
	return nil
}

// computeL is the liquidity implied by the real balances and the price bounds (1e18 fixed point):
//
//	alpha = 1e18 - sqrtPmin*1e18/sqrtPmax
//	beta  = bLt*sqrtPmin/1e18 + bGt*1e18/sqrtPmax
//	L     = (beta + sqrt(beta^2 + (4*alpha*bLt/1e18)*bGt)) * 1e18 / (2*alpha)
func computeL(l, bLt, bGt, sqrtPMin, sqrtPMax *uint256.Int) error {
	var ratio, alpha, beta, part, fourAC, disc, root, num, den uint256.Int
	if !mulDiv(&ratio, sqrtPMin, uOne, sqrtPMax) {
		return ErrOverflow
	}
	alpha.Sub(uOne, &ratio) // sqrtPmin < sqrtPmax keeps the ratio below 1e18

	if !mulDiv(&beta, bLt, sqrtPMin, uOne) || !mulDiv(&part, bGt, uOne, sqrtPMax) {
		return ErrOverflow
	}
	if _, overflow := beta.AddOverflow(&beta, &part); overflow {
		return ErrOverflow
	}

	var fourAlpha uint256.Int
	fourAlpha.Lsh(&alpha, 2)
	if !mulDiv(&fourAC, &fourAlpha, bLt, uOne) {
		return ErrOverflow
	}
	if _, overflow := fourAC.MulOverflow(&fourAC, bGt); overflow {
		return ErrOverflow
	}
	if _, overflow := disc.MulOverflow(&beta, &beta); overflow {
		return ErrOverflow
	}
	if _, overflow := disc.AddOverflow(&disc, &fourAC); overflow {
		return ErrOverflow
	}
	// beta*beta fit in 256 bits, so beta and its root are below 2^128: the sum below 2^129 and the
	// final product with 1e18 below 2^189 cannot overflow.
	root.Sqrt(&disc)
	num.Add(&beta, &root)
	den.Lsh(&alpha, 1)
	l.Mul(&num, uOne)
	l.Div(l, &den)
	return nil
}

// mulDiv sets z = floor(x*y/d) with a 512-bit intermediate and reports false when d is zero or the
// result does not fit in 256 bits. It is not big256.MulDivDown, which drops the overflow flag the router's
// mulDiv turns into a revert. Operands whose product fits in 256 bits take the cheaper Mul and Div.
func mulDiv(z, x, y, d *uint256.Int) bool {
	if d.IsZero() {
		return false
	}
	if x.BitLen()+y.BitLen() <= 256 {
		z.Mul(x, y)
		z.Div(z, d)
		return true
	}
	_, overflow := z.MulDivOverflow(x, y, d)
	return !overflow
}

// mulDivCeil sets z = ceil(x*y/d) with a 512-bit intermediate and reports false when d is zero or the result
// does not fit in 256 bits.
func mulDivCeil(z, x, y, d *uint256.Int) bool {
	return !d.IsZero() && v3Utils.MulDivRoundingUpV2(x, y, d, z) == nil
}

// ceilDiv sets z = ceil(x/d) for d != 0.
func ceilDiv(z, x, d *uint256.Int) {
	if x.IsZero() {
		z.Clear()
		return
	}
	var t uint256.Int
	t.SubUint64(x, 1)
	z.Div(&t, d)
	z.AddUint64(z, 1)
}

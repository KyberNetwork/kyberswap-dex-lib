package uniswapv4

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Helpers for hooks to answer CalcIn (CalcOut=false), which reverses exact-in.

// GrossBeforeFee inverts an exact-in fee deduction net = gross - floor(gross*feeNum/feeDenom): it returns the min
// gross leaving at least net, i.e. floor((net-1)*feeDenom/(feeDenom-feeNum)) + 1. feeNum must be < feeDenom.
func GrossBeforeFee(net, feeNum, feeDenom *big.Int) *big.Int {
	var gross, denom big.Int
	if net.Sign() <= 0 {
		return &gross
	}
	gross.Sub(net, bignumber.One)
	bignumber.MulDivDown(&gross, &gross, feeDenom, denom.Sub(feeDenom, feeNum))
	return gross.Add(&gross, bignumber.One)
}

// maxAmountBits bounds MinAmountIn's search to uint256 amounts.
const maxAmountBits = 256

// MinAmountIn returns the min amountIn with calcAmountOut(amountIn) >= amountOut, for hooks that only price
// exact-in. calcAmountOut must be non-decreasing where it succeeds, and may error outside the tradable range: below
// a successful amountIn an error means too little input, above it the end of the range. It brackets from guess
// (amountOut if nil) by halving/doubling, then bisects. calcAmountOut must not retain amountIn, which gets reused.
func MinAmountIn(amountOut, guess *big.Int, calcAmountOut func(amountIn *big.Int) (*big.Int, error)) (*big.Int,
	error) {
	if amountOut == nil || amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}
	probe := func(amountIn *big.Int) (reached, valid bool) {
		out, err := calcAmountOut(amountIn)
		return err == nil && out.Cmp(amountOut) >= 0, err == nil
	}

	// find a valid amountIn: halve from guess, then double from it
	start := amountOut
	if guess != nil && guess.Sign() > 0 {
		start = guess
	}
	hi := new(big.Int).Set(start)
	reached, valid := probe(hi)
	for !valid && hi.Cmp(bignumber.One) > 0 {
		reached, valid = probe(hi.Rsh(hi, 1))
	}
	if !valid {
		hi.Set(start)
	}
	for !valid {
		if hi.Lsh(hi, 1).BitLen() > maxAmountBits {
			return nil, ErrAmountOutUnreachable
		}
		reached, valid = probe(hi)
	}

	var lo, mid big.Int // lo does not reach amountOut, hi does
	if reached {
		for hi.Cmp(bignumber.One) > 0 { // halve down to a bracket
			if r, _ := probe(mid.Rsh(hi, 1)); !r {
				lo.Set(&mid)
				break
			}
			hi.Set(&mid)
		}
	} else {
		for lo.Set(hi); ; lo.Set(hi) { // double up to a bracket
			if hi.Lsh(hi, 1).BitLen() > maxAmountBits {
				return nil, ErrAmountOutUnreachable
			} else if reached, valid = probe(hi); reached || !valid {
				break
			}
		}
		for !reached { // hi is past the tradable range: look below it for an amountIn that reaches
			if mid.Sub(hi, &lo).Cmp(bignumber.One) <= 0 {
				return nil, ErrAmountOutUnreachable
			} else if reached, valid = probe(mid.Add(&lo, hi).Rsh(&mid, 1)); reached || !valid {
				hi.Set(&mid)
			} else {
				lo.Set(&mid)
			}
		}
	}

	for mid.Sub(hi, &lo).Cmp(bignumber.One) > 0 {
		if r, _ := probe(mid.Add(&lo, hi).Rsh(&mid, 1)); r {
			hi.Set(&mid)
		} else {
			lo.Set(&mid)
		}
	}
	return hi, nil
}

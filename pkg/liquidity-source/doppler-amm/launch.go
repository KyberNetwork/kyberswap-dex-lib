package doppleramm

import (
	"github.com/holiman/uint256"

	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

const feePipsDenominator = 1_000_000

// launchSwap walks a Launch Market exact-input swap in place: sqrtP and liquidity
// are updated to the post-trade state. The module is Uniswap V3 step math over the
// configured level boundaries only (no bitmap word edges). Unlike V3 it takes the
// fee once up front, floor(in*(1e6-fee)/1e6), and walks the steps on the net
// amount with zero fee; V3's per-step fee rounding differs by a wei of input at
// some crossings. Both facts were fixed against DopplerQuoter, wei-exact across
// boundary crossings. bounds are ascending boundary sqrt prices, nets the signed
// liquidity change at each (two's complement).
func launchSwap(
	bounds []uint256.Int, nets []uint256.Int, feePips uint32,
	sqrtP, liquidity, amountIn *uint256.Int, zeroForOne bool,
) (amountOut uint256.Int, err error) {
	// first boundary strictly above the price; a price on a boundary sits in the upper range
	up := 0
	for up < len(bounds) && !bounds[up].Gt(sqrtP) {
		up++
	}

	var rem, next, in, out, fee uint256.Int
	// bounded: amountIn < 2^256/1e6 is enforced by the overflow check
	if _, overflow := rem.MulOverflow(amountIn, uint256.NewInt(feePipsDenominator-uint64(feePips))); overflow {
		return amountOut, ErrOverflow
	}
	rem.Div(&rem, uint256.NewInt(feePipsDenominator))

	i, step := up, 1
	if zeroForOne {
		i, step = up-1, -1
	}
	for !rem.IsZero() {
		if i < 0 || i >= len(bounds) {
			return amountOut, ErrInsufficientLiquidity
		}
		target := &bounds[i]
		if liquidity.IsZero() {
			next.Set(target)
			in.Clear()
			out.Clear()
		} else if err = v3.ComputeSwapStep(sqrtP, target, liquidity, &rem, 0, &next, &in, &out, &fee); err != nil {
			return amountOut, err
		}
		if next.Eq(target) {
			rem.Sub(&rem, &in)
		} else {
			rem.Clear() // a step that stops short spends the rest of the input, as in V3
		}
		amountOut.Add(&amountOut, &out)
		sqrtP.Set(&next)
		if sqrtP.Eq(target) {
			if zeroForOne {
				liquidity.Sub(liquidity, &nets[i])
			} else {
				liquidity.Add(liquidity, &nets[i])
			}
			i += step
		}
	}
	return amountOut, nil
}

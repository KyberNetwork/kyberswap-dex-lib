package dexv2

import (
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var maxDynamicPriceX96 = uint256.MustFromDecimal("4647680745692069522618647333321942173198062861119228")

// dynamicFeeState holds one swap's decayed anchor and kink prices. All prices
// use Q96, while the two sqrt kinks use Q96 square roots.
type dynamicFeeState struct {
	packed, anchor, minPrice, maxPrice, minSqrt, maxSqrt uint256.Int
	minFee, maxFee, factor                               uint256.Int
}

func packedBits(x *uint256.Int, offset, width uint) uint64 {
	var t, mask uint256.Int
	mask.SetOne().Lsh(&mask, width).SubUint64(&mask, 1)
	return t.Rsh(x, offset).And(&t, &mask).Uint64()
}
func putPackedBits(x *uint256.Int, offset, width uint, value uint64) {
	var mask, v uint256.Int
	mask.SetOne().Lsh(&mask, width).SubUint64(&mask, 1).Lsh(&mask, offset).Not(&mask)
	x.And(x, &mask)
	v.SetUint64(value).Lsh(&v, offset)
	x.Or(x, &v)
}

func priceAtSqrt(dst, sqrt *uint256.Int) error {
	var q uint256.Int
	q.SetOne().Lsh(&q, 96)
	if _, overflow := dst.MulDivOverflow(sqrt, sqrt, &q); overflow {
		return ErrOverflow
	}
	return nil
}

func dynamicKink(dst, sqrt, anchor *uint256.Int, impact uint64, zeroForOne bool) error {
	if impact > 1_000_000 {
		impact = 1_000_000
	}
	factor := uint64(1_000_000) + impact
	if zeroForOne {
		factor = 1_000_000 - impact
	}
	var f, denom uint256.Int
	f.SetUint64(factor)
	denom.SetUint64(1_000_000)
	if _, overflow := dst.MulDivOverflow(anchor, &f, &denom); overflow {
		return ErrOverflow
	}
	var min uint256.Int
	min.SetUint64(1_350_587)
	if dst.Cmp(&min) < 0 {
		dst.Set(&min)
	} else if dst.Cmp(maxDynamicPriceX96) > 0 {
		dst.Set(maxDynamicPriceX96)
	}
	var v uint256.Int
	var x160 uint256.Int
	x160.SetOne().Lsh(&x160, 160).SubUint64(&x160, 1)
	if dst.Cmp(&x160) < 0 {
		v.Lsh(dst, 96)
		sqrt.Sqrt(&v)
	} else {
		v.Lsh(dst, 84)
		sqrt.Sqrt(&v).Lsh(sqrt, 6)
	}
	return nil
}

func prepareDynamicFee(packed, sqrt *uint256.Int, zeroForOne bool, timestamp uint64) (dynamicFeeState, error) {
	var d dynamicFeeState
	d.packed.Set(packed)
	last := packedBits(packed, 229, 15)
	now := timestamp & 32767
	elapsed := (now + 32768 - last) & 32767
	remaining := packedBits(packed, 244, 12)
	impact := packedBits(packed, 209, 20)
	positive := packedBits(packed, 208, 1) != 0
	if elapsed < remaining {
		impact = impact * (remaining - elapsed) / remaining
		if impact == 0 {
			positive = true
		}
		remaining -= elapsed
	} else {
		impact = 0
		remaining = 0
		positive = false
	}
	sign := uint64(0)
	if positive {
		sign = 1
	}
	putPackedBits(&d.packed, 208, 1, sign)
	putPackedBits(&d.packed, 209, 20, impact)
	putPackedBits(&d.packed, 229, 15, now)
	putPackedBits(&d.packed, 244, 12, remaining)
	var price, oneMillion, denominator uint256.Int
	if err := priceAtSqrt(&price, sqrt); err != nil {
		return d, err
	}
	oneMillion.SetUint64(1_000_000)
	n := uint64(1_000_000)
	if positive {
		n += impact
	} else {
		if impact >= n {
			return d, ErrUnsupportedDynamicFee
		}
		n -= impact
	}
	denominator.SetUint64(n)
	if _, overflow := d.anchor.MulDivOverflow(&price, &oneMillion, &denominator); overflow || d.anchor.IsZero() {
		return d, ErrUnsupportedDynamicFee
	}
	minFee := packedBits(packed, 176, 16)
	maxFee := packedBits(packed, 192, 16)
	factor := packedBits(packed, 168, 8)
	d.minFee.SetUint64(minFee)
	d.maxFee.SetUint64(maxFee)
	d.factor.SetUint64(factor)
	if err := dynamicKink(&d.minPrice, &d.minSqrt, &d.anchor, minFee*factor, zeroForOne); err != nil {
		return d, err
	}
	if err := dynamicKink(&d.maxPrice, &d.maxSqrt, &d.anchor, maxFee*factor, zeroForOne); err != nil {
		return d, err
	}
	return d, nil
}

func finishDynamicFee(d *dynamicFeeState, sqrt *uint256.Int) (uint256.Int, error) {
	var price, diff, scale, impact uint256.Int
	if err := priceAtSqrt(&price, sqrt); err != nil {
		return uint256.Int{}, err
	}
	positive := price.Cmp(&d.anchor) > 0
	if positive {
		diff.Sub(&price, &d.anchor)
	} else {
		diff.Sub(&d.anchor, &price)
	}
	scale.SetUint64(1_000_000)
	if _, overflow := impact.MulDivOverflow(&diff, &scale, &d.anchor); overflow || impact.Cmp(&scale) >= 0 {
		return uint256.Int{}, ErrUnsupportedDynamicFee
	}
	final := impact.Uint64()
	prev := packedBits(&d.packed, 209, 20)
	prevPositive := packedBits(&d.packed, 208, 1) != 0
	remaining := packedBits(&d.packed, 244, 12)
	maxDecay := packedBits(&d.packed, 156, 12)
	if final == 0 {
		remaining = 0
	} else if prev == 0 || positive != prevPositive {
		remaining = maxDecay
	} else {
		remaining = remaining * final / prev
		if remaining > maxDecay {
			remaining = maxDecay
		}
	}
	sign := uint64(0)
	if positive || final == 0 {
		sign = 1
	}
	putPackedBits(&d.packed, 208, 1, sign)
	putPackedBits(&d.packed, 209, 20, final)
	putPackedBits(&d.packed, 244, 12, remaining)
	return d.packed, nil
}

// dynamicSwapStep mirrors the deployed min, dynamic and max fee substeps.
func dynamicSwapStep(d *dynamicFeeState, zeroForOne bool, current, target, liquidity *uint256.Int, remaining *int256.Int, protocolFee *uint256.Int, next, amountIn, netOut, grossOut, protocolOut, lpOut *uint256.Int) error {
	next.Set(current)
	var price uint256.Int
	if err := priceAtSqrt(&price, next); err != nil {
		return err
	}
	var rem int256.Int
	rem.Set(remaining)
	inZone := func(price, kink *uint256.Int) bool {
		if zeroForOne {
			return price.Cmp(kink) > 0
		}
		return price.Cmp(kink) < 0
	}
	for zone := 0; zone < 3; zone++ {
		var kink, sqrt, fee *uint256.Int
		switch zone {
		case 0:
			kink = &d.minPrice
			sqrt = &d.minSqrt
			fee = &d.minFee
		case 1:
			kink = &d.maxPrice
			sqrt = &d.maxSqrt
		default:
			fee = &d.maxFee
		}
		if zone < 2 && !inZone(&price, kink) {
			continue
		}
		stepTarget := target
		if zone < 2 && ((zeroForOne && sqrt.Cmp(target) > 0) || (!zeroForOne && sqrt.Cmp(target) < 0)) {
			stepTarget = sqrt
		}
		var stepIn, stepOut, unused, nextStep uint256.Int
		if err := utils.ComputeSwapStep(next, stepTarget, liquidity, &rem, 0, &nextStep, &stepIn, &stepOut, &unused); err != nil {
			return err
		}
		next.Set(&nextStep)
		var inSigned int256.Int
		if err := utils.ToInt256(&stepIn, &inSigned); err != nil {
			return err
		}
		rem.Sub(&rem, &inSigned)
		var dynamicFee uint256.Int
		if stepOut.BitLen() > 86 {
			return ErrOverflow
		}
		if zone == 1 && !stepOut.IsZero() {
			var end, mean, diff, impact, scale uint256.Int
			if err := priceAtSqrt(&end, next); err != nil {
				return err
			}
			mean.Add(&price, &end).Rsh(&mean, 1)
			if zeroForOne {
				diff.Sub(&d.anchor, &mean)
			} else {
				diff.Sub(&mean, &d.anchor)
			}
			scale.SetUint64(1_000_000)
			// Solidity deliberately computes the directional difference and product
			// unchecked; preserve modulo-256 behavior at a zero-kink boundary.
			impact.Mul(&diff, &scale).Div(&impact, &d.anchor)
			if !impact.IsZero() {
				dynamicFee.AddUint64(&impact, 1).Div(&dynamicFee, &d.factor).AddUint64(&dynamicFee, 1)
			}
			fee = &dynamicFee
		} else if zone == 1 {
			fee = &dynamicFee
		}
		var net, protocol, lp uint256.Int
		netSwapInStep(&net, &protocol, &lp, &stepOut, protocolFee, fee)
		if _, o := protocolOut.AddOverflow(protocolOut, &protocol); o {
			return ErrOverflow
		}
		if _, o := lpOut.AddOverflow(lpOut, &lp); o {
			return ErrOverflow
		}
		if _, o := amountIn.AddOverflow(amountIn, &stepIn); o {
			return ErrOverflow
		}
		if _, o := netOut.AddOverflow(netOut, &net); o {
			return ErrOverflow
		}
		if _, o := grossOut.AddOverflow(grossOut, &stepOut); o {
			return ErrOverflow
		}
		if next.Eq(target) || rem.IsZero() {
			return nil
		}
		if zone < 2 {
			price.Set(kink)
		}
	}
	return nil
}

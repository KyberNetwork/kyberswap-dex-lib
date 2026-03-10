package poe

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func computeVirtualReserves(reserveX, reserveY, price, alpha *uint256.Int) *virtualReserves {
	if reserveX.IsZero() && reserveY.IsZero() {
		return &virtualReserves{xv: new(uint256.Int), yv: new(uint256.Int)}
	}

	sqrtPa := u256.MulDiv(price, bps, alpha)
	sqrtPa.Mul(sqrtPa, pricePrecision).Sqrt(sqrtPa)

	sqrtPb := u256.MulDiv(price, alpha, bps)
	sqrtPb.Mul(sqrtPb, pricePrecision).Sqrt(sqrtPb)

	if sqrtPa.IsZero() || sqrtPb.IsZero() || sqrtPa.Cmp(sqrtPb) >= 0 {
		return &virtualReserves{xv: new(uint256.Int).Set(reserveX), yv: new(uint256.Int).Set(reserveY)}
	}

	SP := pricePrecision

	a := new(uint256.Int).Sub(sqrtPb, sqrtPa)
	if a.IsZero() {
		return &virtualReserves{xv: new(uint256.Int).Set(reserveX), yv: new(uint256.Int).Set(reserveY)}
	}

	tmp := new(uint256.Int)

	b := u256.MulDiv(reserveX, tmp.Mul(sqrtPa, sqrtPb), SP)
	b.Add(b, tmp.Mul(reserveY, SP))

	disc := new(uint256.Int).Mul(reserveX, reserveY)
	disc.Mul(disc, sqrtPb).Mul(disc, a).Lsh(disc, 2)
	disc.Add(disc, tmp.Mul(b, b))
	disc.Sqrt(disc)

	L := b.Add(b, disc)
	L.Div(L, tmp.Lsh(a, 1))

	xv := new(uint256.Int).Add(reserveX, u256.MulDivDown(disc, L, SP, sqrtPb))
	yv := new(uint256.Int).Add(reserveY, u256.MulDivDown(tmp, L, sqrtPa, SP))

	return &virtualReserves{xv: xv, yv: yv}
}

func calcAmountOutCPMM(xv, yv, netAmountIn *uint256.Int) *uint256.Int {
	if xv.IsZero() || netAmountIn.IsZero() {
		return new(uint256.Int)
	}

	denom := new(uint256.Int).Add(xv, netAmountIn)
	if denom.IsZero() {
		return new(uint256.Int)
	}

	return u256.MulDivDown(denom, yv, netAmountIn, denom)
}

func calcAmountInCPMM(xv, yv, amountOut *uint256.Int) *uint256.Int {
	if yv.Cmp(amountOut) <= 0 {
		return nil
	}

	denom := new(uint256.Int).Sub(yv, amountOut)
	if denom.IsZero() {
		return nil
	}

	return u256.MulDivUp(denom, xv, amountOut, denom)
}

func applyFeeCeil(amount, feePPM *uint256.Int) *uint256.Int {
	if feePPM.IsZero() {
		return new(uint256.Int)
	}

	return u256.MulDivUp(new(uint256.Int), amount, feePPM, feePrecision)
}

func deductFeeCeil(amount, feePPM *uint256.Int) *uint256.Int {
	if feePPM.IsZero() {
		return new(uint256.Int)
	}

	remaining := new(uint256.Int).Sub(feePrecision, feePPM)
	if remaining.IsZero() {
		return nil
	}

	return u256.MulDivUp(remaining, amount, feePPM, remaining)
}

package brownfi

import (
	"fmt"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var (
	XAuxConst64, _ = uint256.FromHex("0x100000000000000000000000000000000")
	XAuxConst32, _ = uint256.FromHex("0x10000000000000000")
	XAuxConst16, _ = uint256.FromHex("0x100000000")
	XAuxConst8, _  = uint256.FromHex("0x10000")
	XAuxConst4, _  = uint256.FromHex("0x100")
	XAuxConst2, _  = uint256.FromHex("0x10")
	XAuxConst1, _  = uint256.FromHex("0x4")
)

func sqrt(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return new(uint256.Int)
	}

	// x = new(uint256.Int).Mul(x, bignumber.BONE)
	// Set the initial guess to the least power of two that is greater than or equal to sqrt(x).
	var xAux uint256.Int
	xAux.Set(x)
	result := uint256.NewInt(1)
	if xAux.Cmp(XAuxConst64) >= 0 {
		xAux.Rsh(&xAux, 128)
		result.Lsh(result, 64)
	}
	if xAux.Cmp(XAuxConst32) >= 0 {
		xAux.Rsh(&xAux, 64)
		result.Lsh(result, 32)
	}
	if xAux.Cmp(XAuxConst16) >= 0 {
		xAux.Rsh(&xAux, 32)
		result.Lsh(result, 16)
	}
	if xAux.Cmp(XAuxConst8) >= 0 {
		xAux.Rsh(&xAux, 16)
		result.Lsh(result, 8)
	}
	if xAux.Cmp(XAuxConst4) >= 0 {
		xAux.Rsh(&xAux, 8)
		result.Lsh(result, 4)
	}
	if xAux.Cmp(XAuxConst2) >= 0 {
		xAux.Rsh(&xAux, 4)
		result.Lsh(result, 2)
	}
	if xAux.Cmp(XAuxConst1) >= 0 {
		result.Lsh(result, 1)
	}

	for i := 0; i < 7; i++ { // Seven iterations should be enough
		xAux.Div(x, result)
		result.Add(result, &xAux)
		result.Rsh(result, 1)
	}

	roundedDownResult := xAux.Div(x, result)
	if result.Cmp(roundedDownResult) >= 0 {
		return roundedDownResult
	}
	return result
}

func delta(amountIn, reserveOut *uint256.Int, kappa, oPrice *uint256.Int, isSell bool) (*uint256.Int, error) {
	temp1 := new(uint256.Int)
	if isSell {
		// temp1 = (P * dx - y)^2
		res, _ := v3Utils.MulDiv(oPrice, amountIn, q128)
		temp1 = lo.Ternary(res.Cmp(reserveOut) < 0, new(uint256.Int).Sub(reserveOut, res), new(uint256.Int).Sub(res, reserveOut))
		temp1.Mul(temp1, temp1)
	} else {
		// temp1 = (P * x - dy)^2
		res, _ := v3Utils.MulDiv(oPrice, reserveOut, q128)
		temp1 = lo.Ternary(res.Cmp(amountIn) < 0, new(uint256.Int).Sub(amountIn, res), new(uint256.Int).Sub(res, amountIn))
		temp1.Mul(temp1, temp1)
	}
	// temp2 = 2 * P * K * y * dx
	temp2, _ := v3Utils.MulDiv(oPrice, amountIn, q128)
	temp3, _ := v3Utils.MulDiv(kappa, reserveOut, q128)
	temp2.Mul(temp2, temp3).Mul(temp2, uint256.NewInt(2))
	fmt.Println("temp1", temp1, "temp2", temp2)
	// _delta = temp1.add(temp2)
	delta := new(uint256.Int).Add(temp1, temp2)
	return delta, nil
}

func getAmountOut(amountIn, reserveOut *uint256.Int, kappa, oPrice *uint256.Int, zeroForOne bool, fee *uint256.Int, feePrecision *uint256.Int) *uint256.Int {
	// Check if kappa equals 2 * Q128
	amountOut, numerator, denominator := new(uint256.Int), new(uint256.Int), new(uint256.Int)
	if kappa.Cmp(q128x2) == 0 {
		if zeroForOne {
			// dy = P * y * dx / (P * dx + y)
			numerator, _ = v3Utils.MulDiv(oPrice, reserveOut, q128)
			denominator, _ = v3Utils.MulDivRoundingUp(oPrice, amountIn, q128)
			numerator.Mul(numerator, amountIn)
			denominator.Add(denominator, reserveOut)
		} else {
			// dx = (x * dy) / (P * x + dy)
			numerator.Mul(amountIn, reserveOut)
			denominator, _ = v3Utils.MulDivRoundingUp(oPrice, reserveOut, q128)
			denominator.Add(denominator, amountIn)
		}
		amountOut = new(uint256.Int).Div(numerator, denominator)
	} else {
		delta, _ := delta(amountIn, reserveOut, kappa, oPrice, zeroForOne)
		if zeroForOne {
			// P * dx + y - sqrt(delta)
			numerator, _ = v3Utils.MulDiv(oPrice, amountIn, q128)
			numerator.Add(numerator, reserveOut).Sub(numerator, sqrt(delta))
			// (2 - K)
			denominator.Sub(q128x2, kappa)
		} else {
			// P * x + dy - sqrt(delta)
			numerator, _ = v3Utils.MulDiv(oPrice, reserveOut, q128)
			numerator.Add(numerator, amountIn).Sub(numerator, sqrt(delta))
			// P * (2 - K)
			denominator, _ = v3Utils.MulDiv(oPrice, denominator.Sub(q128x2, kappa), q128)
		}
		amountOut, _ = v3Utils.MulDiv(numerator, q128, denominator)
	}

	// Apply fee: amountOut * (FEE_DENOMINATOR - fee) / FEE_DENOMINATOR
	amountOut.Mul(amountOut, new(uint256.Int).Sub(feePrecision, fee)).Div(amountOut, feePrecision)
	return amountOut
}

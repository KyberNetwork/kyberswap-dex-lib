package sd59x18

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/math"
	"github.com/holiman/uint256"
)

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/lib/prb-math/src/sd59x18/Math.sol#L480
func (z *SD59x18) Log2(x *SD59x18) (*SD59x18, error) {
	xInt := x.value
	if xInt.Cmp(integer.Zero()) <= 0 {
		return nil, Err_PRBMath_SD59x18_Log_InputTooSmall
	}

	var sign *big.Int
	if xInt.Cmp(uUNIT) >= 0 {
		sign = big.NewInt(1)
	} else {
		sign = big.NewInt(-1)

		xInt = new(big.Int).Quo(uUNIT_SQUARED, xInt)
	}

	var n *big.Int
	{
		xInt_div_uUNIT := new(big.Int).Quo(xInt, uUNIT)
		xInt_div_uUNIT_U256, _ := uint256.FromBig(xInt_div_uUNIT)
		nUint256 := math.Common.Msb(xInt_div_uUNIT_U256)

		n = nUint256.ToBig()
	}

	resultInt := new(big.Int).Mul(n, uUNIT)
	y := new(big.Int).Rsh(xInt, uint(n.Uint64()))
	if y.Cmp(uUNIT) == 0 {
		z.value = new(big.Int).Mul(resultInt, sign)
		return z, nil
	}

	doubleUnit := big.NewInt(2e18)
	for delta := uHALF_UNIT; delta.Cmp(integer.Zero()) > 0; delta = new(big.Int).Rsh(delta, 1) {
		y = new(big.Int).Quo(new(big.Int).Mul(y, y), uUNIT)
		if y.Cmp(doubleUnit) >= 0 {
			resultInt = new(big.Int).Add(resultInt, delta)
			y = new(big.Int).Rsh(y, 1)
		}
	}

	resultInt = new(big.Int).Mul(resultInt, sign)
	z.value = resultInt

	return z, nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/lib/prb-math/src/sd59x18/Math.sol#L201
func (z *SD59x18) Exp2(x *SD59x18) (*SD59x18, error) {
	xInt := x.value

	if xInt.Cmp(integer.Zero()) < 0 {
		magic, _ := new(big.Int).SetString("-59794705707972522261", 10)
		if xInt.Cmp(magic) < 0 {
			z.value = integer.Zero()
			return z, nil
		}

		t, err := new(SD59x18).Exp2(SD(
			new(big.Int).Sub(integer.Zero(), xInt),
		))
		if err != nil {
			return z, err
		}

		z.value = new(big.Int).Quo(uUNIT_SQUARED, t.value)

		return z, nil
	}

	if xInt.Cmp(uEXP2_MAX_INPUT) > 0 {
		return z, Err_PRBMath_SD59x18_Exp2_InputTooBig
	}

	x_192x64, _ := uint256.FromBig(new(big.Int).Quo(new(big.Int).Lsh(xInt, 64), uUNIT))
	z.value = math.Common.Exp2(x_192x64).ToBig()

	return z, nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/lib/prb-math/src/sd59x18/Math.sol#L593
func (z *SD59x18) Pow(x *SD59x18, y *SD59x18) (*SD59x18, error) {
	xInt, yInt := x.value, y.value

	if xInt.Cmp(integer.Zero()) == 0 {
		z.value = integer.Zero()
		if yInt.Cmp(integer.Zero()) == 0 {
			z.value = uUNIT
		}

		return z, nil
	}

	if xInt.Cmp(uUNIT) == 0 {
		z.value = uUNIT
		return z, nil
	}

	if yInt.Cmp(integer.Zero()) == 0 {
		z.value = uUNIT
		return z, nil
	}

	if yInt.Cmp(uUNIT) == 0 {
		z.value = new(big.Int).Set(xInt)
		return z, nil
	}

	log2, err := new(SD59x18).Log2(x)
	if err != nil {
		return z, err
	}

	mul, err := new(SD59x18).Mul(log2, y)
	if err != nil {
		return z, err
	}

	exp2, err := new(SD59x18).Exp2(mul)
	if err != nil {
		return z, err
	}

	z.value = exp2.value

	return z, nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/lib/prb-math/src/sd59x18/Math.sol#L545
func (z *SD59x18) Mul(x, y *SD59x18) (*SD59x18, error) {
	xInt, yInt := x.value, y.value

	if xInt.Cmp(uMIN_SD59x18) == 0 || yInt.Cmp(uMIN_SD59x18) == 0 {
		return z, Err_PRBMath_SD59x18_Mul_InputTooSmall
	}

	xAbs := new(big.Int).Abs(xInt)
	yAbs := new(big.Int).Abs(yInt)

	resultAbsU256, err := math.Common.MulDiv18(
		uint256.MustFromBig(xAbs),
		uint256.MustFromBig(yAbs),
	)
	if err != nil {
		return nil, err
	}
	resultAbs := resultAbsU256.ToBig()

	if resultAbs.Cmp(uMAX_SD59x18) > 0 {
		return z, Err_PRBMath_SD59x18_Mul_Overflow
	}

	result := resultAbs
	if xInt.Sign() != yInt.Sign() {
		result = new(big.Int).Neg(resultAbs)
	}

	z.value = result

	return z, nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/lib/prb-math/src/sd59x18/Math.sol#L121
func (z *SD59x18) Div(x, y *SD59x18) (*SD59x18, error) {
	xInt, yInt := x.value, y.value

	if xInt.Cmp(uMIN_SD59x18) == 0 || yInt.Cmp(uMIN_SD59x18) == 0 {
		return nil, Err_PRBMath_SD59x18_Div_InputTooSmall
	}

	var (
		xAbs = new(big.Int).Abs(xInt)
		yAbs = new(big.Int).Abs(yInt)
	)

	resultAbsU256, err := math.Common.MulDiv(
		uint256.MustFromBig(xAbs),
		uint256.MustFromBig(uUNIT),
		uint256.MustFromBig(yAbs),
	)
	if err != nil {
		return z, err
	}
	resultAbs := resultAbsU256.ToBig()

	if resultAbs.Cmp(uMAX_SD59x18) > 0 {
		return z, Err_PRBMath_SD59x18_Div_Overflow
	}

	result := resultAbs
	if xInt.Sign() != yInt.Sign() {
		result = new(big.Int).Neg(resultAbs)
	}

	z.value = result

	return z, nil
}

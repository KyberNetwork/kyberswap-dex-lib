package tricryptong

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// only sort slice of 3 elements
func sort_inplace(x []uint256.Int) {
	if x[0].Cmp(&x[1]) < 0 {
		tmp := number.Set(&x[0])
		x[0].Set(&x[1])
		x[1].Set(tmp)
	}
	if x[0].Cmp(&x[2]) < 0 {
		tmp := number.Set(&x[0])
		x[0].Set(&x[2])
		x[2].Set(tmp)
	}
	if x[1].Cmp(&x[2]) < 0 {
		tmp := number.Set(&x[1])
		x[1].Set(&x[2])
		x[2].Set(tmp)
	}
}

// func _geometric_mean(unsorted_x []*big.Int, sort bool) (*big.Int, error) {
// 	var nCoins = len(unsorted_x)
// 	var nCoinsBi = big.NewInt(int64(nCoins))
// 	var x = unsorted_x
// 	if sort {
// 		x = sortArray(unsorted_x)
// 	}
// 	var D = x[0]
// 	var diff = constant.ZeroBI
// 	for i := 0; i < 255; i += 1 {
// 		var D_prev = D
// 		var tmp = constant.BONE
// 		for _, _x := range x {
// 			tmp = number.Div(number.SafeMul(tmp, _x), D)
// 		}
// 		D = number.Div(number.SafeMul(D, number.SafeAdd(number.SafeMul(big.NewInt(int64(nCoins-1)), constant.BONE), tmp)), number.SafeMul(nCoinsBi, constant.BONE))
// 		if D.Cmp(D_prev) > 0 {
// 			diff = number.SafeSub(D, D_prev)
// 		} else {
// 			diff = number.SafeSub(D_prev, D)
// 		}
// 		if diff.Cmp(number.Number_1) <= 0 || number.SafeMul(diff, constant.BONE).Cmp(D) < 0 {
// 			return D, nil
// 		}
// 	}
// 	return nil, errors.New("did not converge")
// }

// func sqrt_int(x *big.Int) (*big.Int, error) {
// 	if x.Cmp(constant.ZeroBI) == 0 {
// 		return constant.ZeroBI, nil
// 	}
// 	var z = number.Div(number.SafeAdd(x, constant.BONE), number.Number_2)
// 	var y = x
// 	for i := 0; i < 256; i += 1 {
// 		if z.Cmp(y) == 0 {
// 			return y, nil
// 		}
// 		y = z
// 		z = number.Div(number.SafeAdd(number.Div(number.SafeMul(x, constant.BONE), z), z), number.Number_2)
// 	}
// 	return nil, errors.New("sqrt_int did not converge")
// }

// func newton_D(ANN *big.Int, gamma *big.Int, x_unsorted []*big.Int) (*big.Int, error) {
// 	// todo: MinA, MaxA
// 	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
// 		return nil, errors.New("unsafe values gamma")
// 	}
// 	var nCoins = len(x_unsorted)
// 	var nCoinsBi = big.NewInt(int64(nCoins))
// 	var x = sortArray(x_unsorted)
// 	if x[0].Cmp(constant.TenPowInt(9)) < 0 || x[0].Cmp(constant.TenPowInt(33)) > 0 {
// 		return nil, errors.New("unsafe values x[0]")
// 	}
// 	for i := 1; i < nCoins; i += 1 {
// 		var frac = number.Div(number.SafeMul(x[i], constant.BONE), x[0])
// 		if frac.Cmp(constant.TenPowInt(11)) < 0 {
// 			return nil, errors.New("unsafe values x[i]")
// 		}
// 	}
// 	var mean, err = _geometric_mean(x, false)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var D = number.SafeMul(nCoinsBi, mean)
// 	var S = constant.ZeroBI
// 	for _, x_i := range x {
// 		S = number.SafeAdd(S, x_i)
// 	}
// 	for i := 0; i < 255; i += 1 {
// 		var D_prev = D
// 		var K0 = constant.BONE
// 		for _, _x := range x {
// 			K0 = number.Div(number.SafeMul(number.SafeMul(K0, _x), nCoinsBi), D)
// 		}
// 		var _g1k0 = number.SafeAdd(gamma, constant.BONE)
// 		if _g1k0.Cmp(K0) > 0 {
// 			_g1k0 = number.SafeAdd(number.SafeSub(_g1k0, K0), number.Number_1)
// 		} else {
// 			_g1k0 = number.SafeAdd(number.SafeSub(K0, _g1k0), number.Number_1)
// 		}
// 		var mul1 = number.Div(number.SafeMul(
// 			number.SafeMul(
// 				number.Div(number.SafeMul(number.Div(number.SafeMul(constant.BONE, D), gamma), _g1k0), gamma),
// 				_g1k0),
// 			AMultiplier), ANN)
// 		var mul2 = number.Div(number.SafeMul(number.SafeMul(number.SafeMul(number.Number_2, constant.BONE), nCoinsBi), K0), _g1k0)
// 		var neg_fprime = number.SafeSub(
// 			number.SafeAdd(
// 				number.SafeAdd(S, number.Div(number.SafeMul(S, mul2), constant.BONE)),
// 				number.Div(number.SafeMul(mul1, nCoinsBi), K0),
// 			),
// 			number.Div(number.SafeMul(mul2, D), constant.BONE))
// 		var D_plus = number.Div(number.SafeMul(D, number.SafeAdd(neg_fprime, S)), neg_fprime)
// 		var D_minus = number.Div(number.SafeMul(D, D), neg_fprime)
// 		if constant.BONE.Cmp(K0) > 0 {
// 			D_minus = number.SafeAdd(D_minus,
// 				number.Div(
// 					number.SafeMul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), constant.BONE), number.SafeSub(constant.BONE, K0)),
// 					K0,
// 				),
// 			)
// 		} else {
// 			D_minus = number.SafeSub(D_minus,
// 				number.Div(
// 					number.SafeMul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), constant.BONE), number.SafeSub(K0, constant.BONE)),
// 					K0,
// 				),
// 			)
// 		}
// 		if D_plus.Cmp(D_minus) > 0 {
// 			D = number.SafeSub(D_plus, D_minus)
// 		} else {
// 			D = number.Div(number.SafeSub(D_minus, D_plus), number.Number_2)
// 		}
// 		var diff *big.Int
// 		if D.Cmp(D_prev) > 0 {
// 			diff = number.SafeSub(D, D_prev)
// 		} else {
// 			diff = number.SafeSub(D_prev, D)
// 		}
// 		var temp = constant.TenPowInt(16)
// 		if D.Cmp(temp) > 0 {
// 			temp = D
// 		}
// 		if number.SafeMul(diff, constant.TenPowInt(14)).Cmp(temp) < 0 {
// 			for _, _x := range x {
// 				var frac = number.Div(number.SafeMul(_x, constant.BONE), D)
// 				if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
// 					return nil, errors.New("unsafe values x[i]")
// 				}
// 			}
// 			return D, nil
// 		}
// 	}
// 	return nil, errors.New("did not converge")
// }

// contracts/main/CurveCryptoMathOptimized3.vy
func get_y(
	ann, gamma *uint256.Int, x []uint256.Int, D *uint256.Int, i int,
	//output
	y *uint256.Int,
) error {
	if ann.Cmp(MinA) < 0 || ann.Cmp(MaxA) > 0 {
		return errors.New("unsafe values A")
	}

	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
		return errors.New("unsafe values gamma")
	}

	if D.Cmp(MinD) < 0 || D.Cmp(MaxD) > 0 {
		return errors.New("unsafe values D")
	}

	for k := 0; k < 3; k++ {
		if k == i {
			continue
		}
		frac := number.Div(number.Mul(&x[k], Precision), D)
		if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
			return fmt.Errorf("unsafe values x[%d] %s", i, frac.Dec())
		}
	}

	// the new function `get_y` use a new method but still fallback to `newton_y` in some cases
	// also, the different between new and old method are around 2 wei which is not that much, so here we'll still use `newton_y`
	return newton_y(ann, gamma, x, D, i, y)
}

// Calculate x[i] given A, gamma, xp and D using newton's method.
func newton_y(
	ann, gamma *uint256.Int, x []uint256.Int, D *uint256.Int, i int,
	//output
	y *uint256.Int,
) error {
	y.Div(D, NumTokensU256)
	var K0i, Si uint256.Int
	K0i.Set(Precision)
	Si.Clear()

	var xSorted [NumTokens]uint256.Int
	for j := 0; j < NumTokens; j += 1 {
		xSorted[j].Set(&x[j])
	}
	xSorted[i].Clear()
	sort_inplace(xSorted[:])

	// convergence_limit: uint256 = max(max(x_sorted[0] / 10**14, D / 10**14), 100)
	var convergenceLimit = number.Div(&xSorted[0], TenPow14)
	var temp = number.Div(D, TenPow14)
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if uint256.NewInt(100).Cmp(convergenceLimit) > 0 {
		convergenceLimit.SetUint64(100)
	}

	for j := 2; j < NumTokens+1; j += 1 {
		var _x = &xSorted[NumTokens-j]
		if _x.IsZero() {
			return ErrZero
		}
		y.Div(number.SafeMul(y, D), number.SafeMul(_x, NumTokensU256))
		Si.Add(&Si, _x)
	}
	for j := 0; j < NumTokens-1; j += 1 {
		K0i.Div(number.SafeMul(number.SafeMul(&K0i, &xSorted[j]), NumTokensU256), D)
	}

	var yPrev, K0, S, _g1k0, mul1, yfprime uint256.Int
	De18 := number.SafeMul(D, Precision)

	for j := 0; j < 255; j += 1 {
		yPrev.Set(y)
		K0.Div(number.SafeMul(number.SafeMul(&K0i, y), NumTokensU256), D)
		S.Add(&Si, y)

		_g1k0.Add(gamma, Precision)
		if _g1k0.Cmp(&K0) > 0 {
			number.SafeAddZ(number.SafeSub(&_g1k0, &K0), number.Number_1, &_g1k0)
		} else {
			number.SafeAddZ(number.SafeSub(&K0, &_g1k0), number.Number_1, &_g1k0)
		}

		// mul1 = 10**18 * D / gamma * _g1k0 / gamma * _g1k0 * A_MULTIPLIER / ANN
		mul1.Div(
			number.SafeMul(
				number.Div(
					number.SafeMul(
						number.Div(De18, gamma),
						&_g1k0,
					), gamma,
				),
				number.SafeMul(&_g1k0, AMultiplier),
			), ann)

		// mul2 = 10**18 + (2 * 10**18) * K0 / _g1k0
		var mul2 = number.SafeAdd(
			Precision,
			number.Div(number.SafeMul(TwoE18, &K0), &_g1k0),
		)

		// yfprime = 10**18 * y + S * mul2 + mul1
		number.SafeAddZ(
			number.SafeAdd(number.SafeMul(Precision, y), number.SafeMul(&S, mul2)),
			&mul1, &yfprime)
		var _dyfprime = number.SafeMul(D, mul2)
		if yfprime.Cmp(_dyfprime) < 0 {
			y.Div(&yPrev, number.Number_2)
			continue
		} else {
			number.SafeSubZ(&yfprime, _dyfprime, &yfprime)
		}

		if y.IsZero() {
			return ErrZero
		}

		var fprime = number.Div(&yfprime, y)

		if fprime.IsZero() {
			return ErrZero
		}

		var yMinus = number.Div(&mul1, fprime)
		var yPlus = number.SafeAdd(number.Div(
			number.SafeAdd(&yfprime, De18),
			fprime),
			number.Div(number.SafeMul(yMinus, Precision), &K0))
		number.SafeAddZ(yMinus, number.Div(number.SafeMul(Precision, &S), fprime), yMinus)
		if yPlus.Cmp(yMinus) < 0 {
			y.Div(&yPrev, number.Number_2)
		} else {
			number.SafeSubZ(yPlus, yMinus, y)
		}
		var diff uint256.Int
		if y.Cmp(&yPrev) > 0 {
			diff.Sub(y, &yPrev)
		} else {
			diff.Sub(&yPrev, y)
		}
		var t = number.Div(y, TenPow14)
		if convergenceLimit.Cmp(t) > 0 {
			t = convergenceLimit
		}
		if diff.Cmp(t) < 0 {
			var frac = number.Div(number.SafeMul(y, Precision), D)
			if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
				return ErrUnsafeY
			}
			return nil
		}
	}
	return ErrYDoesNotConverge
}

func reductionCoefficient(x []uint256.Int, feeGamma *uint256.Int, K *uint256.Int) error {
	var S uint256.Int
	number.SafeAddZ(number.SafeAdd(&x[0], &x[1]), &x[2], &S)
	if S.IsZero() {
		return ErrZero
	}

	K.Div(number.SafeMul(number.SafeMul(Precision, NumTokensU256), &x[0]), &S)
	K.Div(number.SafeMul(number.SafeMul(K, NumTokensU256), &x[1]), &S)
	K.Div(number.SafeMul(number.SafeMul(K, NumTokensU256), &x[2]), &S)

	if !feeGamma.IsZero() {
		K.Div(
			number.SafeMul(feeGamma, Precision),
			number.SafeSub(number.SafeAdd(feeGamma, Precision), K))
	}
	return nil
}

// func halfpow(power *big.Int, precision *big.Int) (*big.Int, error) {
// 	var intpow = number.Div(power, constant.BONE)
// 	var otherpow = number.SafeSub(power, number.SafeMul(intpow, constant.BONE))
// 	if intpow.Cmp(big.NewInt(59)) > 0 {
// 		return constant.ZeroBI, nil
// 	}
// 	var result = number.Div(constant.BONE, new(big.Int).Exp(number.Number_2, intpow, nil))
// 	if otherpow.Cmp(constant.ZeroBI) == 0 {
// 		return result, nil
// 	}
// 	var term = constant.BONE
// 	var x = number.SafeMul(constant.Five, constant.TenPowInt(17))
// 	var S = constant.BONE
// 	var neg = false
// 	for i := 1; i < 256; i += 1 {
// 		var K = number.SafeMul(big.NewInt(int64(i)), constant.BONE)
// 		var c = number.SafeSub(K, constant.BONE)
// 		if otherpow.Cmp(c) > 0 {
// 			c = number.SafeSub(otherpow, c)
// 			neg = !neg
// 		} else {
// 			c = number.SafeSub(c, otherpow)
// 		}
// 		term = number.Div(number.SafeMul(term, number.Div(number.SafeMul(c, x), constant.BONE)), K)
// 		if neg {
// 			S = number.SafeSub(S, term)
// 		} else {
// 			S = number.SafeAdd(S, term)
// 		}
// 		if term.Cmp(precision) < 0 {
// 			return number.Div(number.SafeMul(result, S), constant.BONE), nil
// 		}
// 	}
// 	return nil, errors.New("did not converge")
// }

func (t *PoolSimulator) _A_gamma() (*uint256.Int, *uint256.Int) {
	var A, gamma uint256.Int
	t._A_gamma_inplace(&A, &gamma)
	return &A, &gamma
}

func (t *PoolSimulator) _A_gamma_inplace(A, gamma *uint256.Int) {
	var t1 = t.Extra.FutureAGammaTime
	A.Set(t.Extra.FutureA)
	gamma.Set(t.Extra.FutureGamma)
	var now = time.Now().Unix()
	if now < t1 {
		var A0 = t.Extra.InitialA
		var gamma0 = t.Extra.InitialGamma
		var t0 = t.Extra.InitialAGammaTime
		t1 -= t0
		t0 = now - t0
		var t2 = t1 - t0
		A.Div(number.Add(
			number.Mul(A0, uint256.NewInt(uint64(t2))),
			number.Mul(A, uint256.NewInt(uint64(t0))),
		), uint256.NewInt(uint64(t1)))

		gamma.Div(number.Add(
			number.Mul(gamma0, uint256.NewInt(uint64(t2))),
			number.Mul(gamma, uint256.NewInt(uint64(t0))),
		), uint256.NewInt(uint64(t1)))
	}
}

func (t *PoolSimulator) FeeCalc(xp []uint256.Int, fee *uint256.Int) error {
	var f uint256.Int
	var err = reductionCoefficient(xp, t.Extra.FeeGamma, &f)
	if err != nil {
		return err
	}
	fee.Div(
		number.SafeAdd(
			number.SafeMul(t.Extra.MidFee, &f),
			number.SafeMul(t.Extra.OutFee, number.SafeSub(Precision, &f))),
		Precision)
	return nil
}

func (t *PoolSimulator) GetDy(i int, j int, dx, dy, fee *uint256.Int) error {
	var xp [NumTokens]uint256.Int
	for k := 0; k < NumTokens; k += 1 {
		if k == i {
			number.SafeAddZ(&t.Reserves[k], dx, &xp[k])
			continue
		}
		xp[k].Set(&t.Reserves[k])
	}

	number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])
	for k := 0; k < 2; k += 1 {
		xp[k+1].Div(
			number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
			Precision,
		)
	}
	fmt.Println("--", xp[0].Dec(), xp[1].Dec(), xp[2].Dec())

	A, gamma := t._A_gamma()
	fmt.Println("A gamma", A.Dec(), gamma.Dec())
	var y uint256.Int
	var err = get_y(A, gamma, xp[:], t.Extra.D, j, &y)
	if err != nil {
		return err
	}
	fmt.Println("y", y.Dec())
	number.SafeSubZ(number.SafeSub(&xp[j], &y), number.Number_1, dy)
	xp[j] = y
	if j > 0 {
		dy.Div(number.SafeMul(dy, Precision), &t.Extra.PriceScale[j-1])
	}
	dy.Div(dy, &t.precisionMultipliers[j])

	err = t.FeeCalc(xp[:], fee)
	if err != nil {
		return err
	}
	fmt.Println("fee", fee.Dec())

	fee.Div(number.SafeMul(fee, dy), TenPow10)
	fmt.Println("fee1", fee.Dec())
	dy.Sub(dy, fee)

	return nil
}

func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	return nil, nil
	// var nCoins = len(t.Info.Tokens)
	// if i == j {
	// 	return nil, errors.New("i = j")
	// }
	// if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
	// 	return nil, errors.New("coin index out of range")
	// }
	// if dx.Cmp(constant.ZeroBI) <= 0 {
	// 	return nil, errors.New("do not exchange 0 coins")
	// }

	// var A_gamma = t._A_gamma()
	// var xp = make([]*big.Int, nCoins)
	// for k := 0; k < nCoins; k += 1 {
	// 	xp[k] = t.Info.Reserves[k]
	// }
	// var ix = j
	// var p = constant.ZeroBI
	// var dy = constant.ZeroBI

	// var y = xp[j]
	// var x0 = xp[i]
	// xp[i] = number.SafeAdd(x0, dx)
	// t.Info.Reserves[i] = new(big.Int).Set(xp[i])
	// var price_scale = make([]*big.Int, nCoins-1)
	// var packed_price = t.PriceScalePacked

	// for k := 0; k < nCoins-1; k += 1 {
	// 	price_scale[k] = new(big.Int).And(packed_price, PriceMask)
	// 	packed_price = new(big.Int).Rsh(packed_price, PriceSize)
	// }
	// xp[0] = number.SafeMul(xp[0], t.precisionMultipliers[0])
	// for k := 1; k < nCoins; k += 1 {
	// 	xp[k] = number.Div(number.SafeMul(number.SafeMul(xp[k], price_scale[k-1]), t.precisionMultipliers[k]), Precision)
	// }
	// {
	// 	var ti = t.FutureAGammaTime
	// 	if ti > 0 {
	// 		x0 = number.SafeMul(x0, t.precisionMultipliers[i])
	// 		if i > 0 {
	// 			x0 = number.Div(number.SafeMul(x0, price_scale[i-1]), Precision)
	// 		}
	// 		var x1 = xp[i]
	// 		xp[i] = x0
	// 		var temp, err = newton_D(A_gamma[0], A_gamma[1], xp)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		t.D = temp
	// 		xp[i] = x1
	// 		if time.Now().Unix() >= ti {
	// 			t.FutureAGammaTime = 1
	// 		}
	// 	}
	// }
	// var temp, err = newtonY(A_gamma[0], A_gamma[1], xp, t.D, j)
	// if err != nil {
	// 	return nil, err
	// }
	// dy = number.SafeSub(xp[j], temp)
	// xp[j] = number.SafeSub(xp[j], dy)
	// dy = number.SafeSub(dy, number.Number_1)
	// if j > 0 {
	// 	dy = number.Div(number.SafeMul(dy, Precision), price_scale[j-1])
	// }
	// dy = number.Div(dy, t.precisionMultipliers[j])
	// dy = number.SafeSub(dy, number.Div(number.SafeMul(t.FeeCalc(xp), dy), constant.TenPowInt(10)))
	// //assert dy >= min_dy, "Slippage"
	// y = number.SafeSub(y, dy)
	// t.Info.Reserves[j] = y
	// y = number.SafeMul(y, t.precisionMultipliers[j])
	// if j > 0 {
	// 	y = number.Div(number.SafeMul(y, price_scale[j-1]), Precision)
	// }
	// xp[j] = y
	// if dx.Cmp(constant.TenPowInt(5)) > 0 && dy.Cmp(constant.TenPowInt(5)) > 0 {
	// 	var _dx = number.SafeMul(dx, t.precisionMultipliers[i])
	// 	var _dy = number.SafeMul(dy, t.precisionMultipliers[j])
	// 	if i != 0 && j != 0 {
	// 		p = number.Div(number.SafeMul(new(big.Int).And(new(big.Int).Rsh(t.LastPricesPacked, PriceSize*uint(i-1)), PriceMask), _dx), _dy)
	// 	} else if i == 0 {
	// 		p = number.Div(number.SafeMul(_dx, constant.BONE), _dy)
	// 	} else {
	// 		p = number.Div(number.SafeMul(_dy, constant.BONE), _dx)
	// 		ix = i
	// 	}
	// }
	// err = t.tweak_price(A_gamma, xp, ix, p, constant.ZeroBI)
	// return dy, err
}

// func (t *PoolSimulator) tweak_price(A_gamma []*big.Int, _xp []*big.Int, i int, p_i *big.Int, new_D *big.Int) error {
// 	var nCoins = len(_xp)
// 	var nCoinsBi = big.NewInt(int64(nCoins))
// 	var price_oracle = make([]*big.Int, nCoins-1)
// 	var last_prices = make([]*big.Int, nCoins-1)
// 	var price_scale = make([]*big.Int, nCoins-1)
// 	var xp = make([]*big.Int, nCoins)
// 	var p_new = make([]*big.Int, nCoins-1)

// 	var packed_prices = t.PriceOraclePacked
// 	for k := 0; k < nCoins-1; k += 1 {
// 		price_oracle[k] = new(big.Int).And(packed_prices, PriceMask)
// 		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
// 	}
// 	var last_prices_timestamp = t.LastPricesTimestamp
// 	packed_prices = t.LastPricesPacked
// 	for k := 0; k < nCoins-1; k += 1 {
// 		last_prices[k] = new(big.Int).And(packed_prices, PriceMask)
// 		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
// 	}
// 	var blockTimestamp = time.Now().Unix()
// 	if last_prices_timestamp < blockTimestamp {
// 		var ma_half_time = t.MaHalfTime
// 		var alpha, _ = halfpow(
// 			number.Div(number.SafeMul(big.NewInt(blockTimestamp-last_prices_timestamp), constant.BONE), ma_half_time),
// 			constant.TenPowInt(10),
// 		)
// 		packed_prices = constant.ZeroBI
// 		for k := 0; k < nCoins-1; k += 1 {
// 			price_oracle[k] = number.Div(
// 				number.SafeAdd(number.SafeMul(last_prices[k], number.SafeSub(constant.BONE, alpha)), number.SafeMul(price_oracle[k], alpha)),
// 				constant.BONE,
// 			)
// 		}
// 		for k := 0; k < nCoins-1; k += 1 {
// 			packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
// 			var p = price_oracle[nCoins-2-k]
// 			packed_prices = new(big.Int).Or(p, packed_prices)
// 		}
// 		t.PriceOraclePacked = packed_prices
// 		t.LastPricesTimestamp = blockTimestamp
// 	}
// 	var D_unadjusted = new_D
// 	if new_D.Cmp(constant.ZeroBI) == 0 {
// 		D_unadjusted, _ = newton_D(A_gamma[0], A_gamma[1], _xp)
// 	}
// 	packed_prices = t.PriceScalePacked
// 	for k := 0; k < nCoins-1; k += 1 {
// 		price_scale[k] = new(big.Int).And(packed_prices, PriceMask)
// 		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
// 	}
// 	if p_i.Cmp(constant.ZeroBI) > 0 {
// 		if i > 0 {
// 			last_prices[i-1] = p_i
// 		} else {
// 			for k := 0; k < nCoins-1; k += 1 {
// 				last_prices[k] = number.Div(number.SafeMul(last_prices[k], constant.BONE), p_i)
// 			}
// 		}
// 	} else {
// 		var __xp = make([]*big.Int, nCoins)
// 		for k := 0; k < nCoins; k += 1 {
// 			__xp[k] = new(big.Int).Set(_xp[k])
// 		}
// 		var dx_price = number.Div(__xp[0], constant.TenPowInt(6))
// 		__xp[0] = number.SafeAdd(__xp[0], dx_price)
// 		for k := 0; k < nCoins-1; k += 1 {
// 			var temp, err = newtonY(A_gamma[0], A_gamma[1], __xp, D_unadjusted, k+1)
// 			if err != nil {
// 				return err
// 			}
// 			last_prices[k] = number.Div(number.SafeMul(price_scale[k], dx_price), number.SafeSub(_xp[k+1], temp))
// 		}
// 	}
// 	packed_prices = constant.ZeroBI
// 	for k := 0; k < nCoins-1; k += 1 {
// 		packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
// 		var p = last_prices[nCoins-2-k]
// 		packed_prices = new(big.Int).Or(p, packed_prices)
// 	}
// 	t.LastPricesPacked = packed_prices

// 	var total_supply = t.LpSupply
// 	var old_xcp_profit = t.XcpProfit
// 	var old_virtual_price = t.VirtualPrice

// 	xp[0] = number.Div(D_unadjusted, nCoinsBi)
// 	for k := 0; k < nCoins-1; k += 1 {
// 		xp[k+1] = number.Div(number.SafeMul(D_unadjusted, constant.BONE), number.SafeMul(nCoinsBi, price_scale[k]))
// 	}
// 	var xcp_profit = constant.BONE
// 	var virtual_price = constant.BONE
// 	if old_virtual_price.Cmp(constant.ZeroBI) > 0 {
// 		var xcp, err = _geometric_mean(xp, true)
// 		if err != nil {
// 			return err
// 		}
// 		virtual_price = number.Div(number.SafeMul(constant.BONE, xcp), total_supply)
// 		xcp_profit = number.Div(number.SafeMul(old_xcp_profit, virtual_price), old_virtual_price)
// 		var aGammaTime = t.FutureAGammaTime
// 		if virtual_price.Cmp(old_virtual_price) < 0 && aGammaTime == 0 {
// 			return errors.New("loss")
// 		}
// 		if aGammaTime == 1 {
// 			t.FutureAGammaTime = 0
// 		}
// 	}
// 	t.XcpProfit = xcp_profit
// 	var needs_adjustment = t.NotAdjusted
// 	if number.SafeSub(number.SafeMul(virtual_price, number.Number_2), constant.BONE).Cmp(
// 		number.SafeAdd(xcp_profit, number.SafeMul(number.Number_2, t.AllowedExtraProfit))) > 0 {
// 		needs_adjustment = true
// 		t.NotAdjusted = true
// 	}
// 	if needs_adjustment {
// 		var adjustment_step = t.AdjustmentStep
// 		var norm = constant.ZeroBI
// 		for k := 0; k < nCoins-1; k += 1 {
// 			var ratio = number.Div(number.SafeMul(price_oracle[k], constant.BONE), price_scale[k])
// 			if ratio.Cmp(constant.BONE) > 0 {
// 				ratio = number.SafeSub(ratio, constant.BONE)
// 			} else {
// 				ratio = number.SafeSub(constant.BONE, ratio)
// 			}
// 			norm = number.SafeAdd(norm, number.SafeMul(ratio, ratio))
// 		}
// 		if norm.Cmp(number.SafeMul(adjustment_step, adjustment_step)) > 0 && old_virtual_price.Cmp(constant.ZeroBI) > 0 {
// 			var temp, err = sqrt_int(number.Div(norm, constant.BONE))
// 			if err != nil {
// 				return err
// 			}
// 			norm = temp
// 			for k := 0; k < nCoins-1; k += 1 {
// 				p_new[k] = number.Div(
// 					number.SafeAdd(
// 						number.SafeMul(price_scale[k], number.SafeSub(norm, adjustment_step)),
// 						number.SafeMul(adjustment_step, price_oracle[k]),
// 					), norm)
// 			}
// 			for k := 0; k < nCoins; k += 1 {
// 				xp[k] = new(big.Int).Set(_xp[k])
// 			}
// 			for k := 0; k < nCoins-1; k += 1 {
// 				xp[k+1] = number.Div(number.SafeMul(_xp[k+1], p_new[k]), price_scale[k])
// 			}
// 			D, err := newton_D(A_gamma[0], A_gamma[1], xp)
// 			if err != nil {
// 				return err
// 			}

// 			xp[0] = number.Div(D, nCoinsBi)
// 			for k := 0; k < nCoins-1; k += 1 {
// 				xp[k+1] = number.Div(number.SafeMul(D, constant.BONE), number.SafeMul(nCoinsBi, p_new[k]))
// 			}
// 			temp, err = _geometric_mean(xp, true)
// 			if err != nil {
// 				return err
// 			}
// 			old_virtual_price = number.Div(number.SafeMul(constant.BONE, temp), total_supply)
// 			if old_virtual_price.Cmp(constant.BONE) > 0 && number.SafeSub(number.SafeMul(number.Number_2, old_virtual_price), constant.BONE).Cmp(xcp_profit) > 0 {
// 				packed_prices = constant.ZeroBI
// 				for k := 0; k < nCoins-1; k += 1 {
// 					packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
// 					packed_prices = new(big.Int).Or(p_new[nCoins-2-k], packed_prices)
// 				}
// 				t.PriceScalePacked = packed_prices
// 				t.D = D
// 				t.VirtualPrice = old_virtual_price
// 				return nil
// 			} else {
// 				t.NotAdjusted = false
// 			}
// 		}
// 	}
// 	t.D = D_unadjusted
// 	t.VirtualPrice = virtual_price
// 	return nil
// }

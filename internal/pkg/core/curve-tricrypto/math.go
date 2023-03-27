package curveTricrypto

import (
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
)

func sortArray(A0 []*big.Int) []*big.Int {
	var nCoins = len(A0)
	var ret = make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		ret[i] = A0[i]
	}
	for i := 1; i < nCoins; i += 1 {
		var x = ret[i]
		var cur = i
		for j := 0; j < nCoins; j += 1 {
			var y = ret[cur-1]
			if y.Cmp(x) > 0 {
				break
			}
			ret[cur] = y
			cur -= 1
			if cur == 0 {
				break
			}
		}
		ret[cur] = x
	}
	return ret
}

func _geometric_mean(unsorted_x []*big.Int, sort bool) (*big.Int, error) {
	var nCoins = len(unsorted_x)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = unsorted_x
	if sort {
		x = sortArray(unsorted_x)
	}
	var D = x[0]
	var diff = constant.Zero
	for i := 0; i < 255; i += 1 {
		var D_prev = D
		var tmp = constant.BONE
		for _, _x := range x {
			tmp = new(big.Int).Div(new(big.Int).Mul(tmp, _x), D)
		}
		D = new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Add(new(big.Int).Mul(big.NewInt(int64(nCoins-1)), constant.BONE), tmp)), new(big.Int).Mul(nCoinsBi, constant.BONE))
		if D.Cmp(D_prev) > 0 {
			diff = new(big.Int).Sub(D, D_prev)
		} else {
			diff = new(big.Int).Sub(D_prev, D)
		}
		if diff.Cmp(constant.One) <= 0 || new(big.Int).Mul(diff, constant.BONE).Cmp(D) < 0 {
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func sqrt_int(x *big.Int) (*big.Int, error) {
	if x.Cmp(constant.Zero) == 0 {
		return constant.Zero, nil
	}
	var z = new(big.Int).Div(new(big.Int).Add(x, constant.BONE), constant.Two)
	var y = x
	for i := 0; i < 256; i += 1 {
		if z.Cmp(y) == 0 {
			return y, nil
		}
		y = z
		z = new(big.Int).Div(new(big.Int).Add(new(big.Int).Div(new(big.Int).Mul(x, constant.BONE), z), z), constant.Two)
	}
	return nil, errors.New("sqrt_int did not converge")
}

func newton_D(ANN *big.Int, gamma *big.Int, x_unsorted []*big.Int) (*big.Int, error) {
	// todo: MinA, MaxA
	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
		return nil, errors.New("unsafe values gamma")
	}
	var nCoins = len(x_unsorted)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = sortArray(x_unsorted)
	if x[0].Cmp(constant.TenPowInt(9)) < 0 || x[0].Cmp(constant.TenPowInt(33)) > 0 {
		return nil, errors.New("unsafe values x[0]")
	}
	for i := 1; i < nCoins; i += 1 {
		var frac = new(big.Int).Div(new(big.Int).Mul(x[i], constant.BONE), x[0])
		if frac.Cmp(constant.TenPowInt(11)) < 0 {
			return nil, errors.New("unsafe values x[i]")
		}
	}
	var mean, err = _geometric_mean(x, false)
	if err != nil {
		return nil, err
	}
	var D = new(big.Int).Mul(nCoinsBi, mean)
	var S = constant.Zero
	for _, x_i := range x {
		S = new(big.Int).Add(S, x_i)
	}
	for i := 0; i < 255; i += 1 {
		var D_prev = D
		var K0 = constant.BONE
		for _, _x := range x {
			K0 = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K0, _x), nCoinsBi), D)
		}
		var _g1k0 = new(big.Int).Add(gamma, constant.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(_g1k0, K0), constant.One)
		} else {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, _g1k0), constant.One)
		}
		var mul1 = new(big.Int).Div(new(big.Int).Mul(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(new(big.Int).Mul(constant.BONE, D), gamma), _g1k0), gamma),
				_g1k0),
			AMultiplier), ANN)
		var mul2 = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(new(big.Int).Mul(constant.Two, constant.BONE), nCoinsBi), K0), _g1k0)
		var neg_fprime = new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Add(S, new(big.Int).Div(new(big.Int).Mul(S, mul2), constant.BONE)),
				new(big.Int).Div(new(big.Int).Mul(mul1, nCoinsBi), K0),
			),
			new(big.Int).Div(new(big.Int).Mul(mul2, D), constant.BONE))
		var D_plus = new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Add(neg_fprime, S)), neg_fprime)
		var D_minus = new(big.Int).Div(new(big.Int).Mul(D, D), neg_fprime)
		if constant.BONE.Cmp(K0) > 0 {
			D_minus = new(big.Int).Add(D_minus,
				new(big.Int).Div(
					new(big.Int).Mul(new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Div(mul1, neg_fprime)), constant.BONE), new(big.Int).Sub(constant.BONE, K0)),
					K0,
				),
			)
		} else {
			D_minus = new(big.Int).Sub(D_minus,
				new(big.Int).Div(
					new(big.Int).Mul(new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Div(mul1, neg_fprime)), constant.BONE), new(big.Int).Sub(K0, constant.BONE)),
					K0,
				),
			)
		}
		if D_plus.Cmp(D_minus) > 0 {
			D = new(big.Int).Sub(D_plus, D_minus)
		} else {
			D = new(big.Int).Div(new(big.Int).Sub(D_minus, D_plus), constant.Two)
		}
		var diff = constant.Zero
		if D.Cmp(D_prev) > 0 {
			diff = new(big.Int).Sub(D, D_prev)
		} else {
			diff = new(big.Int).Sub(D_prev, D)
		}
		var temp = constant.TenPowInt(16)
		if D.Cmp(temp) > 0 {
			temp = D
		}
		if new(big.Int).Mul(diff, constant.TenPowInt(14)).Cmp(temp) < 0 {
			for _, _x := range x {
				var frac = new(big.Int).Div(new(big.Int).Mul(_x, constant.BONE), D)
				if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
					return nil, errors.New("unsafe values x[i]")
				}
			}
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonY(ann *big.Int, gamma *big.Int, x []*big.Int, D *big.Int, i int) (*big.Int, error) {
	var nCoins = len(x)
	var nCoinBi = big.NewInt(int64(nCoins))
	var y = new(big.Int).Div(D, nCoinBi)
	var K0i = constant.BONE
	var Si = constant.Zero

	var xSorted = make([]*big.Int, nCoins)
	for j := 0; j < nCoins; j += 1 {
		xSorted[j] = x[j]
	}
	xSorted[i] = constant.Zero
	xSorted = sortArray(xSorted)
	var tenPow14 = constant.TenPowInt(14)
	var convergenceLimit = new(big.Int).Div(xSorted[0], tenPow14)
	var temp = new(big.Int).Div(D, tenPow14)
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if big.NewInt(100).Cmp(convergenceLimit) > 0 {
		convergenceLimit = big.NewInt(100)
	}

	for j := 2; j < nCoins+1; j += 1 {
		var _x = xSorted[nCoins-j]
		y = new(big.Int).Div(new(big.Int).Mul(y, D), new(big.Int).Mul(_x, nCoinBi))
		Si = new(big.Int).Add(Si, _x)
	}
	for j := 0; j < nCoins-1; j += 1 {
		K0i = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K0i, xSorted[j]), nCoinBi), D)
	}
	for j := 0; j < 255; j += 1 {
		var yPrev = y
		var K0 = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K0i, y), nCoinBi), D)
		var S = new(big.Int).Add(Si, y)
		var _g1k0 = new(big.Int).Add(gamma, constant.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(_g1k0, K0), constant.One)
		} else {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, _g1k0), constant.One)
		}
		var mul1 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(new(big.Int).Mul(constant.BONE, D), gamma),
						_g1k0,
					), gamma,
				),
				new(big.Int).Mul(_g1k0, AMultiplier),
			), ann)
		var mul2 = new(big.Int).Add(new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(constant.Two, constant.BONE), K0), _g1k0), constant.BONE)
		var yfprime = new(big.Int).Add(new(big.Int).Add(new(big.Int).Mul(constant.BONE, y), new(big.Int).Mul(S, mul2)), mul1)
		var _dyfprime = new(big.Int).Mul(D, mul2)
		if yfprime.Cmp(_dyfprime) < 0 {
			y = new(big.Int).Div(yPrev, constant.Two)
			continue
		} else {
			yfprime = new(big.Int).Sub(yfprime, _dyfprime)
		}

		if y.Cmp(constant.Zero) == 0 {
			return nil, ErrDenominatorZero
		}

		var fprime = new(big.Int).Div(yfprime, y)

		if fprime.Cmp(constant.Zero) == 0 {
			return nil, ErrDenominatorZero
		}

		var yMinus = new(big.Int).Div(mul1, fprime)
		var yPlus = new(big.Int).Add(new(big.Int).Div(
			new(big.Int).Add(yfprime, new(big.Int).Mul(constant.BONE, D)),
			fprime),
			new(big.Int).Div(new(big.Int).Mul(yMinus, constant.BONE), K0))
		yMinus = new(big.Int).Add(yMinus, new(big.Int).Div(new(big.Int).Mul(constant.BONE, S), fprime))
		if yPlus.Cmp(yMinus) < 0 {
			y = new(big.Int).Div(yPrev, constant.Two)
		} else {
			y = new(big.Int).Sub(yPlus, yMinus)
		}
		var diff = constant.Zero
		if y.Cmp(yPrev) > 0 {
			diff = new(big.Int).Sub(y, yPrev)
		} else {
			diff = new(big.Int).Sub(yPrev, y)
		}
		var t = new(big.Int).Div(y, tenPow14)
		if convergenceLimit.Cmp(t) > 0 {
			t = convergenceLimit
		}
		if diff.Cmp(t) < 0 {
			var frac = new(big.Int).Div(new(big.Int).Mul(y, constant.BONE), D)
			if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
				return nil, errors.New("unsafe value for y")
			}
			return y, nil
		}
	}
	return nil, errors.New("did not converge")
}

func reductionCoefficient(x []*big.Int, feeGamma *big.Int) *big.Int {
	var nCoinsBi = big.NewInt(int64(len(x)))
	var K = constant.BONE
	var S = constant.Zero
	for _, xi := range x {
		S = new(big.Int).Add(S, xi)
	}
	for _, xi := range x {
		K = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K, nCoinsBi), xi), S)
	}
	if feeGamma.Cmp(constant.Zero) > 0 {
		K = new(big.Int).Div(new(big.Int).Mul(feeGamma, constant.BONE), new(big.Int).Sub(new(big.Int).Add(feeGamma, constant.BONE), K))
	}
	return K
}

func halfpow(power *big.Int, precision *big.Int) (*big.Int, error) {
	var intpow = new(big.Int).Div(power, constant.BONE)
	var otherpow = new(big.Int).Sub(power, new(big.Int).Mul(intpow, constant.BONE))
	if intpow.Cmp(big.NewInt(59)) > 0 {
		return constant.Zero, nil
	}
	var result = new(big.Int).Div(constant.BONE, new(big.Int).Exp(constant.Two, intpow, nil))
	if otherpow.Cmp(constant.Zero) == 0 {
		return result, nil
	}
	var term = constant.BONE
	var x = new(big.Int).Mul(constant.Five, constant.TenPowInt(17))
	var S = constant.BONE
	var neg = false
	for i := 1; i < 256; i += 1 {
		var K = new(big.Int).Mul(big.NewInt(int64(i)), constant.BONE)
		var c = new(big.Int).Sub(K, constant.BONE)
		if otherpow.Cmp(c) > 0 {
			c = new(big.Int).Sub(otherpow, c)
			neg = !neg
		} else {
			c = new(big.Int).Sub(c, otherpow)
		}
		term = new(big.Int).Div(new(big.Int).Mul(term, new(big.Int).Div(new(big.Int).Mul(c, x), constant.BONE)), K)
		if neg {
			S = new(big.Int).Sub(S, term)
		} else {
			S = new(big.Int).Add(S, term)
		}
		if term.Cmp(precision) < 0 {
			return new(big.Int).Div(new(big.Int).Mul(result, S), constant.BONE), nil
		}
	}
	return nil, errors.New("did not converge")
}

func (t *Pool) _packed_view(k uint, p *big.Int) *big.Int {
	var ret = new(big.Int).Rsh(p, k*128)
	return new(big.Int).And(ret, PriceMask)
}

func (t *Pool) price_scale(k uint) *big.Int {
	return t._packed_view(k, t.PriceScalePacked)
}

//func (t *Pool) price_oracle(k uint) *big.Int {
//	return t._packed_view(k, t.PriceOraclePacked)
//}

//func (t *Pool) last_prices(k uint) *big.Int {
//	return t._packed_view(k, t.LastPricesPacked)
//}

func (t *Pool) _A_gamma() []*big.Int {
	var t1 = t.FutureAGammaTime
	var A_gamma_1 = t.FutureAGamma
	var gamma1 = new(big.Int).And(A_gamma_1, PriceMask)
	var A1 = new(big.Int).Rsh(A_gamma_1, 128)
	var now = time.Now().Unix()
	if now < t1 {
		var A_gamma_0 = t.InitialAGamma
		var t0 = t.InitialAGammaTime
		t1 -= t0
		t0 = now - t0
		var t2 = t1 - t0
		A1 = new(big.Int).Div(new(big.Int).Add(
			new(big.Int).Mul(
				new(big.Int).Rsh(A_gamma_0, 128),
				big.NewInt(t2),
			),
			new(big.Int).Mul(
				A1,
				big.NewInt(t0),
			),
		), big.NewInt(t1))
		gamma1 = new(big.Int).Div(new(big.Int).Add(
			new(big.Int).Mul(new(big.Int).And(A_gamma_0, PriceMask), big.NewInt(t2)),
			new(big.Int).Mul(gamma1, big.NewInt(t0)),
		), big.NewInt(t1))
	}
	return []*big.Int{
		A1, gamma1,
	}
}

func (t *Pool) FeeCalc(xp []*big.Int) *big.Int {
	var f = reductionCoefficient(xp, t.FeeGamma)
	var ret = new(big.Int).Div(new(big.Int).Add(new(big.Int).Mul(t.MidFee, f), new(big.Int).Mul(t.OutFee, new(big.Int).Sub(constant.BONE, f))), constant.BONE)
	return ret
}

func (t *Pool) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	var xp = make([]*big.Int, 3)
	var price_scale = make([]*big.Int, 2)
	for k := 0; k < 2; k += 1 {
		price_scale[k] = t.price_scale(uint(k))
	}
	for k := 0; k < 3; k += 1 {
		xp[k] = t.Pool.Info.Reserves[k]
	}
	xp[i] = new(big.Int).Add(xp[i], dx)
	xp[0] = new(big.Int).Mul(xp[0], t.Precisions[0])

	for k := 0; k < 2; k += 1 {
		xp[k+1] = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(xp[k+1], price_scale[k]), t.Precisions[k+1]), Precision)
	}
	var y, err = newtonY(t.A, t.Gamma, xp, t.D, j)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[j], y), constant.One)
	xp[j] = y
	if j > 0 {
		dy = new(big.Int).Div(new(big.Int).Mul(dy, Precision), t.price_scale(uint(j-1)))
	}
	dy = new(big.Int).Div(dy, t.Precisions[j])
	var feeCalc = t.FeeCalc(xp)
	var fee = new(big.Int).Div(new(big.Int).Mul(feeCalc, dy), constant.TenPowInt(10))
	dy = new(big.Int).Sub(dy, fee)
	return dy, fee, nil
}

func (t *Pool) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	var nCoins = len(t.Info.Tokens)
	if i == j {
		return nil, errors.New("i = j")
	}
	if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
		return nil, errors.New("coin index out of range")
	}
	if dx.Cmp(constant.Zero) <= 0 {
		return nil, errors.New("do not exchange 0 coins")
	}

	var A_gamma = t._A_gamma()
	var xp = make([]*big.Int, nCoins)
	for k := 0; k < nCoins; k += 1 {
		xp[k] = t.Info.Reserves[k]
	}
	var ix = j
	var p = constant.Zero
	var dy = constant.Zero

	var y = xp[j]
	var x0 = xp[i]
	xp[i] = new(big.Int).Add(x0, dx)
	t.Info.Reserves[i] = new(big.Int).Set(xp[i])
	var price_scale = make([]*big.Int, nCoins-1)
	var packed_price = t.PriceScalePacked

	for k := 0; k < nCoins-1; k += 1 {
		price_scale[k] = new(big.Int).And(packed_price, PriceMask)
		packed_price = new(big.Int).Rsh(packed_price, PriceSize)
	}
	xp[0] = new(big.Int).Mul(xp[0], t.Precisions[0])
	for k := 1; k < nCoins; k += 1 {
		xp[k] = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(xp[k], price_scale[k-1]), t.Precisions[k]), Precision)
	}
	{
		var ti = t.FutureAGammaTime
		if ti > 0 {
			x0 = new(big.Int).Mul(x0, t.Precisions[i])
			if i > 0 {
				x0 = new(big.Int).Div(new(big.Int).Mul(x0, price_scale[i-1]), Precision)
			}
			var x1 = xp[i]
			xp[i] = x0
			var temp, err = newton_D(A_gamma[0], A_gamma[1], xp)
			if err != nil {
				return nil, err
			}
			t.D = temp
			xp[i] = x1
			if time.Now().Unix() >= ti {
				t.FutureAGammaTime = 1
			}
		}
	}
	var temp, err = newtonY(A_gamma[0], A_gamma[1], xp, t.D, j)
	if err != nil {
		return nil, err
	}
	dy = new(big.Int).Sub(xp[j], temp)
	xp[j] = new(big.Int).Sub(xp[j], dy)
	dy = new(big.Int).Sub(dy, constant.One)
	if j > 0 {
		dy = new(big.Int).Div(new(big.Int).Mul(dy, Precision), price_scale[j-1])
	}
	dy = new(big.Int).Div(dy, t.Precisions[j])
	dy = new(big.Int).Sub(dy, new(big.Int).Div(new(big.Int).Mul(t.FeeCalc(xp), dy), constant.TenPowInt(10)))
	//assert dy >= min_dy, "Slippage"
	y = new(big.Int).Sub(y, dy)
	t.Info.Reserves[j] = y
	y = new(big.Int).Mul(y, t.Precisions[j])
	if j > 0 {
		y = new(big.Int).Div(new(big.Int).Mul(y, price_scale[j-1]), Precision)
	}
	xp[j] = y
	if dx.Cmp(constant.TenPowInt(5)) > 0 && dy.Cmp(constant.TenPowInt(5)) > 0 {
		var _dx = new(big.Int).Mul(dx, t.Precisions[i])
		var _dy = new(big.Int).Mul(dy, t.Precisions[j])
		if i != 0 && j != 0 {
			p = new(big.Int).Div(new(big.Int).Mul(new(big.Int).And(new(big.Int).Rsh(t.LastPricesPacked, PriceSize*uint(i-1)), PriceMask), _dx), _dy)
		} else if i == 0 {
			p = new(big.Int).Div(new(big.Int).Mul(_dx, constant.BONE), _dy)
		} else {
			p = new(big.Int).Div(new(big.Int).Mul(_dy, constant.BONE), _dx)
			ix = i
		}
	}
	err = t.tweak_price(A_gamma, xp, ix, p, constant.Zero)
	return dy, err
}

func (t *Pool) tweak_price(A_gamma []*big.Int, _xp []*big.Int, i int, p_i *big.Int, new_D *big.Int) error {
	var nCoins = len(_xp)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var price_oracle = make([]*big.Int, nCoins-1)
	var last_prices = make([]*big.Int, nCoins-1)
	var price_scale = make([]*big.Int, nCoins-1)
	var xp = make([]*big.Int, nCoins)
	var p_new = make([]*big.Int, nCoins-1)

	var packed_prices = t.PriceOraclePacked
	for k := 0; k < nCoins-1; k += 1 {
		price_oracle[k] = new(big.Int).And(packed_prices, PriceMask)
		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
	}
	var last_prices_timestamp = t.LastPricesTimestamp
	packed_prices = t.LastPricesPacked
	for k := 0; k < nCoins-1; k += 1 {
		last_prices[k] = new(big.Int).And(packed_prices, PriceMask)
		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
	}
	var blockTimestamp = time.Now().Unix()
	if last_prices_timestamp < blockTimestamp {
		var ma_half_time = t.MaHalfTime
		var alpha, _ = halfpow(
			new(big.Int).Div(new(big.Int).Mul(big.NewInt(blockTimestamp-last_prices_timestamp), constant.BONE), ma_half_time),
			constant.TenPowInt(10),
		)
		packed_prices = constant.Zero
		for k := 0; k < nCoins-1; k += 1 {
			price_oracle[k] = new(big.Int).Div(
				new(big.Int).Add(new(big.Int).Mul(last_prices[k], new(big.Int).Sub(constant.BONE, alpha)), new(big.Int).Mul(price_oracle[k], alpha)),
				constant.BONE,
			)
		}
		for k := 0; k < nCoins-1; k += 1 {
			packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
			var p = price_oracle[nCoins-2-k]
			packed_prices = new(big.Int).Or(p, packed_prices)
		}
		t.PriceOraclePacked = packed_prices
		t.LastPricesTimestamp = blockTimestamp
	}
	var D_unadjusted = new_D
	if new_D.Cmp(constant.Zero) == 0 {
		D_unadjusted, _ = newton_D(A_gamma[0], A_gamma[1], _xp)
	}
	packed_prices = t.PriceScalePacked
	for k := 0; k < nCoins-1; k += 1 {
		price_scale[k] = new(big.Int).And(packed_prices, PriceMask)
		packed_prices = new(big.Int).Rsh(packed_prices, PriceSize)
	}
	if p_i.Cmp(constant.Zero) > 0 {
		if i > 0 {
			last_prices[i-1] = p_i
		} else {
			for k := 0; k < nCoins-1; k += 1 {
				last_prices[k] = new(big.Int).Div(new(big.Int).Mul(last_prices[k], constant.BONE), p_i)
			}
		}
	} else {
		var __xp = make([]*big.Int, nCoins)
		for k := 0; k < nCoins; k += 1 {
			__xp[k] = new(big.Int).Set(_xp[k])
		}
		var dx_price = new(big.Int).Div(__xp[0], constant.TenPowInt(6))
		__xp[0] = new(big.Int).Add(__xp[0], dx_price)
		for k := 0; k < nCoins-1; k += 1 {
			var temp, err = newtonY(A_gamma[0], A_gamma[1], __xp, D_unadjusted, k+1)
			if err != nil {
				return err
			}
			last_prices[k] = new(big.Int).Div(new(big.Int).Mul(price_scale[k], dx_price), new(big.Int).Sub(_xp[k+1], temp))
		}
	}
	packed_prices = constant.Zero
	for k := 0; k < nCoins-1; k += 1 {
		packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
		var p = last_prices[nCoins-2-k]
		packed_prices = new(big.Int).Or(p, packed_prices)
	}
	t.LastPricesPacked = packed_prices

	var total_supply = t.LpSupply
	var old_xcp_profit = t.XcpProfit
	var old_virtual_price = t.VirtualPrice

	xp[0] = new(big.Int).Div(D_unadjusted, nCoinsBi)
	for k := 0; k < nCoins-1; k += 1 {
		xp[k+1] = new(big.Int).Div(new(big.Int).Mul(D_unadjusted, constant.BONE), new(big.Int).Mul(nCoinsBi, price_scale[k]))
	}
	var xcp_profit = constant.BONE
	var virtual_price = constant.BONE
	if old_virtual_price.Cmp(constant.Zero) > 0 {
		var xcp, err = _geometric_mean(xp, true)
		if err != nil {
			return err
		}
		virtual_price = new(big.Int).Div(new(big.Int).Mul(constant.BONE, xcp), total_supply)
		xcp_profit = new(big.Int).Div(new(big.Int).Mul(old_xcp_profit, virtual_price), old_virtual_price)
		var aGammaTime = t.FutureAGammaTime
		if virtual_price.Cmp(old_virtual_price) < 0 && aGammaTime == 0 {
			return errors.New("loss")
		}
		if aGammaTime == 1 {
			t.FutureAGammaTime = 0
		}
	}
	t.XcpProfit = xcp_profit
	var needs_adjustment = t.NotAdjusted
	if new(big.Int).Sub(new(big.Int).Mul(virtual_price, constant.Two), constant.BONE).Cmp(
		new(big.Int).Add(xcp_profit, new(big.Int).Mul(constant.Two, t.AllowedExtraProfit))) > 0 {
		needs_adjustment = true
		t.NotAdjusted = true
	}
	if needs_adjustment {
		var adjustment_step = t.AdjustmentStep
		var norm = constant.Zero
		for k := 0; k < nCoins-1; k += 1 {
			var ratio = new(big.Int).Div(new(big.Int).Mul(price_oracle[k], constant.BONE), price_scale[k])
			if ratio.Cmp(constant.BONE) > 0 {
				ratio = new(big.Int).Sub(ratio, constant.BONE)
			} else {
				ratio = new(big.Int).Sub(constant.BONE, ratio)
			}
			norm = new(big.Int).Add(norm, new(big.Int).Mul(ratio, ratio))
		}
		if norm.Cmp(new(big.Int).Mul(adjustment_step, adjustment_step)) > 0 && old_virtual_price.Cmp(constant.Zero) > 0 {
			var temp, err = sqrt_int(new(big.Int).Div(norm, constant.BONE))
			if err != nil {
				return err
			}
			norm = temp
			for k := 0; k < nCoins-1; k += 1 {
				p_new[k] = new(big.Int).Div(
					new(big.Int).Add(
						new(big.Int).Mul(price_scale[k], new(big.Int).Sub(norm, adjustment_step)),
						new(big.Int).Mul(adjustment_step, price_oracle[k]),
					), norm)
			}
			for k := 0; k < nCoins; k += 1 {
				xp[k] = new(big.Int).Set(_xp[k])
			}
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(big.Int).Div(new(big.Int).Mul(_xp[k+1], p_new[k]), price_scale[k])
			}
			D, err := newton_D(A_gamma[0], A_gamma[1], xp)
			if err != nil {
				return err
			}

			xp[0] = new(big.Int).Div(D, nCoinsBi)
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(big.Int).Div(new(big.Int).Mul(D, constant.BONE), new(big.Int).Mul(nCoinsBi, p_new[k]))
			}
			temp, err = _geometric_mean(xp, true)
			if err != nil {
				return err
			}
			old_virtual_price = new(big.Int).Div(new(big.Int).Mul(constant.BONE, temp), total_supply)
			if old_virtual_price.Cmp(constant.BONE) > 0 && new(big.Int).Sub(new(big.Int).Mul(constant.Two, old_virtual_price), constant.BONE).Cmp(xcp_profit) > 0 {
				packed_prices = constant.Zero
				for k := 0; k < nCoins-1; k += 1 {
					packed_prices = new(big.Int).Lsh(packed_prices, PriceSize)
					packed_prices = new(big.Int).Or(p_new[nCoins-2-k], packed_prices)
				}
				t.PriceScalePacked = packed_prices
				t.D = D
				t.VirtualPrice = old_virtual_price
				return nil
			} else {
				t.NotAdjusted = false
			}
		}
	}
	t.D = D_unadjusted
	t.VirtualPrice = virtual_price
	return nil
}

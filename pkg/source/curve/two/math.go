package two

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

func geometricMean(unsortedX []*big.Int, sort bool) (*big.Int, error) {
	var nCoins = len(unsortedX)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = unsortedX
	if sort {
		x = sortArray(unsortedX)
	}
	var D = x[0]
	var diff = bignumber.ZeroBI
	for i := 0; i < 255; i += 1 {
		var DPrev = D
		var tmp = bignumber.BONE
		for _, _x := range x {
			tmp = new(big.Int).Div(new(big.Int).Mul(tmp, _x), D)
		}
		D = new(big.Int).Div(
			new(big.Int).Mul(
				D, new(big.Int).Add(new(big.Int).Mul(big.NewInt(int64(nCoins-1)), bignumber.BONE), tmp),
			), new(big.Int).Mul(nCoinsBi, bignumber.BONE),
		)
		if D.Cmp(DPrev) > 0 {
			diff = new(big.Int).Sub(D, DPrev)
		} else {
			diff = new(big.Int).Sub(DPrev, D)
		}
		if diff.Cmp(bignumber.One) <= 0 || new(big.Int).Mul(diff, bignumber.BONE).Cmp(D) < 0 {
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func sqrtInt(x *big.Int) (*big.Int, error) {
	if x.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI, nil
	}
	var z = new(big.Int).Div(new(big.Int).Add(x, bignumber.BONE), bignumber.Two)
	var y = x
	for i := 0; i < 256; i += 1 {
		if z.Cmp(y) == 0 {
			return y, nil
		}
		y = z
		z = new(big.Int).Div(new(big.Int).Add(new(big.Int).Div(new(big.Int).Mul(x, bignumber.BONE), z), z), bignumber.Two)
	}
	return nil, errors.New("sqrt_int did not converge")
}

func newtonD(ANN *big.Int, gamma *big.Int, xUnsorted []*big.Int) (*big.Int, error) {
	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
		return nil, errors.New("unsafe values gamma")
	}
	var nCoins = len(xUnsorted)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var x = sortArray(xUnsorted)
	if x[0].Cmp(bignumber.TenPowInt(9)) < 0 || x[0].Cmp(bignumber.TenPowInt(33)) > 0 {
		return nil, errors.New("unsafe values x[0]")
	}
	for i := 1; i < nCoins; i += 1 {
		var frac = new(big.Int).Div(new(big.Int).Mul(x[i], bignumber.BONE), x[0])
		if frac.Cmp(bignumber.TenPowInt(11)) < 0 {
			return nil, errors.New("unsafe values x[i]")
		}
	}
	var mean, err = geometricMean(x, false)
	if err != nil {
		return nil, err
	}
	var D = new(big.Int).Mul(nCoinsBi, mean)
	var S = bignumber.ZeroBI
	for _, xI := range x {
		S = new(big.Int).Add(S, xI)
	}
	for i := 0; i < 255; i += 1 {
		var DPrev = D
		var K0 = bignumber.BONE
		for _, _x := range x {
			K0 = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K0, _x), nCoinsBi), D)
		}
		var _g1k0 = new(big.Int).Add(gamma, bignumber.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(_g1k0, K0), bignumber.One)
		} else {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, _g1k0), bignumber.One)
		}
		var mul1 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, D), gamma), _g1k0,
						), gamma,
					),
					_g1k0,
				),
				AMultiplier,
			), ANN,
		)
		var mul2 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					new(big.Int).Mul(bignumber.Two, bignumber.BONE), nCoinsBi,
				), K0,
			), _g1k0,
		)
		var negFprime = new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Add(S, new(big.Int).Div(new(big.Int).Mul(S, mul2), bignumber.BONE)),
				new(big.Int).Div(new(big.Int).Mul(mul1, nCoinsBi), K0),
			),
			new(big.Int).Div(new(big.Int).Mul(mul2, D), bignumber.BONE),
		)
		var DPlus = new(big.Int).Div(new(big.Int).Mul(D, new(big.Int).Add(negFprime, S)), negFprime)
		var DMinus = new(big.Int).Div(new(big.Int).Mul(D, D), negFprime)
		if bignumber.BONE.Cmp(K0) > 0 {
			DMinus = new(big.Int).Add(
				DMinus,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(
							new(big.Int).Mul(D, new(big.Int).Div(mul1, negFprime)), bignumber.BONE,
						), new(big.Int).Sub(bignumber.BONE, K0),
					),
					K0,
				),
			)
		} else {
			DMinus = new(big.Int).Sub(
				DMinus,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(
							new(big.Int).Mul(D, new(big.Int).Div(mul1, negFprime)), bignumber.BONE,
						), new(big.Int).Sub(K0, bignumber.BONE),
					),
					K0,
				),
			)
		}
		if DPlus.Cmp(DMinus) > 0 {
			D = new(big.Int).Sub(DPlus, DMinus)
		} else {
			D = new(big.Int).Div(new(big.Int).Sub(DMinus, DPlus), bignumber.Two)
		}
		var diff *big.Int
		if D.Cmp(DPrev) > 0 {
			diff = new(big.Int).Sub(D, DPrev)
		} else {
			diff = new(big.Int).Sub(DPrev, D)
		}
		var temp = bignumber.TenPowInt(16)
		if D.Cmp(temp) > 0 {
			temp = D
		}
		if new(big.Int).Mul(diff, bignumber.TenPowInt(14)).Cmp(temp) < 0 {
			for _, _x := range x {
				var frac = new(big.Int).Div(new(big.Int).Mul(_x, bignumber.BONE), D)
				if frac.Cmp(bignumber.TenPowInt(16)) < 0 || frac.Cmp(bignumber.TenPowInt(20)) > 0 {
					return nil, errors.New("unsafe values x[i]")
				}
			}
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonY(ann *big.Int, gamma *big.Int, x []*big.Int, D *big.Int, i int) (*big.Int, error) {
	// Reference: https://github.com/curvefi/curve-crypto-contract/blob/master/contracts/two/CurveCryptoSwap2ETH.vy#L346-L349
	if ann.Cmp(new(big.Int).Sub(MinA, bignumber.One)) <= 0 || ann.Cmp(new(big.Int).Add(MaxA, bignumber.One)) >= 0 {
		return nil, errors.New("unsafe values A")
	}

	if gamma.Cmp(new(big.Int).Sub(MinGamma, bignumber.One)) <= 0 || gamma.Cmp(new(big.Int).Add(MaxGamma, bignumber.One)) >= 0 {
		return nil, errors.New("unsafe values gamma")
	}

	if D.Cmp(new(big.Int).Sub(bignumber.TenPowInt(17), bignumber.One)) <= 0 {
		return nil, errors.New("unsafe values D")
	}
	if D.Cmp(new(big.Int).Add(new(big.Int).Mul(bignumber.TenPowInt(15), bignumber.TenPowInt(18)), bignumber.One)) >= 0 {
		return nil, errors.New("unsafe values D")
	}

	var nCoins = len(x)
	var nCoinBi = big.NewInt(int64(nCoins))
	var y = new(big.Int).Div(D, nCoinBi)
	var K0i = bignumber.BONE
	var Si = bignumber.ZeroBI

	var xSorted = make([]*big.Int, nCoins)
	for j := 0; j < nCoins; j += 1 {
		xSorted[j] = x[j]
	}
	xSorted[i] = bignumber.ZeroBI
	xSorted = sortArray(xSorted)
	var tenPow14 = bignumber.TenPowInt(14)
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
		if _x.Cmp(bignumber.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}
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
		var _g1k0 = new(big.Int).Add(gamma, bignumber.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(_g1k0, K0), bignumber.One)
		} else {
			_g1k0 = new(big.Int).Add(new(big.Int).Sub(K0, _g1k0), bignumber.One)
		}
		var mul1 = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, D), gamma),
						_g1k0,
					), gamma,
				),
				new(big.Int).Mul(_g1k0, AMultiplier),
			), ann,
		)
		var mul2 = new(big.Int).Add(
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Mul(bignumber.Two, bignumber.BONE), K0,
				), _g1k0,
			), bignumber.BONE,
		)
		var yfprime = new(big.Int).Add(
			new(big.Int).Add(new(big.Int).Mul(bignumber.BONE, y), new(big.Int).Mul(S, mul2)), mul1,
		)
		var _dyfprime = new(big.Int).Mul(D, mul2)
		if yfprime.Cmp(_dyfprime) < 0 {
			y = new(big.Int).Div(yPrev, bignumber.Two)
			continue
		} else {
			yfprime = new(big.Int).Sub(yfprime, _dyfprime)
		}

		if y.Cmp(bignumber.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var fprime = new(big.Int).Div(yfprime, y)

		if fprime.Cmp(bignumber.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var yMinus = new(big.Int).Div(mul1, fprime)
		var yPlus = new(big.Int).Add(
			new(big.Int).Div(
				new(big.Int).Add(yfprime, new(big.Int).Mul(bignumber.BONE, D)),
				fprime,
			),
			new(big.Int).Div(new(big.Int).Mul(yMinus, bignumber.BONE), K0),
		)
		yMinus = new(big.Int).Add(yMinus, new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, S), fprime))
		if yPlus.Cmp(yMinus) < 0 {
			y = new(big.Int).Div(yPrev, bignumber.Two)
		} else {
			y = new(big.Int).Sub(yPlus, yMinus)
		}
		var diff = bignumber.ZeroBI
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
			var frac = new(big.Int).Div(new(big.Int).Mul(y, bignumber.BONE), D)
			if frac.Cmp(bignumber.TenPowInt(16)) < 0 || frac.Cmp(bignumber.TenPowInt(20)) > 0 {
				return nil, errors.New("unsafe value for y")
			}
			return y, nil
		}
	}
	return nil, errors.New("did not converge")
}

func reductionCoefficient(x []*big.Int, feeGamma *big.Int) *big.Int {
	var nCoinsBi = big.NewInt(int64(len(x)))
	var K = bignumber.BONE
	var S = bignumber.ZeroBI
	for _, xi := range x {
		S = new(big.Int).Add(S, xi)
	}
	for _, xi := range x {
		K = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mul(K, nCoinsBi), xi), S)
	}
	if feeGamma.Cmp(bignumber.ZeroBI) > 0 {
		K = new(big.Int).Div(
			new(big.Int).Mul(feeGamma, bignumber.BONE), new(big.Int).Sub(new(big.Int).Add(feeGamma, bignumber.BONE), K),
		)
	}
	return K
}

func halfpow(power *big.Int, precision *big.Int) (*big.Int, error) {
	var intpow = new(big.Int).Div(power, bignumber.BONE)
	var otherpow = new(big.Int).Sub(power, new(big.Int).Mul(intpow, bignumber.BONE))
	if intpow.Cmp(big.NewInt(59)) > 0 {
		return bignumber.ZeroBI, nil
	}
	var result = new(big.Int).Div(bignumber.BONE, new(big.Int).Exp(bignumber.Two, intpow, nil))
	if otherpow.Cmp(bignumber.ZeroBI) == 0 {
		return result, nil
	}
	var term = bignumber.BONE
	var x = new(big.Int).Mul(bignumber.Five, bignumber.TenPowInt(17))
	var S = bignumber.BONE
	var neg = false
	for i := 1; i < 256; i += 1 {
		var K = new(big.Int).Mul(big.NewInt(int64(i)), bignumber.BONE)
		var c = new(big.Int).Sub(K, bignumber.BONE)
		if otherpow.Cmp(c) > 0 {
			c = new(big.Int).Sub(otherpow, c)
			neg = !neg
		} else {
			c = new(big.Int).Sub(c, otherpow)
		}
		term = new(big.Int).Div(new(big.Int).Mul(term, new(big.Int).Div(new(big.Int).Mul(c, x), bignumber.BONE)), K)
		if neg {
			S = new(big.Int).Sub(S, term)
		} else {
			S = new(big.Int).Add(S, term)
		}
		if term.Cmp(precision) < 0 {
			return new(big.Int).Div(new(big.Int).Mul(result, S), bignumber.BONE), nil
		}
	}
	return nil, errors.New("did not converge")
}

// func (t *PoolSimulator) packedView(k uint, p *big.Int) *big.Int {
// 	var ret = new(big.Int).Rsh(p, k*128)
// 	return new(big.Int).And(ret, PriceMask)
// }
//
// func (t *PoolSimulator) priceScale(k uint) *big.Int {
// 	return t.packedView(k, t.PriceScalePacked)
// }

func (t *PoolSimulator) aGamma() []*big.Int {
	var t1 = t.FutureAGammaTime
	var AGamma1 = t.FutureAGamma
	var gamma1 = new(big.Int).And(AGamma1, PriceMask)
	var A1 = new(big.Int).Rsh(AGamma1, 128)
	var now = time.Now().Unix()
	if now < t1 {
		var AGamma0 = t.InitialAGamma
		var t0 = t.InitialAGammaTime
		t1 -= t0
		t0 = now - t0
		var t2 = t1 - t0
		A1 = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(
					new(big.Int).Rsh(AGamma0, 128),
					big.NewInt(t2),
				),
				new(big.Int).Mul(
					A1,
					big.NewInt(t0),
				),
			), big.NewInt(t1),
		)
		gamma1 = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).And(AGamma0, PriceMask), big.NewInt(t2)),
				new(big.Int).Mul(gamma1, big.NewInt(t0)),
			), big.NewInt(t1),
		)
	}
	return []*big.Int{
		A1, gamma1,
	}
}

func (t *PoolSimulator) FeeCalc(xp []*big.Int) *big.Int {
	var f = reductionCoefficient(xp, t.FeeGamma)
	var ret = new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(t.MidFee, f), new(big.Int).Mul(t.OutFee, new(big.Int).Sub(bignumber.BONE, f)),
		), bignumber.BONE,
	)
	return ret
}

// GetDy https://github.com/curvefi/curve-crypto-contract/blob/d7d04cd9ae038970e40be850df99de8c1ff7241b/contracts/two/CurveCryptoSwap2.vy#L842
func (t *PoolSimulator) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	if i == j {
		return nil, nil, fmt.Errorf("tokenIn and tokenOut must not be the same")
	}
	if i < 0 || i > len(t.GetTokens()) || j < 0 || j > len(t.GetTokens()) {
		return nil, nil, fmt.Errorf("token index is out of range")
	}

	var priceScale = new(big.Int).Mul(t.PriceScalePacked, t.Precisions[1])

	var xp = []*big.Int{t.Info.Reserves[0], t.Info.Reserves[1]} // xp: uint256[N_COINS] = self.balances
	xp[i] = new(big.Int).Add(xp[i], dx)
	xp[0] = new(big.Int).Mul(xp[0], t.Precisions[0])
	xp[1] = new(big.Int).Div(new(big.Int).Mul(xp[1], priceScale), Precision)

	var aGamma = t.aGamma()
	var y, err = newtonY(aGamma[0], aGamma[1], xp, t.D, j)
	if err != nil {
		return nil, nil, err
	}

	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[j], y), bignumber.One) // dy: uint256 = xp[j] - y - 1
	xp[j] = y
	if j > 0 {
		dy = new(big.Int).Div(new(big.Int).Mul(dy, Precision), priceScale) // dy = dy * PRECISION / price_scale
	} else {
		dy = new(big.Int).Div(dy, t.Precisions[0]) // dy /= precisions[0]
	}
	// dy -= self._fee(xp) * dy / 10**10
	var fee = t.FeeCalc(xp)
	dy = new(big.Int).Sub(dy, new(big.Int).Div(new(big.Int).Mul(fee, dy), bignumber.TenPowInt(10)))

	return dy, fee, nil
}

func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	var nCoins = len(t.Info.Tokens)
	if i == j {
		return nil, errors.New("i = j")
	}
	if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
		return nil, errors.New("coin index out of range")
	}
	if dx.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, errors.New("do not exchange 0 coins")
	}

	var AGamma = t.aGamma()
	var xp = make([]*big.Int, nCoins)
	for k := 0; k < nCoins; k += 1 {
		xp[k] = t.Info.Reserves[k]
	}
	var ix = j
	var p = bignumber.ZeroBI
	var dy = bignumber.ZeroBI

	var y = xp[j]
	var x0 = xp[i]
	xp[i] = new(big.Int).Add(x0, dx)
	t.Info.Reserves[i] = new(big.Int).Set(xp[i])
	var priceScale = make([]*big.Int, nCoins-1)
	var packedPrice = t.PriceScalePacked

	for k := 0; k < nCoins-1; k += 1 {
		priceScale[k] = new(big.Int).And(packedPrice, PriceMask)
		packedPrice = new(big.Int).Rsh(packedPrice, PriceSize)
	}
	xp[0] = new(big.Int).Mul(xp[0], t.Precisions[0])
	for k := 1; k < nCoins; k += 1 {
		xp[k] = new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).Mul(xp[k], priceScale[k-1]), t.Precisions[k]), Precision,
		)
	}
	{
		var ti = t.FutureAGammaTime
		if ti > 0 {
			x0 = new(big.Int).Mul(x0, t.Precisions[i])
			if i > 0 {
				x0 = new(big.Int).Div(new(big.Int).Mul(x0, priceScale[i-1]), Precision)
			}
			var x1 = xp[i]
			xp[i] = x0
			var temp, err = newtonD(AGamma[0], AGamma[1], xp)
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
	var temp, err = newtonY(AGamma[0], AGamma[1], xp, t.D, j)
	if err != nil {
		return nil, err
	}
	dy = new(big.Int).Sub(xp[j], temp)
	xp[j] = new(big.Int).Sub(xp[j], dy)
	dy = new(big.Int).Sub(dy, bignumber.One)
	if j > 0 {
		dy = new(big.Int).Div(new(big.Int).Mul(dy, Precision), priceScale[j-1])
	}
	dy = new(big.Int).Div(dy, t.Precisions[j])
	dy = new(big.Int).Sub(dy, new(big.Int).Div(new(big.Int).Mul(t.FeeCalc(xp), dy), bignumber.TenPowInt(10)))
	// assert dy >= min_dy, "Slippage"
	y = new(big.Int).Sub(y, dy)
	t.Info.Reserves[j] = y
	y = new(big.Int).Mul(y, t.Precisions[j])
	if j > 0 {
		y = new(big.Int).Div(new(big.Int).Mul(y, priceScale[j-1]), Precision)
	}
	xp[j] = y
	if dx.Cmp(bignumber.TenPowInt(5)) > 0 && dy.Cmp(bignumber.TenPowInt(5)) > 0 {
		var _dx = new(big.Int).Mul(dx, t.Precisions[i])
		var _dy = new(big.Int).Mul(dy, t.Precisions[j])
		if i != 0 && j != 0 {
			p = new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).And(
						new(big.Int).Rsh(
							t.LastPricesPacked, PriceSize*uint(i-1),
						), PriceMask,
					), _dx,
				), _dy,
			)
		} else if i == 0 {
			p = new(big.Int).Div(new(big.Int).Mul(_dx, bignumber.BONE), _dy)
		} else {
			p = new(big.Int).Div(new(big.Int).Mul(_dy, bignumber.BONE), _dx)
			ix = i
		}
	}
	err = t.tweakPrice(AGamma, xp, ix, p, bignumber.ZeroBI)
	return dy, err
}

func (t *PoolSimulator) tweakPrice(AGamma []*big.Int, _xp []*big.Int, i int, pI *big.Int, newD *big.Int) error {
	var nCoins = len(_xp)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var priceOracle = make([]*big.Int, nCoins-1)
	var lastPrices = make([]*big.Int, nCoins-1)
	var priceScale = make([]*big.Int, nCoins-1)
	var xp = make([]*big.Int, nCoins)
	var pNew = make([]*big.Int, nCoins-1)

	var packedPrices = t.PriceOraclePacked
	for k := 0; k < nCoins-1; k += 1 {
		priceOracle[k] = new(big.Int).And(packedPrices, PriceMask)
		packedPrices = new(big.Int).Rsh(packedPrices, PriceSize)
	}
	var lastPricesTimestamp = t.LastPricesTimestamp
	packedPrices = t.LastPricesPacked
	for k := 0; k < nCoins-1; k += 1 {
		lastPrices[k] = new(big.Int).And(packedPrices, PriceMask)
		packedPrices = new(big.Int).Rsh(packedPrices, PriceSize)
	}
	var blockTimestamp = time.Now().Unix()
	if lastPricesTimestamp < blockTimestamp {
		var maHalfTime = t.MaHalfTime
		var alpha, _ = halfpow(
			new(big.Int).Div(
				new(big.Int).Mul(big.NewInt(blockTimestamp-lastPricesTimestamp), bignumber.BONE), maHalfTime,
			),
			bignumber.TenPowInt(10),
		)
		packedPrices = bignumber.ZeroBI
		for k := 0; k < nCoins-1; k += 1 {
			priceOracle[k] = new(big.Int).Div(
				new(big.Int).Add(
					new(big.Int).Mul(lastPrices[k], new(big.Int).Sub(bignumber.BONE, alpha)),
					new(big.Int).Mul(priceOracle[k], alpha),
				),
				bignumber.BONE,
			)
		}
		for k := 0; k < nCoins-1; k += 1 {
			packedPrices = new(big.Int).Lsh(packedPrices, PriceSize)
			var p = priceOracle[nCoins-2-k]
			packedPrices = new(big.Int).Or(p, packedPrices)
		}
		t.PriceOraclePacked = packedPrices
		t.LastPricesTimestamp = blockTimestamp
	}
	var DUnadjusted = newD
	if newD.Cmp(bignumber.ZeroBI) == 0 {
		DUnadjusted, _ = newtonD(AGamma[0], AGamma[1], _xp)
	}
	packedPrices = t.PriceScalePacked
	for k := 0; k < nCoins-1; k += 1 {
		priceScale[k] = new(big.Int).And(packedPrices, PriceMask)
		packedPrices = new(big.Int).Rsh(packedPrices, PriceSize)
	}
	if pI.Cmp(bignumber.ZeroBI) > 0 {
		if i > 0 {
			lastPrices[i-1] = pI
		} else {
			for k := 0; k < nCoins-1; k += 1 {
				lastPrices[k] = new(big.Int).Div(new(big.Int).Mul(lastPrices[k], bignumber.BONE), pI)
			}
		}
	} else {
		var __xp = make([]*big.Int, nCoins)
		for k := 0; k < nCoins; k += 1 {
			__xp[k] = new(big.Int).Set(_xp[k])
		}
		var dxPrice = new(big.Int).Div(__xp[0], bignumber.TenPowInt(6))
		__xp[0] = new(big.Int).Add(__xp[0], dxPrice)
		for k := 0; k < nCoins-1; k += 1 {
			var temp, err = newtonY(AGamma[0], AGamma[1], __xp, DUnadjusted, k+1)
			if err != nil {
				return err
			}
			lastPrices[k] = new(big.Int).Div(
				new(big.Int).Mul(priceScale[k], dxPrice), new(big.Int).Sub(_xp[k+1], temp),
			)
		}
	}
	packedPrices = bignumber.ZeroBI
	for k := 0; k < nCoins-1; k += 1 {
		packedPrices = new(big.Int).Lsh(packedPrices, PriceSize)
		var p = lastPrices[nCoins-2-k]
		packedPrices = new(big.Int).Or(p, packedPrices)
	}
	t.LastPricesPacked = packedPrices

	var totalSupply = t.LpSupply
	var oldXcpProfit = t.XcpProfit
	var oldVirtualPrice = t.VirtualPrice

	xp[0] = new(big.Int).Div(DUnadjusted, nCoinsBi)
	for k := 0; k < nCoins-1; k += 1 {
		xp[k+1] = new(big.Int).Div(
			new(big.Int).Mul(DUnadjusted, bignumber.BONE), new(big.Int).Mul(nCoinsBi, priceScale[k]),
		)
	}
	var xcpProfit = bignumber.BONE
	var virtualPrice = bignumber.BONE
	if oldVirtualPrice.Cmp(bignumber.ZeroBI) > 0 {
		var xcp, err = geometricMean(xp, true)
		if err != nil {
			return err
		}
		virtualPrice = new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, xcp), totalSupply)
		xcpProfit = new(big.Int).Div(new(big.Int).Mul(oldXcpProfit, virtualPrice), oldVirtualPrice)
		var aGammaTime = t.FutureAGammaTime
		if virtualPrice.Cmp(oldVirtualPrice) < 0 && aGammaTime == 0 {
			return errors.New("loss")
		}
		if aGammaTime == 1 {
			t.FutureAGammaTime = 0
		}
	}
	t.XcpProfit = xcpProfit
	var needsAdjustment = t.NotAdjusted
	if new(big.Int).Sub(new(big.Int).Mul(virtualPrice, bignumber.Two), bignumber.BONE).Cmp(
		new(big.Int).Add(xcpProfit, new(big.Int).Mul(bignumber.Two, t.AllowedExtraProfit)),
	) > 0 {
		needsAdjustment = true
		t.NotAdjusted = true
	}
	if needsAdjustment {
		var adjustmentStep = t.AdjustmentStep
		var norm = bignumber.ZeroBI
		for k := 0; k < nCoins-1; k += 1 {
			var ratio = new(big.Int).Div(new(big.Int).Mul(priceOracle[k], bignumber.BONE), priceScale[k])
			if ratio.Cmp(bignumber.BONE) > 0 {
				ratio = new(big.Int).Sub(ratio, bignumber.BONE)
			} else {
				ratio = new(big.Int).Sub(bignumber.BONE, ratio)
			}
			norm = new(big.Int).Add(norm, new(big.Int).Mul(ratio, ratio))
		}
		if norm.Cmp(
			new(big.Int).Mul(
				adjustmentStep, adjustmentStep,
			),
		) > 0 && oldVirtualPrice.Cmp(bignumber.ZeroBI) > 0 {
			var temp, err = sqrtInt(new(big.Int).Div(norm, bignumber.BONE))
			if err != nil {
				return err
			}
			norm = temp
			for k := 0; k < nCoins-1; k += 1 {
				pNew[k] = new(big.Int).Div(
					new(big.Int).Add(
						new(big.Int).Mul(priceScale[k], new(big.Int).Sub(norm, adjustmentStep)),
						new(big.Int).Mul(adjustmentStep, priceOracle[k]),
					), norm,
				)
			}
			for k := 0; k < nCoins; k += 1 {
				xp[k] = new(big.Int).Set(_xp[k])
			}
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(big.Int).Div(new(big.Int).Mul(_xp[k+1], pNew[k]), priceScale[k])
			}
			D, err := newtonD(AGamma[0], AGamma[1], xp)
			if err != nil {
				return err
			}

			xp[0] = new(big.Int).Div(D, nCoinsBi)
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(big.Int).Div(new(big.Int).Mul(D, bignumber.BONE), new(big.Int).Mul(nCoinsBi, pNew[k]))
			}
			temp, err = geometricMean(xp, true)
			if err != nil {
				return err
			}
			oldVirtualPrice = new(big.Int).Div(new(big.Int).Mul(bignumber.BONE, temp), totalSupply)
			if oldVirtualPrice.Cmp(bignumber.BONE) > 0 && new(big.Int).Sub(
				new(big.Int).Mul(
					bignumber.Two, oldVirtualPrice,
				), bignumber.BONE,
			).Cmp(xcpProfit) > 0 {
				packedPrices = bignumber.ZeroBI
				for k := 0; k < nCoins-1; k += 1 {
					packedPrices = new(big.Int).Lsh(packedPrices, PriceSize)
					packedPrices = new(big.Int).Or(pNew[nCoins-2-k], packedPrices)
				}
				t.PriceScalePacked = packedPrices
				t.D = D
				t.VirtualPrice = oldVirtualPrice
				return nil
			} else {
				t.NotAdjusted = false
			}
		}
	}
	t.D = DUnadjusted
	t.VirtualPrice = virtualPrice
	return nil
}

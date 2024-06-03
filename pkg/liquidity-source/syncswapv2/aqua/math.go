package syncswapv2aqua

import (
	"errors"
	"time"

	"github.com/holiman/uint256"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrDenominatorZero      = errors.New("denominator should not be 0")
	PriceMask               = new(uint256.Int).Sub(new(uint256.Int).Lsh(constant.One, 128), constant.One)
	PriceSize          uint = 128
)

func sortArray(A0 []*uint256.Int) []*uint256.Int {
	var nCoins = len(A0)
	var ret = make([]*uint256.Int, nCoins)
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

func geometricMean(unsortedX []*uint256.Int, sort bool) (*uint256.Int, error) {
	// https://gist.github.com/0xnakato/3785ba596c6fa661a5bc56f045360bf6#file-syncswaphelper-ts-L2099
	// Assuming ethers has sqrt method for BigNumber (This may be an oversimplification, and a more detailed algorithm might be needed for accurate square root computation)
	// sqrt(x0.mul(x1));
	res := constant.One
	for _, x := range unsortedX {
		res = new(uint256.Int).Mul(res, x)
	}
	return new(uint256.Int).Sqrt(res), nil
}

func sqrtInt(x *uint256.Int) (*uint256.Int, error) {
	if x.Cmp(constant.ZeroBI) == 0 {
		return constant.ZeroBI, nil
	}
	var z = new(uint256.Int).Div(new(uint256.Int).Add(x, constant.BONE), constant.Two)
	var y = x
	for i := 0; i < 256; i += 1 {
		if z.Cmp(y) == 0 {
			return y, nil
		}
		y = z
		z = new(uint256.Int).Div(new(uint256.Int).Add(new(uint256.Int).Div(new(uint256.Int).Mul(x, constant.BONE), z), z), constant.Two)
	}
	return nil, errors.New("sqrt_int did not converge")
}

func halfpow(power *uint256.Int, precision *uint256.Int) (*uint256.Int, error) {
	var intpow = new(uint256.Int).Div(power, constant.BONE)
	var otherpow = new(uint256.Int).Sub(power, new(uint256.Int).Mul(intpow, constant.BONE))
	if intpow.Cmp(uint256.NewInt(59)) > 0 {
		return constant.ZeroBI, nil
	}
	var result = new(uint256.Int).Div(constant.BONE, new(uint256.Int).Exp(constant.Two, intpow))
	if otherpow.Cmp(constant.ZeroBI) == 0 {
		return result, nil
	}
	var term = constant.BONE
	var x = new(uint256.Int).Mul(constant.Five, constant.TenPowInt(17))
	var S = constant.BONE
	var neg = false
	for i := 1; i < 256; i += 1 {
		var K = new(uint256.Int).Mul(uint256.NewInt(uint64(i)), constant.BONE)
		var c = new(uint256.Int).Sub(K, constant.BONE)
		if otherpow.Cmp(c) > 0 {
			c = new(uint256.Int).Sub(otherpow, c)
			neg = !neg
		} else {
			c = new(uint256.Int).Sub(c, otherpow)
		}
		term = new(uint256.Int).Div(new(uint256.Int).Mul(term, new(uint256.Int).Div(new(uint256.Int).Mul(c, x), constant.BONE)), K)
		if neg {
			S = new(uint256.Int).Sub(S, term)
		} else {
			S = new(uint256.Int).Add(S, term)
		}
		if term.Cmp(precision) < 0 {
			return new(uint256.Int).Div(new(uint256.Int).Mul(result, S), constant.BONE), nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonD(ANN *uint256.Int, gamma *uint256.Int, xUnsorted []*uint256.Int) (*uint256.Int, error) {
	var nCoins = len(xUnsorted)
	var nCoinsBi = uint256.NewInt(uint64(nCoins))
	var x = sortArray(xUnsorted)
	if x[0].Cmp(constant.TenPowInt(9)) < 0 || x[0].Cmp(constant.TenPowInt(33)) > 0 {
		return nil, errors.New("unsafe values x[0]")
	}
	for i := 1; i < nCoins; i += 1 {
		var frac = new(uint256.Int).Div(new(uint256.Int).Mul(x[i], constant.BONE), x[0])
		if frac.Cmp(constant.TenPowInt(11)) < 0 {
			return nil, errors.New("unsafe values x[i]")
		}
	}
	var mean, err = geometricMean(x, false)
	if err != nil {
		return nil, err
	}
	var D = new(uint256.Int).Mul(nCoinsBi, mean)
	var S = constant.ZeroBI
	for _, xI := range x {
		S = new(uint256.Int).Add(S, xI)
	}
	for i := 0; i < 255; i += 1 {
		var DPrev = D
		var K0 = constant.BONE
		for _, _x := range x {
			K0 = new(uint256.Int).Div(new(uint256.Int).Mul(new(uint256.Int).Mul(K0, _x), nCoinsBi), D)
		}
		var _g1k0 = new(uint256.Int).Add(gamma, constant.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(uint256.Int).Add(new(uint256.Int).Sub(_g1k0, K0), constant.One)
		} else {
			_g1k0 = new(uint256.Int).Add(new(uint256.Int).Sub(K0, _g1k0), constant.One)
		}
		var mul1 = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(
							new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, D), gamma), _g1k0,
						), gamma,
					),
					_g1k0,
				),
				AMultiplier,
			), ANN,
		)
		var mul2 = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Mul(
					new(uint256.Int).Mul(constant.Two, constant.BONE), nCoinsBi,
				), K0,
			), _g1k0,
		)
		var negFprime = new(uint256.Int).Sub(
			new(uint256.Int).Add(
				new(uint256.Int).Add(S, new(uint256.Int).Div(new(uint256.Int).Mul(S, mul2), constant.BONE)),
				new(uint256.Int).Div(new(uint256.Int).Mul(mul1, nCoinsBi), K0),
			),
			new(uint256.Int).Div(new(uint256.Int).Mul(mul2, D), constant.BONE),
		)
		var DPlus = new(uint256.Int).Div(new(uint256.Int).Mul(D, new(uint256.Int).Add(negFprime, S)), negFprime)
		var DMinus = new(uint256.Int).Div(new(uint256.Int).Mul(D, D), negFprime)
		if constant.BONE.Cmp(K0) > 0 {
			DMinus = new(uint256.Int).Add(
				DMinus,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Div(
							new(uint256.Int).Mul(D, new(uint256.Int).Div(mul1, negFprime)), constant.BONE,
						), new(uint256.Int).Sub(constant.BONE, K0),
					),
					K0,
				),
			)
		} else {
			DMinus = new(uint256.Int).Sub(
				DMinus,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Div(
							new(uint256.Int).Mul(D, new(uint256.Int).Div(mul1, negFprime)), constant.BONE,
						), new(uint256.Int).Sub(K0, constant.BONE),
					),
					K0,
				),
			)
		}
		if DPlus.Cmp(DMinus) > 0 {
			D = new(uint256.Int).Sub(DPlus, DMinus)
		} else {
			D = new(uint256.Int).Div(new(uint256.Int).Sub(DMinus, DPlus), constant.Two)
		}
		var diff *uint256.Int
		if D.Cmp(DPrev) > 0 {
			diff = new(uint256.Int).Sub(D, DPrev)
		} else {
			diff = new(uint256.Int).Sub(DPrev, D)
		}
		var temp = constant.TenPowInt(16)
		if D.Cmp(temp) > 0 {
			temp = D
		}
		if new(uint256.Int).Mul(diff, constant.TenPowInt(14)).Cmp(temp) < 0 {
			for _, _x := range x {
				var frac = new(uint256.Int).Div(new(uint256.Int).Mul(_x, constant.BONE), D)
				if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
					return nil, errors.New("unsafe values x[i]")
				}
			}
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonY(ann *uint256.Int, gamma *uint256.Int, x []*uint256.Int, D *uint256.Int, i int) (*uint256.Int, error) {
	// ann := new(uint256.Int).Mul(A, uint256.NewInt(4))
	// assert D > 10**17 - 1 and D < 10**15 * 10**18 + 1 # dev: unsafe values D
	var nCoins = len(x)
	var nCoinBi = uint256.NewInt(uint64(nCoins))
	var y = new(uint256.Int).Div(D, nCoinBi)
	var K0i = constant.BONE
	var Si = constant.ZeroBI

	var xSorted = make([]*uint256.Int, nCoins)
	for j := 0; j < nCoins; j += 1 {
		xSorted[j] = x[j]
	}
	xSorted[i] = constant.ZeroBI
	xSorted = sortArray(xSorted)
	var tenPow14 = constant.TenPowInt(14)
	var convergenceLimit = new(uint256.Int).Div(xSorted[0], tenPow14)
	var temp = new(uint256.Int).Div(D, tenPow14)
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if uint256.NewInt(100).Cmp(convergenceLimit) > 0 {
		convergenceLimit = uint256.NewInt(100)
	}

	for j := 2; j < nCoins+1; j += 1 {
		var _x = xSorted[nCoins-j]
		if _x.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}
		y = new(uint256.Int).Div(new(uint256.Int).Mul(y, D), new(uint256.Int).Mul(_x, nCoinBi))
		Si = new(uint256.Int).Add(Si, _x)
	}
	for j := 0; j < nCoins-1; j += 1 {
		K0i = new(uint256.Int).Div(new(uint256.Int).Mul(new(uint256.Int).Mul(K0i, xSorted[j]), nCoinBi), D)
	}
	for j := 0; j < 255; j += 1 {
		var yPrev = y
		var K0 = new(uint256.Int).Div(new(uint256.Int).Mul(new(uint256.Int).Mul(K0i, y), nCoinBi), D)
		var S = new(uint256.Int).Add(Si, y)
		var _g1k0 = new(uint256.Int).Add(gamma, constant.BONE)
		if _g1k0.Cmp(K0) > 0 {
			_g1k0 = new(uint256.Int).Add(new(uint256.Int).Sub(_g1k0, K0), constant.One)
		} else {
			_g1k0 = new(uint256.Int).Add(new(uint256.Int).Sub(K0, _g1k0), constant.One)
		}
		var mul1 = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, D), gamma),
						_g1k0,
					), gamma,
				),
				new(uint256.Int).Mul(_g1k0, AMultiplier),
			), ann,
		)
		var mul2 = new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Mul(constant.Two, constant.BONE), K0,
				), _g1k0,
			), constant.BONE,
		)
		var yfprime = new(uint256.Int).Add(
			new(uint256.Int).Add(new(uint256.Int).Mul(constant.BONE, y), new(uint256.Int).Mul(S, mul2)), mul1,
		)
		var _dyfprime = new(uint256.Int).Mul(D, mul2)
		if yfprime.Cmp(_dyfprime) < 0 {
			y = new(uint256.Int).Div(yPrev, constant.Two)
			continue
		} else {
			yfprime = new(uint256.Int).Sub(yfprime, _dyfprime)
		}

		if y.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var fprime = new(uint256.Int).Div(yfprime, y)

		if fprime.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrDenominatorZero
		}

		var yMinus = new(uint256.Int).Div(mul1, fprime)
		var yPlus = new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Add(yfprime, new(uint256.Int).Mul(constant.BONE, D)),
				fprime,
			),
			new(uint256.Int).Div(new(uint256.Int).Mul(yMinus, constant.BONE), K0),
		)
		yMinus = new(uint256.Int).Add(yMinus, new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, S), fprime))
		if yPlus.Cmp(yMinus) < 0 {
			y = new(uint256.Int).Div(yPrev, constant.Two)
		} else {
			y = new(uint256.Int).Sub(yPlus, yMinus)
		}
		var diff = constant.ZeroBI
		if y.Cmp(yPrev) > 0 {
			diff = new(uint256.Int).Sub(y, yPrev)
		} else {
			diff = new(uint256.Int).Sub(yPrev, y)
		}
		var t = new(uint256.Int).Div(y, tenPow14)
		if convergenceLimit.Cmp(t) > 0 {
			t = convergenceLimit
		}
		if diff.Cmp(t) < 0 {
			var frac = new(uint256.Int).Div(new(uint256.Int).Mul(y, constant.BONE), D)
			if frac.Cmp(constant.TenPowInt(16)) < 0 || frac.Cmp(constant.TenPowInt(20)) > 0 {
				return nil, errors.New("unsafe value for y")
			}
			return y, nil
		}
	}
	return nil, errors.New("did not converge")
}

func (t *PoolSimulator) GetDy(i int, j int, dx *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	var priceScale = new(uint256.Int).Mul(t.PriceScalePacked, t.Precisions[1])
	var xp = []*uint256.Int{uint256.MustFromBig(t.Pool.Info.Reserves[0]), uint256.MustFromBig(t.Pool.Info.Reserves[1])} // xp: uint256[N_COINS] = self.balances
	xp[i] = new(uint256.Int).Add(xp[i], dx)
	xp[0] = new(uint256.Int).Mul(xp[0], t.Precisions[0])
	xp[1] = new(uint256.Int).Div(new(uint256.Int).Mul(xp[1], priceScale), Precision)

	var aGamma = t.aGamma()
	D, err1 := t.aD(xp)
	if err1 != nil {
		return nil, nil, err1
	}
	var y, err = newtonY(aGamma[0], aGamma[1], xp, D, j)
	if err != nil {
		return nil, nil, err
	}

	var dy = new(uint256.Int).Sub(new(uint256.Int).Sub(xp[j], y), constant.One) // dy: uint256 = xp[j] - y - 1
	xp[j] = y
	if j > 0 {
		dy = new(uint256.Int).Div(new(uint256.Int).Mul(dy, Precision), priceScale) // dy = dy * PRECISION / price_scale
	} else {
		dy = new(uint256.Int).Div(dy, t.Precisions[0]) // dy /= precisions[0]
	}
	swapFee := getCryptoFee(t.swapFeesMin[i], t.swapFeesMax[i], t.swapFeesGamma[i], xp[i], xp[j])
	amountFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			dy,
			swapFee,
		),
		uint256.NewInt(100000),
	)
	amountOutAfterFee := new(uint256.Int).Sub(
		dy, amountFee,
	)
	return amountOutAfterFee, amountFee, nil
}

func (t *PoolSimulator) Exchange(i int, j int, dx *uint256.Int) (*uint256.Int, error) {
	var nCoins = len(t.Info.Tokens)
	if i == j {
		return nil, errors.New("i = j")
	}
	if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
		return nil, errors.New("coin index out of range")
	}
	if dx.Cmp(constant.ZeroBI) <= 0 {
		return nil, errors.New("do not exchange 0 coins")
	}

	var xp = make([]*uint256.Int, nCoins)
	for k := 0; k < nCoins; k += 1 {
		xp[k] = uint256.MustFromBig(t.Info.Reserves[k])
	}
	var ix = j
	var p = constant.ZeroBI
	var dy = constant.ZeroBI

	var y = xp[j]
	var x0 = xp[i]
	xp[i] = new(uint256.Int).Add(x0, dx)
	t.Info.Reserves[i] = new(uint256.Int).Set(xp[i]).ToBig()
	var priceScale = make([]*uint256.Int, nCoins-1)
	var packedPrice = t.PriceScalePacked
	for k := 0; k < nCoins-1; k += 1 {
		priceScale[k] = new(uint256.Int).And(packedPrice, PriceMask)
		packedPrice = new(uint256.Int).Rsh(packedPrice, PriceSize)
	}
	xp[0] = new(uint256.Int).Mul(xp[0], t.Precisions[0])
	for k := 1; k < nCoins; k += 1 {
		xp[k] = new(uint256.Int).Div(
			new(uint256.Int).Mul(new(uint256.Int).Mul(xp[k], priceScale[k-1]), t.Precisions[k]), Precision,
		)
	}
	var aGamma = t.aGamma()
	D, err1 := t.aD(xp)
	if err1 != nil {
		return nil, err1
	}
	var temp, err = newtonY(aGamma[0], aGamma[1], xp, D, j)
	if err != nil {
		return nil, err
	}
	dy = new(uint256.Int).Sub(xp[j], temp)
	xp[j] = new(uint256.Int).Sub(xp[j], dy)
	dy = new(uint256.Int).Sub(dy, constant.One)
	if j > 0 {
		dy = new(uint256.Int).Div(new(uint256.Int).Mul(dy, Precision), priceScale[j-1])
	}
	dy = new(uint256.Int).Div(dy, t.Precisions[j])
	swapFee := getCryptoFee(t.swapFeesMin[i], t.swapFeesMax[i], t.swapFeesGamma[i], xp[i], xp[j])
	amountFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			dy,
			swapFee,
		),
		uint256.NewInt(100000),
	)
	dy = dy.Sub(dy, amountFee)
	y = new(uint256.Int).Sub(y, dy)
	t.Info.Reserves[j] = y.ToBig()
	y = new(uint256.Int).Mul(y, t.Precisions[j])
	if j > 0 {
		y = new(uint256.Int).Div(new(uint256.Int).Mul(y, priceScale[j-1]), Precision)
	}
	xp[j] = y
	if dx.Cmp(constant.TenPowInt(5)) > 0 && dy.Cmp(constant.TenPowInt(5)) > 0 {
		var _dx = new(uint256.Int).Mul(dx, t.Precisions[i])
		var _dy = new(uint256.Int).Mul(dy, t.Precisions[j])
		if i != 0 && j != 0 {
			p = new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).And(
						new(uint256.Int).Rsh(
							t.LastPricesPacked, PriceSize*uint(i-1),
						), PriceMask,
					), _dx,
				), _dy,
			)
		} else if i == 0 {
			p = new(uint256.Int).Div(new(uint256.Int).Mul(_dx, constant.BONE), _dy)
		} else {
			p = new(uint256.Int).Div(new(uint256.Int).Mul(_dy, constant.BONE), _dx)
			ix = i
		}
	}
	err = t.tweakPrice(aGamma, xp, ix, p, constant.ZeroBI)
	return dy, err
}

func (t *PoolSimulator) tweakPrice(AGamma []*uint256.Int, _xp []*uint256.Int, i int, pI *uint256.Int, newD *uint256.Int) error {
	var nCoins = len(_xp)
	var nCoinsBi = uint256.NewInt(uint64(nCoins))
	var priceOracle = make([]*uint256.Int, nCoins-1)
	var lastPrices = make([]*uint256.Int, nCoins-1)
	var priceScale = make([]*uint256.Int, nCoins-1)
	var xp = make([]*uint256.Int, nCoins)
	var pNew = make([]*uint256.Int, nCoins-1)

	var packedPrices = t.PriceOraclePacked
	for k := 0; k < nCoins-1; k += 1 {
		priceOracle[k] = new(uint256.Int).And(packedPrices, PriceMask)
		packedPrices = new(uint256.Int).Rsh(packedPrices, PriceSize)
	}
	var lastPricesTimestamp = t.LastPricesTimestamp
	packedPrices = t.LastPricesPacked
	for k := 0; k < nCoins-1; k += 1 {
		lastPrices[k] = new(uint256.Int).And(packedPrices, PriceMask)
		packedPrices = new(uint256.Int).Rsh(packedPrices, PriceSize)
	}
	var blockTimestamp = time.Now().Unix()
	if lastPricesTimestamp < blockTimestamp {
		var maHalfTime = t.MaHalfTime
		var alpha, _ = halfpow(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(uint256.NewInt(uint64(blockTimestamp-lastPricesTimestamp)), constant.BONE), maHalfTime,
			),
			constant.TenPowInt(10),
		)
		packedPrices = constant.ZeroBI
		for k := 0; k < nCoins-1; k += 1 {
			priceOracle[k] = new(uint256.Int).Div(
				new(uint256.Int).Add(
					new(uint256.Int).Mul(lastPrices[k], new(uint256.Int).Sub(constant.BONE, alpha)),
					new(uint256.Int).Mul(priceOracle[k], alpha),
				),
				constant.BONE,
			)
		}
		for k := 0; k < nCoins-1; k += 1 {
			packedPrices = new(uint256.Int).Lsh(packedPrices, PriceSize)
			var p = priceOracle[nCoins-2-k]
			packedPrices = new(uint256.Int).Or(p, packedPrices)
		}
		t.PriceOraclePacked = packedPrices
		t.LastPricesTimestamp = blockTimestamp
	}
	var DUnadjusted = newD
	if newD.Cmp(constant.ZeroBI) == 0 {
		DUnadjusted, _ = newtonD(AGamma[0], AGamma[1], _xp)
	}
	packedPrices = t.PriceScalePacked
	for k := 0; k < nCoins-1; k += 1 {
		priceScale[k] = new(uint256.Int).And(packedPrices, PriceMask)
		packedPrices = new(uint256.Int).Rsh(packedPrices, PriceSize)
	}
	if pI.Cmp(constant.ZeroBI) > 0 {
		if i > 0 {
			lastPrices[i-1] = pI
		} else {
			for k := 0; k < nCoins-1; k += 1 {
				lastPrices[k] = new(uint256.Int).Div(new(uint256.Int).Mul(lastPrices[k], constant.BONE), pI)
			}
		}
	} else {
		var __xp = make([]*uint256.Int, nCoins)
		for k := 0; k < nCoins; k += 1 {
			__xp[k] = new(uint256.Int).Set(_xp[k])
		}
		var dxPrice = new(uint256.Int).Div(__xp[0], constant.TenPowInt(6))
		__xp[0] = new(uint256.Int).Add(__xp[0], dxPrice)
		for k := 0; k < nCoins-1; k += 1 {
			var temp, err = newtonY(AGamma[0], AGamma[1], __xp, DUnadjusted, k+1)
			if err != nil {
				return err
			}
			lastPrices[k] = new(uint256.Int).Div(
				new(uint256.Int).Mul(priceScale[k], dxPrice), new(uint256.Int).Sub(_xp[k+1], temp),
			)
		}
	}
	packedPrices = constant.ZeroBI
	for k := 0; k < nCoins-1; k += 1 {
		packedPrices = new(uint256.Int).Lsh(packedPrices, PriceSize)
		var p = lastPrices[nCoins-2-k]
		packedPrices = new(uint256.Int).Or(p, packedPrices)
	}
	t.LastPricesPacked = packedPrices

	var totalSupply = t.LpSupply
	var oldXcpProfit = t.XcpProfit
	var oldVirtualPrice = t.VirtualPrice

	xp[0] = new(uint256.Int).Div(DUnadjusted, nCoinsBi)
	for k := 0; k < nCoins-1; k += 1 {
		xp[k+1] = new(uint256.Int).Div(
			new(uint256.Int).Mul(DUnadjusted, constant.BONE), new(uint256.Int).Mul(nCoinsBi, priceScale[k]),
		)
	}
	var xcpProfit = constant.BONE
	var virtualPrice = constant.BONE
	if oldVirtualPrice.Cmp(constant.ZeroBI) > 0 {
		var xcp, err = geometricMean(xp, true)
		if err != nil {
			return err
		}
		virtualPrice = new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, xcp), totalSupply)
		xcpProfit = new(uint256.Int).Div(new(uint256.Int).Mul(oldXcpProfit, virtualPrice), oldVirtualPrice)
		var aGammaTime = t.FutureTime
		if virtualPrice.Cmp(oldVirtualPrice) < 0 && aGammaTime == 0 {
			return errors.New("loss")
		}
		if aGammaTime == 1 {
			t.FutureTime = 0
		}
	}
	t.XcpProfit = xcpProfit
	var needsAdjustment = t.NotAdjusted
	if new(uint256.Int).Sub(new(uint256.Int).Mul(virtualPrice, constant.Two), constant.BONE).Cmp(
		new(uint256.Int).Add(xcpProfit, new(uint256.Int).Mul(constant.Two, t.AllowedExtraProfit)),
	) > 0 {
		needsAdjustment = true
		t.NotAdjusted = true
	}
	if needsAdjustment {
		var adjustmentStep = t.AdjustmentStep
		var norm = constant.ZeroBI
		for k := 0; k < nCoins-1; k += 1 {
			var ratio = new(uint256.Int).Div(new(uint256.Int).Mul(priceOracle[k], constant.BONE), priceScale[k])
			if ratio.Cmp(constant.BONE) > 0 {
				ratio = new(uint256.Int).Sub(ratio, constant.BONE)
			} else {
				ratio = new(uint256.Int).Sub(constant.BONE, ratio)
			}
			norm = new(uint256.Int).Add(norm, new(uint256.Int).Mul(ratio, ratio))
		}
		if norm.Cmp(
			new(uint256.Int).Mul(
				adjustmentStep, adjustmentStep,
			),
		) > 0 && oldVirtualPrice.Cmp(constant.ZeroBI) > 0 {
			var temp, err = sqrtInt(new(uint256.Int).Div(norm, constant.BONE))
			if err != nil {
				return err
			}
			norm = temp
			for k := 0; k < nCoins-1; k += 1 {
				pNew[k] = new(uint256.Int).Div(
					new(uint256.Int).Add(
						new(uint256.Int).Mul(priceScale[k], new(uint256.Int).Sub(norm, adjustmentStep)),
						new(uint256.Int).Mul(adjustmentStep, priceOracle[k]),
					), norm,
				)
			}
			for k := 0; k < nCoins; k += 1 {
				xp[k] = new(uint256.Int).Set(_xp[k])
			}
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(uint256.Int).Div(new(uint256.Int).Mul(_xp[k+1], pNew[k]), priceScale[k])
			}
			D, err := newtonD(AGamma[0], AGamma[1], xp)
			if err != nil {
				return err
			}

			xp[0] = new(uint256.Int).Div(D, nCoinsBi)
			for k := 0; k < nCoins-1; k += 1 {
				xp[k+1] = new(uint256.Int).Div(new(uint256.Int).Mul(D, constant.BONE), new(uint256.Int).Mul(nCoinsBi, pNew[k]))
			}
			temp, err = geometricMean(xp, true)
			if err != nil {
				return err
			}
			oldVirtualPrice = new(uint256.Int).Div(new(uint256.Int).Mul(constant.BONE, temp), totalSupply)
			if oldVirtualPrice.Cmp(constant.BONE) > 0 && new(uint256.Int).Sub(
				new(uint256.Int).Mul(
					constant.Two, oldVirtualPrice,
				), constant.BONE,
			).Cmp(xcpProfit) > 0 {
				packedPrices = constant.ZeroBI
				for k := 0; k < nCoins-1; k += 1 {
					packedPrices = new(uint256.Int).Lsh(packedPrices, PriceSize)
					packedPrices = new(uint256.Int).Or(pNew[nCoins-2-k], packedPrices)
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

func getCryptoFee(minFee, maxFee, gamma, xp0, xp1 *uint256.Int) *uint256.Int {
	f := new(uint256.Int).Add(xp0, xp1)

	// f = gamma.mul(ETHER).div(
	//     gamma.add(ETHER).sub(ETHER.mul(4).mul(xp0).div(f).mul(xp1).div(f))
	// );

	f = new(uint256.Int).Div(
		new(uint256.Int).Mul(
			gamma,
			constant.BONE,
		),
		new(uint256.Int).Sub(
			new(uint256.Int).Add(
				gamma,
				constant.BONE,
			),
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(
							new(uint256.Int).Mul(
								constant.BONE,
								constant.Four,
							),
							xp0,
						),
						f,
					),
					xp1,
				),
				f,
			),
		),
	)
	// const fee = minFee.mul(f).add(maxFee.mul(ETHER.sub(f))).div(ETHER);
	fee := new(uint256.Int).Div(
		new(uint256.Int).Add(
			new(uint256.Int).Mul(
				minFee, f,
			),
			new(uint256.Int).Mul(
				maxFee,
				new(uint256.Int).Sub(
					constant.BONE, f,
				),
			),
		),
		constant.BONE,
	)
	return fee
}

func (t *PoolSimulator) aGamma() []*uint256.Int {
	return []*uint256.Int{t.A, t.Gamma}
}

func (t *PoolSimulator) aD(xp []*uint256.Int) (*uint256.Int, error) {
	D := t.D
	// https://gist.github.com/0xnakato/3785ba596c6fa661a5bc56f045360bf6#file-syncswaphelper-ts-L1365
	if t.FutureTime > time.Now().Unix() {
		temp, err := newtonD(t.A, t.Gamma, xp)
		if err != nil {
			return nil, err
		}
		D = temp
	}
	return D, nil
}

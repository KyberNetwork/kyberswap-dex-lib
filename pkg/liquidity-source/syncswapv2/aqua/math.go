package syncswapv2aqua

import (
	"errors"
	"time"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrDenominatorZero              = errors.New("denominator should not be 0")
	ErrAmountOutSmallerThanFee      = errors.New("amount out smaller than fee")
	ErrReserveViolation             = errors.New("reserve violation")
	PriceMask                       = new(uint256.Int).Sub(new(uint256.Int).Lsh(big256.U1, 128), big256.U1)
	PriceSize                  uint = 128
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
	var res uint256.Int
	res.SetUint64(1)
	for _, x := range unsortedX {
		res.Mul(&res, x)
	}
	var sqrtResult uint256.Int
	sqrtResult.Sqrt(&res)
	return &sqrtResult, nil
}

func sqrtInt(x *uint256.Int) (*uint256.Int, error) {
	if x.Cmp(big256.U0) == 0 {
		return big256.U0, nil
	}
	var z uint256.Int
	z.Add(x, big256.BONE)
	z.Div(&z, big256.U2)
	var y uint256.Int
	y.Set(x)
	for i := 0; i < 256; i += 1 {
		if z.Cmp(&y) == 0 {
			return &y, nil
		}
		y.Set(&z)
		var inner uint256.Int
		inner.Mul(x, big256.BONE)
		inner.Div(&inner, &z)
		inner.Add(&inner, &z)
		z.Div(&inner, big256.U2)
	}
	return nil, errors.New("sqrt_int did not converge")
}

func halfpow(power *uint256.Int, precision *uint256.Int) (*uint256.Int, error) {
	var intpow uint256.Int
	intpow.Div(power, big256.BONE)
	var otherpowTmp uint256.Int
	otherpowTmp.Mul(&intpow, big256.BONE)
	var otherpow uint256.Int
	otherpow.Sub(power, &otherpowTmp)
	var u59 uint256.Int
	u59.SetUint64(59)
	if intpow.Cmp(&u59) > 0 {
		return big256.U0, nil
	}
	var resultExp uint256.Int
	resultExp.Exp(big256.U2, &intpow)
	var result uint256.Int
	result.Div(big256.BONE, &resultExp)
	if otherpow.Cmp(big256.U0) == 0 {
		return &result, nil
	}
	var term uint256.Int
	term.Set(big256.BONE)
	var x uint256.Int
	x.Mul(big256.U5, big256.TenPow(17))
	var S uint256.Int
	S.Set(big256.BONE)
	var neg = false
	for i := 1; i < 256; i += 1 {
		var K uint256.Int
		K.SetUint64(uint64(i))
		K.Mul(&K, big256.BONE)
		var c uint256.Int
		c.Sub(&K, big256.BONE)
		if otherpow.Cmp(&c) > 0 {
			c.Sub(&otherpow, &c)
			neg = !neg
		} else {
			c.Sub(&c, &otherpow)
		}
		var cx uint256.Int
		cx.Mul(&c, &x)
		cx.Div(&cx, big256.BONE)
		term.Mul(&term, &cx)
		term.Div(&term, &K)
		if neg {
			S.Sub(&S, &term)
		} else {
			S.Add(&S, &term)
		}
		if term.Cmp(precision) < 0 {
			var final uint256.Int
			final.Mul(&result, &S)
			final.Div(&final, big256.BONE)
			return &final, nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonD(ANN *uint256.Int, gamma *uint256.Int, xUnsorted []*uint256.Int) (*uint256.Int, error) {
	var nCoins = len(xUnsorted)
	var nCoinsBi uint256.Int
	nCoinsBi.SetUint64(uint64(nCoins))
	var x = sortArray(xUnsorted)
	if x[0].Cmp(big256.TenPow(9)) < 0 || x[0].Cmp(big256.TenPow(33)) > 0 {
		return nil, errors.New("unsafe values x[0]")
	}
	for i := 1; i < nCoins; i += 1 {
		var frac uint256.Int
		frac.Mul(x[i], big256.BONE)
		frac.Div(&frac, x[0])
		if frac.Cmp(big256.TenPow(11)) < 0 {
			return nil, errors.New("unsafe values x[i]")
		}
	}
	var mean, err = geometricMean(x, false)
	if err != nil {
		return nil, err
	}
	var D uint256.Int
	D.Mul(&nCoinsBi, mean)
	var S uint256.Int
	for _, xI := range x {
		S.Add(&S, xI)
	}
	for i := 0; i < 255; i += 1 {
		var DPrev uint256.Int
		DPrev.Set(&D)
		var K0 uint256.Int
		K0.Set(big256.BONE)
		for _, _x := range x {
			var tmp uint256.Int
			tmp.Mul(&K0, _x)
			tmp.Mul(&tmp, &nCoinsBi)
			K0.Div(&tmp, &D)
		}
		var _g1k0 uint256.Int
		_g1k0.Add(gamma, big256.BONE)
		if _g1k0.Cmp(&K0) > 0 {
			_g1k0.Sub(&_g1k0, &K0)
			_g1k0.Add(&_g1k0, big256.U1)
		} else {
			_g1k0.Sub(&K0, &_g1k0)
			_g1k0.Add(&_g1k0, big256.U1)
		}
		var mul1 uint256.Int
		mul1.Mul(big256.BONE, &D).Div(&mul1, gamma).Mul(&mul1, &_g1k0).Div(&mul1, gamma).Mul(&mul1, &_g1k0).Mul(&mul1, AMultiplier).Div(&mul1, ANN)
		var mul2 uint256.Int
		mul2.Mul(big256.U2, big256.BONE).Mul(&mul2, &nCoinsBi).Mul(&mul2, &K0).Div(&mul2, &_g1k0)
		var negFprime uint256.Int
		negFprime.Mul(&S, &mul2)
		negFprime.Div(&negFprime, big256.BONE)
		negFprime.Add(&S, &negFprime)
		var tmp uint256.Int
		tmp.Mul(&mul1, &nCoinsBi)
		tmp.Div(&tmp, &K0)
		negFprime.Add(&negFprime, &tmp)
		tmp.Mul(&mul2, &D)
		tmp.Div(&tmp, big256.BONE)
		negFprime.Sub(&negFprime, &tmp)
		var DPlus uint256.Int
		DPlus.Add(&negFprime, &S)
		DPlus.Mul(&D, &DPlus)
		DPlus.Div(&DPlus, &negFprime)
		var DMinus uint256.Int
		DMinus.Mul(&D, &D)
		DMinus.Div(&DMinus, &negFprime)
		var adjust uint256.Int
		adjust.Div(&mul1, &negFprime)
		adjust.Mul(&D, &adjust)
		adjust.Div(&adjust, big256.BONE)
		var delta uint256.Int
		if big256.BONE.Cmp(&K0) > 0 {
			delta.Sub(big256.BONE, &K0)
		} else {
			delta.Sub(&K0, big256.BONE)
		}
		adjust.Mul(&adjust, &delta)
		adjust.Div(&adjust, &K0)
		DMinus.Add(&DMinus, &adjust)
		if DPlus.Cmp(&DMinus) > 0 {
			D.Sub(&DPlus, &DMinus)
		} else {
			D.Sub(&DMinus, &DPlus)
			D.Div(&D, big256.U2)
		}
		var diff uint256.Int
		if D.Cmp(&DPrev) > 0 {
			diff.Sub(&D, &DPrev)
		} else {
			diff.Sub(&DPrev, &D)
		}
		var temp uint256.Int
		temp.Set(big256.TenPow(16))
		if D.Cmp(&temp) > 0 {
			temp.Set(&D)
		}
		var diffScaled uint256.Int
		diffScaled.Mul(&diff, big256.TenPow(14))
		if diffScaled.Cmp(&temp) < 0 {
			for _, _x := range x {
				var frac uint256.Int
				frac.Mul(_x, big256.BONE)
				frac.Div(&frac, &D)
				if frac.Cmp(big256.TenPow(16)) < 0 || frac.Cmp(big256.TenPow(20)) > 0 {
					return nil, errors.New("unsafe values x[i]")
				}
			}
			return &D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func newtonY(ann *uint256.Int, gamma *uint256.Int, x []*uint256.Int, D *uint256.Int, i int) (*uint256.Int, error) {
	// ann := new(uint256.Int).Mul(A, uint256.NewInt(4))
	// assert D > 10**17 - 1 and D < 10**15 * 10**18 + 1 # dev: unsafe values D
	var nCoins = len(x)
	var nCoinBi uint256.Int
	nCoinBi.SetUint64(uint64(nCoins))
	var y uint256.Int
	y.Div(D, &nCoinBi)
	var K0i uint256.Int
	K0i.Set(big256.BONE)
	var Si uint256.Int

	var xSorted = make([]*uint256.Int, nCoins)
	for j := 0; j < nCoins; j += 1 {
		xSorted[j] = x[j]
	}
	xSorted[i] = big256.U0
	xSorted = sortArray(xSorted)
	var tenPow14 = big256.TenPow(14)
	var convergenceLimit uint256.Int
	convergenceLimit.Div(xSorted[0], tenPow14)
	var temp uint256.Int
	temp.Div(D, tenPow14)
	if temp.Cmp(&convergenceLimit) > 0 {
		convergenceLimit.Set(&temp)
	}
	var u100 uint256.Int
	u100.SetUint64(100)
	if big256.U100.Cmp(&convergenceLimit) > 0 {
		convergenceLimit.Set(&u100)
	}

	for j := 2; j < nCoins+1; j += 1 {
		var _x = xSorted[nCoins-j]
		if _x.Cmp(big256.U0) == 0 {
			return nil, ErrDenominatorZero
		}
		var denom uint256.Int
		denom.Mul(_x, &nCoinBi)
		y.Mul(&y, D)
		y.Div(&y, &denom)
		Si.Add(&Si, _x)
	}
	for j := 0; j < nCoins-1; j += 1 {
		var tmp uint256.Int
		tmp.Mul(&K0i, xSorted[j])
		tmp.Mul(&tmp, &nCoinBi)
		K0i.Div(&tmp, D)
	}
	for j := 0; j < 255; j += 1 {
		var yPrev uint256.Int
		yPrev.Set(&y)
		var K0 uint256.Int
		K0.Mul(&K0i, &y)
		K0.Mul(&K0, &nCoinBi)
		K0.Div(&K0, D)
		var S uint256.Int
		S.Add(&Si, &y)
		var _g1k0 uint256.Int
		_g1k0.Add(gamma, big256.BONE)
		if _g1k0.Cmp(&K0) > 0 {
			_g1k0.Sub(&_g1k0, &K0)
			_g1k0.Add(&_g1k0, big256.U1)
		} else {
			_g1k0.Sub(&K0, &_g1k0)
			_g1k0.Add(&_g1k0, big256.U1)
		}
		var mul1 uint256.Int
		mul1.Mul(big256.BONE, D).Div(&mul1, gamma).Mul(&mul1, &_g1k0).Div(&mul1, gamma).Mul(&mul1, &_g1k0).Mul(&mul1, AMultiplier).Div(&mul1, ann)
		var mul2 uint256.Int
		mul2.Mul(big256.U2, big256.BONE).Mul(&mul2, &K0).Div(&mul2, &_g1k0).Add(&mul2, big256.BONE)
		var yfprime uint256.Int
		yfprime.Mul(big256.BONE, &y)
		var tmp uint256.Int
		tmp.Mul(&S, &mul2)
		yfprime.Add(&yfprime, &tmp)
		yfprime.Add(&yfprime, &mul1)
		var _dyfprime uint256.Int
		_dyfprime.Mul(D, &mul2)
		if yfprime.Cmp(&_dyfprime) < 0 {
			y.Div(&yPrev, big256.U2)
			continue
		} else {
			yfprime.Sub(&yfprime, &_dyfprime)
		}

		if y.Cmp(big256.U0) == 0 {
			return nil, ErrDenominatorZero
		}

		var fprime uint256.Int
		fprime.Div(&yfprime, &y)

		if fprime.Cmp(big256.U0) == 0 {
			return nil, ErrDenominatorZero
		}

		var yMinus uint256.Int
		yMinus.Div(&mul1, &fprime)
		var yPlus uint256.Int
		tmp.Mul(big256.BONE, D)
		tmp.Add(&yfprime, &tmp)
		yPlus.Div(&tmp, &fprime)
		tmp.Mul(&yMinus, big256.BONE)
		tmp.Div(&tmp, &K0)
		yPlus.Add(&yPlus, &tmp)
		tmp.Mul(big256.BONE, &S)
		tmp.Div(&tmp, &fprime)
		yMinus.Add(&yMinus, &tmp)
		if yPlus.Cmp(&yMinus) < 0 {
			y.Div(&yPrev, big256.U2)
		} else {
			y.Sub(&yPlus, &yMinus)
		}
		var diff uint256.Int
		if y.Cmp(&yPrev) > 0 {
			diff.Sub(&y, &yPrev)
		} else {
			diff.Sub(&yPrev, &y)
		}
		var t uint256.Int
		t.Div(&y, tenPow14)
		if convergenceLimit.Cmp(&t) > 0 {
			t.Set(&convergenceLimit)
		}
		if diff.Cmp(&t) < 0 {
			var frac uint256.Int
			frac.Mul(&y, big256.BONE)
			frac.Div(&frac, D)
			if frac.Cmp(big256.TenPow(16)) < 0 || frac.Cmp(big256.TenPow(20)) > 0 {
				return nil, errors.New("unsafe value for y")
			}
			return &y, nil
		}
	}
	return nil, errors.New("did not converge")
}

func (t *PoolSimulator) GetDy(i int, j int, dx *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	var priceScale uint256.Int
	priceScale.Mul(t.PriceScalePacked, t.Precisions[1])
	var xp = []*uint256.Int{uint256.MustFromBig(t.Info.Reserves[0]), uint256.MustFromBig(t.Info.Reserves[1])} // xp: uint256[N_COINS] = self.balances
	var xpI uint256.Int
	xpI.Add(xp[i], dx)
	xp[i] = &xpI
	var xp0 uint256.Int
	xp0.Mul(xp[0], t.Precisions[0])
	xp[0] = &xp0
	var xp1 uint256.Int
	xp1.Mul(xp[1], &priceScale)
	xp1.Div(&xp1, Precision)
	xp[1] = &xp1

	var aGamma = t.aGamma()
	D, err1 := t.aD(xp)
	if err1 != nil {
		return nil, nil, err1
	}
	var y, err = newtonY(aGamma[0], aGamma[1], xp, D, j)
	if err != nil {
		return nil, nil, err
	}

	var yPlusOne uint256.Int
	yPlusOne.Add(y, big256.U1)
	if xp[j].Cmp(&yPlusOne) < 0 {
		return nil, nil, ErrReserveViolation
	}

	var dy uint256.Int
	dy.Sub(xp[j], y)
	dy.Sub(&dy, big256.U1)
	xp[j] = y
	if j > 0 {
		dy.Mul(&dy, Precision)
		dy.Div(&dy, &priceScale)
	} else {
		dy.Div(&dy, t.Precisions[0])
	}
	swapFee := getCryptoFee(t.swapFeesMin[i], t.swapFeesMax[i], t.swapFeesGamma[i], xp[i], xp[j])
	var feeDenominator uint256.Int
	feeDenominator.SetUint64(100000)
	var amountFee uint256.Int
	amountFee.Mul(&dy, swapFee)
	amountFee.Div(&amountFee, &feeDenominator)

	if dy.Cmp(&amountFee) < 0 {
		return nil, nil, ErrAmountOutSmallerThanFee
	}

	var amountOutAfterFee uint256.Int
	amountOutAfterFee.Sub(&dy, &amountFee)
	return &amountOutAfterFee, &amountFee, nil
}

func (t *PoolSimulator) Exchange(i int, j int, dx *uint256.Int) (*uint256.Int, error) {
	var nCoins = len(t.Info.Tokens)
	if i == j {
		return nil, errors.New("i = j")
	}
	if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
		return nil, errors.New("coin index out of range")
	}
	if dx.Cmp(big256.U0) <= 0 {
		return nil, errors.New("do not exchange 0 coins")
	}

	var xp = make([]*uint256.Int, nCoins)
	for k := 0; k < nCoins; k += 1 {
		xp[k] = uint256.MustFromBig(t.Info.Reserves[k])
	}
	var ix = j
	var p uint256.Int
	var dy uint256.Int

	var y = xp[j]
	var x0 = xp[i]
	var xpI uint256.Int
	xpI.Add(x0, dx)
	xp[i] = &xpI
	t.Info.Reserves[i] = xp[i].ToBig()
	var priceScale = make([]*uint256.Int, nCoins-1)
	var packedPrice uint256.Int
	packedPrice.Set(t.PriceScalePacked)
	for k := 0; k < nCoins-1; k += 1 {
		var price uint256.Int
		price.And(&packedPrice, PriceMask)
		priceScale[k] = &price
		packedPrice.Rsh(&packedPrice, PriceSize)
	}
	var xp0 uint256.Int
	xp0.Mul(xp[0], t.Precisions[0])
	xp[0] = &xp0
	for k := 1; k < nCoins; k += 1 {
		var scaled uint256.Int
		scaled.Mul(xp[k], priceScale[k-1])
		scaled.Mul(&scaled, t.Precisions[k])
		scaled.Div(&scaled, Precision)
		xp[k] = &scaled
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
	dy.Sub(xp[j], temp)
	var nextXPJ uint256.Int
	nextXPJ.Sub(xp[j], &dy)
	xp[j] = &nextXPJ
	dy.Sub(&dy, big256.U1)
	if j > 0 {
		dy.Mul(&dy, Precision)
		dy.Div(&dy, priceScale[j-1])
	}
	dy.Div(&dy, t.Precisions[j])
	swapFee := getCryptoFee(t.swapFeesMin[i], t.swapFeesMax[i], t.swapFeesGamma[i], xp[i], xp[j])
	var feeDenominator uint256.Int
	feeDenominator.SetUint64(100000)
	var amountFee uint256.Int
	amountFee.Mul(&dy, swapFee)
	amountFee.Div(&amountFee, &feeDenominator)
	dy.Sub(&dy, &amountFee)
	var nextY uint256.Int
	nextY.Sub(y, &dy)
	y = &nextY
	t.Info.Reserves[j] = y.ToBig()
	var scaledY uint256.Int
	scaledY.Mul(y, t.Precisions[j])
	if j > 0 {
		scaledY.Mul(&scaledY, priceScale[j-1])
		scaledY.Div(&scaledY, Precision)
	}
	xp[j] = &scaledY
	if dx.Cmp(big256.TenPow(5)) > 0 && dy.Cmp(big256.TenPow(5)) > 0 {
		var dxScaled uint256.Int
		dxScaled.Mul(dx, t.Precisions[i])
		var dyScaled uint256.Int
		dyScaled.Mul(&dy, t.Precisions[j])
		if i != 0 && j != 0 {
			var shifted uint256.Int
			shifted.Rsh(t.LastPricesPacked, PriceSize*uint(i-1))
			var masked uint256.Int
			masked.And(&shifted, PriceMask)
			p.Mul(&masked, &dxScaled)
			p.Div(&p, &dyScaled)
		} else if i == 0 {
			p.Mul(&dxScaled, big256.BONE)
			p.Div(&p, &dyScaled)
		} else {
			p.Mul(&dyScaled, big256.BONE)
			p.Div(&p, &dxScaled)
			ix = i
		}
	}
	err = t.tweakPrice(aGamma, xp, ix, &p, big256.U0)
	return &dy, err
}

func (t *PoolSimulator) tweakPrice(AGamma []*uint256.Int, _xp []*uint256.Int, i int, pI *uint256.Int, newD *uint256.Int) error {
	var nCoins = len(_xp)
	var nCoinsBi uint256.Int
	nCoinsBi.SetUint64(uint64(nCoins))
	var priceOracle = make([]*uint256.Int, nCoins-1)
	var lastPrices = make([]*uint256.Int, nCoins-1)
	var priceScale = make([]*uint256.Int, nCoins-1)
	var xp = make([]*uint256.Int, nCoins)
	var pNew = make([]*uint256.Int, nCoins-1)

	var packedPrices uint256.Int
	packedPrices.Set(t.PriceOraclePacked)
	for k := 0; k < nCoins-1; k += 1 {
		var price uint256.Int
		price.And(&packedPrices, PriceMask)
		priceOracle[k] = &price
		packedPrices.Rsh(&packedPrices, PriceSize)
	}
	var lastPricesTimestamp = t.LastPricesTimestamp
	packedPrices.Set(t.LastPricesPacked)
	for k := 0; k < nCoins-1; k += 1 {
		var price uint256.Int
		price.And(&packedPrices, PriceMask)
		lastPrices[k] = &price
		packedPrices.Rsh(&packedPrices, PriceSize)
	}
	var blockTimestamp = time.Now().Unix()
	if lastPricesTimestamp < blockTimestamp {
		var maHalfTime = t.MaHalfTime
		var elapsed uint256.Int
		elapsed.SetUint64(uint64(blockTimestamp - lastPricesTimestamp))
		elapsed.Mul(&elapsed, big256.BONE)
		elapsed.Div(&elapsed, maHalfTime)
		var alpha, _ = halfpow(&elapsed, big256.TenPow(10))
		packedPrices.SetUint64(0)
		for k := 0; k < nCoins-1; k += 1 {
			var oneMinusAlpha uint256.Int
			oneMinusAlpha.Sub(big256.BONE, alpha)
			var updated uint256.Int
			updated.Mul(lastPrices[k], &oneMinusAlpha)
			var tmp uint256.Int
			tmp.Mul(priceOracle[k], alpha)
			updated.Add(&updated, &tmp)
			updated.Div(&updated, big256.BONE)
			priceOracle[k] = &updated
		}
		for k := 0; k < nCoins-1; k += 1 {
			packedPrices.Lsh(&packedPrices, PriceSize)
			packedPrices.Or(priceOracle[nCoins-2-k], &packedPrices)
		}
		var nextPriceOraclePacked uint256.Int
		nextPriceOraclePacked.Set(&packedPrices)
		t.PriceOraclePacked = &nextPriceOraclePacked
		t.LastPricesTimestamp = blockTimestamp
	}
	var DUnadjusted = newD
	if newD.Cmp(big256.U0) == 0 {
		DUnadjusted, _ = newtonD(AGamma[0], AGamma[1], _xp)
	}
	packedPrices.Set(t.PriceScalePacked)
	for k := 0; k < nCoins-1; k += 1 {
		var price uint256.Int
		price.And(&packedPrices, PriceMask)
		priceScale[k] = &price
		packedPrices.Rsh(&packedPrices, PriceSize)
	}
	if pI.Cmp(big256.U0) > 0 {
		if i > 0 {
			lastPrices[i-1] = pI
		} else {
			for k := 0; k < nCoins-1; k += 1 {
				var updated uint256.Int
				updated.Mul(lastPrices[k], big256.BONE)
				updated.Div(&updated, pI)
				lastPrices[k] = &updated
			}
		}
	} else {
		var __xp = make([]*uint256.Int, nCoins)
		for k := 0; k < nCoins; k += 1 {
			var copied uint256.Int
			copied.Set(_xp[k])
			__xp[k] = &copied
		}
		var dxPrice uint256.Int
		dxPrice.Div(__xp[0], big256.TenPow(6))
		var adjustedXP0 uint256.Int
		adjustedXP0.Add(__xp[0], &dxPrice)
		__xp[0] = &adjustedXP0
		for k := 0; k < nCoins-1; k += 1 {
			var temp, err = newtonY(AGamma[0], AGamma[1], __xp, DUnadjusted, k+1)
			if err != nil {
				return err
			}
			var denominator uint256.Int
			denominator.Sub(_xp[k+1], temp)
			var updated uint256.Int
			updated.Mul(priceScale[k], &dxPrice)
			updated.Div(&updated, &denominator)
			lastPrices[k] = &updated
		}
	}
	packedPrices.SetUint64(0)
	for k := 0; k < nCoins-1; k += 1 {
		packedPrices.Lsh(&packedPrices, PriceSize)
		packedPrices.Or(lastPrices[nCoins-2-k], &packedPrices)
	}
	var nextLastPricesPacked uint256.Int
	nextLastPricesPacked.Set(&packedPrices)
	t.LastPricesPacked = &nextLastPricesPacked

	var totalSupply = t.LpSupply
	var oldXcpProfit = t.XcpProfit
	var oldVirtualPrice uint256.Int
	oldVirtualPrice.Set(t.VirtualPrice)

	var xp0 uint256.Int
	xp0.Div(DUnadjusted, &nCoinsBi)
	xp[0] = &xp0
	for k := 0; k < nCoins-1; k += 1 {
		var adjusted uint256.Int
		adjusted.Mul(DUnadjusted, big256.BONE)
		var denominator uint256.Int
		denominator.Mul(&nCoinsBi, priceScale[k])
		adjusted.Div(&adjusted, &denominator)
		xp[k+1] = &adjusted
	}
	var xcpProfit uint256.Int
	xcpProfit.Set(big256.BONE)
	var virtualPrice uint256.Int
	virtualPrice.Set(big256.BONE)
	if oldVirtualPrice.Cmp(big256.U0) > 0 {
		var xcp, err = geometricMean(xp, true)
		if err != nil {
			return err
		}
		virtualPrice.Mul(big256.BONE, xcp)
		virtualPrice.Div(&virtualPrice, totalSupply)
		xcpProfit.Mul(oldXcpProfit, &virtualPrice)
		xcpProfit.Div(&xcpProfit, &oldVirtualPrice)
		var aGammaTime = t.FutureTime
		if virtualPrice.Cmp(&oldVirtualPrice) < 0 && aGammaTime == 0 {
			return errors.New("loss")
		}
		if aGammaTime == 1 {
			t.FutureTime = 0
		}
	}
	t.XcpProfit = &xcpProfit
	var needsAdjustment = t.NotAdjusted
	var thresholdLeft uint256.Int
	thresholdLeft.Mul(&virtualPrice, big256.U2)
	thresholdLeft.Sub(&thresholdLeft, big256.BONE)
	var thresholdRight uint256.Int
	thresholdRight.Mul(big256.U2, t.AllowedExtraProfit)
	thresholdRight.Add(&xcpProfit, &thresholdRight)
	if thresholdLeft.Cmp(&thresholdRight) > 0 {
		needsAdjustment = true
		t.NotAdjusted = true
	}
	if needsAdjustment {
		var adjustmentStep = t.AdjustmentStep
		var norm uint256.Int
		for k := 0; k < nCoins-1; k += 1 {
			var ratio uint256.Int
			ratio.Mul(priceOracle[k], big256.BONE)
			ratio.Div(&ratio, priceScale[k])
			if ratio.Cmp(big256.BONE) > 0 {
				ratio.Sub(&ratio, big256.BONE)
			} else {
				ratio.Sub(big256.BONE, &ratio)
			}
			var ratioSquared uint256.Int
			ratioSquared.Mul(&ratio, &ratio)
			norm.Add(&norm, &ratioSquared)
		}
		var adjustmentStepSquared uint256.Int
		adjustmentStepSquared.Mul(adjustmentStep, adjustmentStep)
		if norm.Cmp(&adjustmentStepSquared) > 0 && oldVirtualPrice.Cmp(big256.U0) > 0 {
			var normalizedNorm uint256.Int
			normalizedNorm.Div(&norm, big256.BONE)
			var temp, err = sqrtInt(&normalizedNorm)
			if err != nil {
				return err
			}
			norm.Set(temp)
			for k := 0; k < nCoins-1; k += 1 {
				var normMinusStep uint256.Int
				normMinusStep.Sub(&norm, adjustmentStep)
				var updated uint256.Int
				updated.Mul(priceScale[k], &normMinusStep)
				var tmp uint256.Int
				tmp.Mul(adjustmentStep, priceOracle[k])
				updated.Add(&updated, &tmp)
				updated.Div(&updated, &norm)
				pNew[k] = &updated
			}
			for k := 0; k < nCoins; k += 1 {
				var copied uint256.Int
				copied.Set(_xp[k])
				xp[k] = &copied
			}
			for k := 0; k < nCoins-1; k += 1 {
				var adjusted uint256.Int
				adjusted.Mul(_xp[k+1], pNew[k])
				adjusted.Div(&adjusted, priceScale[k])
				xp[k+1] = &adjusted
			}
			D, err := newtonD(AGamma[0], AGamma[1], xp)
			if err != nil {
				return err
			}

			var adjustedXP0 uint256.Int
			adjustedXP0.Div(D, &nCoinsBi)
			xp[0] = &adjustedXP0
			for k := 0; k < nCoins-1; k += 1 {
				var adjusted uint256.Int
				adjusted.Mul(D, big256.BONE)
				var denominator uint256.Int
				denominator.Mul(&nCoinsBi, pNew[k])
				adjusted.Div(&adjusted, &denominator)
				xp[k+1] = &adjusted
			}
			temp, err = geometricMean(xp, true)
			if err != nil {
				return err
			}
			var adjustedVirtualPrice uint256.Int
			adjustedVirtualPrice.Mul(big256.BONE, temp)
			adjustedVirtualPrice.Div(&adjustedVirtualPrice, totalSupply)
			var gainThreshold uint256.Int
			gainThreshold.Mul(big256.U2, &adjustedVirtualPrice)
			gainThreshold.Sub(&gainThreshold, big256.BONE)
			if adjustedVirtualPrice.Cmp(big256.BONE) > 0 && gainThreshold.Cmp(&xcpProfit) > 0 {
				packedPrices.SetUint64(0)
				for k := 0; k < nCoins-1; k += 1 {
					packedPrices.Lsh(&packedPrices, PriceSize)
					packedPrices.Or(pNew[nCoins-2-k], &packedPrices)
				}
				var nextPriceScalePacked uint256.Int
				nextPriceScalePacked.Set(&packedPrices)
				t.PriceScalePacked = &nextPriceScalePacked
				t.D = D
				t.VirtualPrice = &adjustedVirtualPrice
				return nil
			} else {
				t.NotAdjusted = false
			}
		}
	}
	t.D = DUnadjusted
	t.VirtualPrice = &virtualPrice
	return nil
}

func getCryptoFee(minFee, maxFee, gamma, xp0, xp1 *uint256.Int) *uint256.Int {
	var f uint256.Int
	f.Add(xp0, xp1)

	// f = gamma.mul(ETHER).div(
	//     gamma.add(ETHER).sub(ETHER.mul(4).mul(xp0).div(f).mul(xp1).div(f))
	// );

	// f = new(uint256.Int).Div(
	// 	new(uint256.Int).Mul(
	// 		gamma,
	// 		constant.BONE,
	// 	),
	// 	new(uint256.Int).Sub(
	// 		new(uint256.Int).Add(
	// 			gamma,
	// 			constant.BONE,
	// 		),
	// 		new(uint256.Int).Div(
	// 			new(uint256.Int).Mul(
	// 				new(uint256.Int).Div(
	// 					new(uint256.Int).Mul(
	// 						new(uint256.Int).Mul(
	// 							constant.BONE,
	// 							constant.Four,
	// 						),
	// 						xp0,
	// 					),
	// 					f,
	// 				),
	// 				xp1,
	// 			),
	// 			f,
	// 		),
	// 	),
	// )
	var f1 uint256.Int
	f1.Set(&f)
	f.Mul(big256.BONE, big256.U4).Mul(&f, xp0).Div(&f, &f1).Mul(&f, xp1).Div(&f, &f1)
	var feeDenominator uint256.Int
	feeDenominator.Add(gamma, big256.BONE)
	feeDenominator.Sub(&feeDenominator, &f)
	var feeNumerator uint256.Int
	feeNumerator.Mul(gamma, big256.BONE)
	f.Div(&feeNumerator, &feeDenominator)

	// const fee = minFee.mul(f).add(maxFee.mul(ETHER.sub(f))).div(ETHER);
	// fee := new(uint256.Int).Div(
	// 	new(uint256.Int).Add(
	// 		new(uint256.Int).Mul(
	// 			minFee, f,
	// 		),
	// 		new(uint256.Int).Mul(
	// 			maxFee,
	// 			new(uint256.Int).Sub(
	// 				constant.BONE, f,
	// 			),
	// 		),
	// 	),
	// 	constant.BONE,
	// )
	var fee uint256.Int
	fee.Sub(big256.BONE, &f).Mul(maxFee, &fee)
	var minFeeComponent uint256.Int
	minFeeComponent.Mul(minFee, &f)
	fee.Add(&fee, &minFeeComponent)
	fee.Div(&fee, big256.BONE)
	return &fee
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

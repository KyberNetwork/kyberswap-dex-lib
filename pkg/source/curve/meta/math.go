package meta

import (
	"math/big"
	"time"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (t *PoolSimulator) _xp_mem(_balances []*big.Int, vPrice *big.Int) ([]*big.Int, error) {
	var nCoins = len(_balances)
	var ret = []*big.Int{t.RateMultiplier, vPrice}
	for i := 0; i < nCoins; i += 1 {
		ret[i] = new(big.Int).Div(new(big.Int).Mul(ret[i], _balances[i]), Precision)
	}
	return ret, nil
}

func (t *PoolSimulator) _get_D(xp []*big.Int, a *big.Int) (*big.Int, error) {
	var numTokens = len(xp)
	var s = new(big.Int).SetInt64(0)
	for i := 0; i < numTokens; i++ {
		s = new(big.Int).Add(s, xp[i])
	}
	if s.Sign() == 0 {
		return s, nil
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var prevD *big.Int
	var d = new(big.Int).Set(s)
	var nA = new(big.Int).Mul(a, numTokensBI)
	for i := 0; i < MaxLoopLimit; i++ {
		var dP = new(big.Int).Set(d)
		for j := 0; j < numTokens; j++ {
			if xp[j].Cmp(constant.ZeroBI) == 0 {
				return nil, ErrDenominatorZero
			}
			dP = new(big.Int).Div(
				new(big.Int).Mul(dP, d),
				new(big.Int).Mul(xp[j], numTokensBI),
			)
		}
		prevD = d
		d = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Add(
					new(big.Int).Div(new(big.Int).Mul(nA, s), t.APrecision),
					new(big.Int).Mul(dP, numTokensBI),
				),
				d,
			),
			new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(nA, t.APrecision), d), t.APrecision),
				new(big.Int).Mul(dP, big.NewInt(int64(numTokens+1))),
			),
		)
		if new(big.Int).Sub(d, prevD).CmpAbs(big.NewInt(1)) <= 0 {
			return d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

//
//func (t *PoolSimulator) _get_D_mem(balances []*big.Int, amp *big.Int) (*big.Int, error) {
//	var xp = t._xp_mem(balances)
//	return t._get_D(xp, amp)
//}

func (t *PoolSimulator) _A() *big.Int {
	var t1 = t.FutureATime
	var a1 = t.FutureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = t.InitialATime
		var a0 = t.InitialA
		if a1.Cmp(a0) > 0 {
			return new(big.Int).Add(
				a0,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Sub(a1, a0),
						new(big.Int).SetInt64(now-t0),
					),
					new(big.Int).SetInt64(t1-t0),
				),
			)
		} else {
			return new(big.Int).Sub(
				a0,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Sub(a0, a1),
						new(big.Int).SetInt64(now-t0),
					),
					new(big.Int).SetInt64(t1-t0),
				),
			)
		}
	}
	return a1
}

func (t *PoolSimulator) A() *big.Int {
	var a = t._A()
	return new(big.Int).Div(a, t.APrecision)
}

func (t *PoolSimulator) APrecise() *big.Int {
	return t._A()
}

func (t *PoolSimulator) _get_y(
	i int,
	j int,
	x *big.Int,
	xp []*big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if i == j {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if i >= numTokens && j >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var nCoins = big.NewInt(int64(numTokens))
	var a = t._A()
	var d, err = t._get_D(xp, a)
	if err != nil {
		return nil, err
	}
	var c = new(big.Int).Set(d)
	var s = constant.ZeroBI
	var Ann = new(big.Int).Mul(a, nCoins)
	var _x *big.Int
	var y_prev *big.Int
	for _i := 0; _i < numTokens; _i++ {
		if _i == i {
			_x = x
		} else if _i != j {
			_x = xp[_i]
		} else {
			continue
		}
		s = new(big.Int).Add(s, _x)
		c = new(big.Int).Div(
			new(big.Int).Mul(c, d),
			new(big.Int).Mul(_x, nCoins),
		)
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).Mul(c, d), t.APrecision),
		new(big.Int).Mul(Ann, nCoins),
	)
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(new(big.Int).Mul(d, t.APrecision), Ann),
	)
	var y = new(big.Int).Set(d)
	for _i := 0; _i < MaxLoopLimit; _i++ {
		y_prev = new(big.Int).Set(y)
		y = new(big.Int).Div(
			new(big.Int).Add(new(big.Int).Mul(y, y), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(y, big.NewInt(2)), b), d),
		)
		if new(big.Int).Sub(y, y_prev).CmpAbs(constant.One) <= 0 {
			return y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

func (t *PoolSimulator) _get_dy_mem(i int, j int, _dx *big.Int, _balances []*big.Int) (*big.Int, *big.Int, error) {
	vPrice, _, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, nil, err
	}
	var rates = []*big.Int{t.RateMultiplier, vPrice}
	xp, err := t._xp_mem(_balances, vPrice)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(_dx, rates[i]), Precision))
	y, err := t._get_y(i, j, x, xp)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[j], y), constant.One)
	var fee = new(big.Int).Div(new(big.Int).Mul(t.GetInfo().SwapFee, dy), FeeDenominator)
	dy = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(dy, fee), Precision), rates[j])
	return dy, fee, nil
}

func (t *PoolSimulator) GetDy(
	i int,
	j int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	return t._get_dy_mem(i, j, dx, t.Info.Reserves)
}

func (t *PoolSimulator) GetDyUnderlying(i int, j int, _dx *big.Int) (*big.Int, *big.Int, error) {
	var nCoins = len(t.Info.Tokens)
	var maxCoin = nCoins - 1
	var baseNCoins = len(t.basePool.GetInfo().Tokens)
	vPrice, D, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, nil, err
	}
	var rates = []*big.Int{t.RateMultiplier, vPrice}
	xp, err := t._xp_mem(t.Info.Reserves, vPrice)
	if err != nil {
		return nil, nil, err
	}
	var base_i = i - maxCoin
	var base_j = j - maxCoin
	var meta_i = maxCoin
	var meta_j = maxCoin
	if base_i < 0 {
		meta_i = i
	}
	if base_j < 0 {
		meta_j = j
	}
	var x *big.Int
	if base_i < 0 {
		//x = new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(_dx, rates[i]), Precision))
		x = new(big.Int).Add(xp[i], new(big.Int).Mul(_dx, new(big.Int).Div(rates[i], Precision)))
	} else {
		if base_j < 0 {
			var base_inputs = make([]*big.Int, baseNCoins)
			for k := 0; k < baseNCoins; k += 1 {
				base_inputs[k] = constant.ZeroBI
			}
			base_inputs[base_i] = _dx
			var temp, err = t.basePool.CalculateTokenAmount(base_inputs, true)
			if err != nil {
				return nil, nil, err
			}
			x = new(big.Int).Div(new(big.Int).Mul(temp, rates[maxCoin]), Precision)
			x = new(big.Int).Sub(x, new(big.Int).Div(new(big.Int).Mul(x, t.basePool.GetInfo().SwapFee), new(big.Int).Mul(constant.Two, FeeDenominator)))
			x = new(big.Int).Add(x, xp[maxCoin])
		} else {
			return t.basePool.GetDy(base_i, base_j, _dx, D)
		}
	}
	y, err := t._get_y(meta_i, meta_j, x, xp)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[meta_j], y), constant.One)
	var dy_fee = new(big.Int).Div(new(big.Int).Mul(t.Info.SwapFee, dy), FeeDenominator)
	dy = new(big.Int).Sub(dy, dy_fee)
	//dy = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(dy, dy_fee), Precision), rates[meta_j])
	//dy_fee = new(big.Int).Div(new(big.Int).Mul(dy_fee, Precision), rates[meta_j])
	if base_j < 0 {
		dy = new(big.Int).Div(new(big.Int).Mul(dy, Precision), rates[j])
		dy_fee = new(big.Int).Div(new(big.Int).Mul(dy_fee, Precision), rates[j])
	} else {
		dy, dy_fee, err = t.basePool.CalculateWithdrawOneCoin(new(big.Int).Div(new(big.Int).Mul(dy, Precision), rates[maxCoin]), base_j)
	}
	return dy, dy_fee, err
}

func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	var nCoins = len(t.Info.Tokens)
	vPrice, _, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, err
	}
	var rates = []*big.Int{
		t.RateMultiplier,
		vPrice,
	}
	var old_balances = make([]*big.Int, nCoins)
	for k := 0; k < nCoins; k += 1 {
		old_balances[k] = t.Info.Reserves[k]
	}
	xp, err := t._xp_mem(old_balances, vPrice)
	if err != nil {
		return nil, err
	}
	var x = new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(dx, rates[i]), Precision))
	y, err := t._get_y(i, j, x, xp)
	if err != nil {
		return nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[j], y), constant.One)
	var dy_fee = new(big.Int).Div(new(big.Int).Mul(dy, t.Info.SwapFee), FeeDenominator)
	dy = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(dy, dy_fee), Precision), rates[j])
	var dy_admin_fee = new(big.Int).Div(new(big.Int).Mul(dy_fee, t.AdminFee), FeeDenominator)
	dy_admin_fee = new(big.Int).Div(new(big.Int).Mul(dy_admin_fee, Precision), rates[j])
	t.Info.Reserves[i] = new(big.Int).Add(old_balances[i], dx)
	t.Info.Reserves[j] = new(big.Int).Sub(new(big.Int).Sub(old_balances[j], dy), dy_admin_fee)
	return dy, nil
}

func (t *PoolSimulator) ExchangeUnderlying(i int, j int, dx *big.Int) (*big.Int, error) {
	var nCoins = len(t.Info.Tokens)
	var maxCoins = nCoins - 1
	var baseNCoins = len(t.basePool.GetInfo().Tokens)
	vPrice, _, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, err
	}
	var rates = []*big.Int{
		t.RateMultiplier,
		vPrice,
	}
	var base_i = i - maxCoins
	var base_j = j - maxCoins
	var meta_i = maxCoins
	var meta_j = maxCoins
	if base_i < 0 {
		meta_i = i
	}
	if base_j < 0 {
		meta_j = j
	}
	var dy *big.Int
	if base_i < 0 || base_j < 0 {
		var old_balances = make([]*big.Int, nCoins)
		for k := 0; k < nCoins; k += 1 {
			old_balances[k] = t.Info.Reserves[k]
		}
		xp, err := t._xp_mem(old_balances, vPrice)
		if err != nil {
			return nil, err
		}
		var x *big.Int
		if base_i < 0 {
			x = new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(dx, rates[i]), Precision))
		} else {
			var base_inputs = make([]*big.Int, baseNCoins)
			for k := 0; k < baseNCoins; k += 1 {
				base_inputs[k] = constant.ZeroBI
			}
			base_inputs[base_i] = dx
			var temp, err = t.basePool.AddLiquidity(base_inputs)
			if err != nil {
				return nil, err
			}
			dx = temp
			x = new(big.Int).Div(new(big.Int).Mul(dx, rates[maxCoins]), Precision)
			x = new(big.Int).Add(x, xp[maxCoins])
		}
		y, err := t._get_y(meta_i, meta_j, x, xp)
		if err != nil {
			return nil, err
		}
		dy = new(big.Int).Sub(new(big.Int).Sub(xp[meta_j], y), constant.One)
		var dy_fee = new(big.Int).Div(new(big.Int).Mul(dy, t.Info.SwapFee), FeeDenominator)
		dy = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(dy, dy_fee), Precision), rates[meta_j])
		var dy_admin_fee = new(big.Int).Div(new(big.Int).Mul(dy_fee, t.AdminFee), FeeDenominator)
		dy_admin_fee = new(big.Int).Div(new(big.Int).Mul(dy_admin_fee, Precision), rates[meta_j])
		t.Info.Reserves[meta_i] = new(big.Int).Add(old_balances[meta_i], dx)
		t.Info.Reserves[meta_j] = new(big.Int).Sub(new(big.Int).Sub(old_balances[meta_j], dy), dy_admin_fee)

		if base_j >= 0 {
			return t.basePool.RemoveLiquidityOneCoin(dy, base_j)
		}
	} else {
		return nil, ErrBasePoolExchangeNotSupported
	}
	return dy, nil
}

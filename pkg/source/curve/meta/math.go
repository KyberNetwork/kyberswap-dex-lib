package meta

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (t *PoolSimulator) _xp_mem(_balances []*uint256.Int, vPrice *uint256.Int) ([]*uint256.Int, error) {
	var nCoins = len(_balances)
	var ret = []*uint256.Int{t.RateMultiplier, vPrice}
	for i := range nCoins {
		ret[i], _ = new(uint256.Int).MulDivOverflow(ret[i], _balances[i], Precision)
	}
	return ret, nil
}

func (t *PoolSimulator) _get_D(xp []*uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	numTokens := len(xp)
	var s uint256.Int
	for i := range numTokens {
		s.Add(&s, xp[i])
	}
	if s.IsZero() {
		return &s, nil
	}
	var uNumTokens, uNumTokensPlusOne uint256.Int
	uNumTokens.SetUint64(uint64(numTokens))
	uNumTokensPlusOne.SetUint64(uint64(numTokens + 1))
	var prevD, d, nA uint256.Int
	d.Set(&s)
	nA.Mul(a, &uNumTokens)
	var dP, tmp, tmp2 uint256.Int
	for range MaxLoopLimit {
		dP.Set(&d)
		for j := range numTokens {
			if xp[j].IsZero() {
				return nil, ErrDenominatorZero
			}
			dP.MulDivOverflow(&dP, &d, tmp.Mul(xp[j], &uNumTokens))
		}
		prevD.Set(&d)
		tmp.MulDivOverflow(&nA, &s, t.APrecision)
		tmp.Mul(tmp.Add(&tmp, tmp2.Mul(&dP, &uNumTokens)), &d)
		d.MulDivOverflow(tmp2.Sub(&nA, t.APrecision), &d, t.APrecision)
		d.Div(&tmp, d.Add(&d, tmp2.Mul(&dP, &uNumTokensPlusOne)))
		if tmp.Abs(tmp.Sub(&d, &prevD)).CmpUint64(1) <= 0 {
			return &d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

func (t *PoolSimulator) _A() *uint256.Int {
	t1, a1 := t.FutureATime, t.FutureA
	now := time.Now().Unix()
	if t1 <= now {
		return a1
	}
	t0, a0 := t.InitialATime, t.InitialA
	var tmp, tmp2 uint256.Int
	return tmp.Add(
		a0,
		tmp.SDiv(
			tmp.Mul(
				tmp.Sub(a1, a0),
				tmp2.SetUint64(uint64(now-t0)),
			),
			tmp2.SetUint64(uint64(t1-t0)),
		),
	)
}

func (t *PoolSimulator) A() *uint256.Int {
	return new(uint256.Int).Div(t._A(), t.APrecision)
}

func (t *PoolSimulator) APrecise() *uint256.Int {
	return t._A()
}

func (t *PoolSimulator) _get_y(
	i int,
	j int,
	x *uint256.Int,
	xp []*uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if i == j {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if i >= numTokens && j >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var nCoins = uint256.NewInt(uint64(numTokens))
	a := t._A()
	d, err := t._get_D(xp, a)
	if err != nil {
		return nil, err
	}
	var c, s, Ann, y_prev, b, y, tmp uint256.Int
	var _x *uint256.Int
	c.Set(d)
	Ann.Mul(a, nCoins)
	for _i := range numTokens {
		if _i == i {
			_x = x
		} else if _i != j {
			_x = xp[_i]
		} else {
			continue
		}
		s.Add(&s, _x)
		c.MulDivOverflow(&c, d, tmp.Mul(_x, nCoins))
	}
	c.MulDivOverflow(c.Mul(&c, d), t.APrecision, tmp.Mul(&Ann, nCoins))
	b.MulDivOverflow(d, t.APrecision, &Ann)
	b.Add(&s, &b)
	y.Set(d)
	for range MaxLoopLimit {
		y_prev.Set(&y)
		tmp.Sub(tmp.Add(tmp.Mul(&y, big256.U2), &b), d)
		y.Div(y.Add(y.Mul(&y, &y), &c), &tmp)
		if tmp.Abs(tmp.Sub(&y, &y_prev)).CmpUint64(1) <= 0 {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

func (t *PoolSimulator) _get_dy_mem(i int, j int, _dx *uint256.Int, _balances []*uint256.Int) (*uint256.Int,
	*uint256.Int, error) {
	bVPrice, _, err := t.basePool.GetVirtualPrice()
	vPrice, _ := uint256.FromBig(bVPrice)
	if err != nil {
		return nil, nil, err
	}
	rates := []*uint256.Int{t.RateMultiplier, vPrice}
	xp, err := t._xp_mem(_balances, vPrice)
	if err != nil {
		return nil, nil, err
	}
	var x, dy, fee uint256.Int
	x.MulDivOverflow(_dx, rates[i], Precision)
	x.Add(xp[i], &x)
	y, err := t._get_y(i, j, &x, xp)
	if err != nil {
		return nil, nil, err
	}
	if _, overflow := dy.SubOverflow(xp[j], y); overflow {
		return nil, nil, number.ErrUnderflow
	}
	if _, overflow := dy.SubOverflow(&dy, big256.U1); overflow {
		return nil, nil, number.ErrUnderflow
	}
	fee.MulDivOverflow(t.SwapFee, &dy, FeeDenominator)
	dy.MulDivOverflow(dy.Sub(&dy, &fee), Precision, rates[j])
	return &dy, &fee, nil
}

func (t *PoolSimulator) GetDy(
	i int,
	j int,
	dx *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	return t._get_dy_mem(i, j, dx, t.Reserves)
}

func (t *PoolSimulator) GetDyUnderlying(i int, j int, _dx *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	nCoins := len(t.Info.Tokens)
	maxCoin := nCoins - 1
	baseNCoins := len(t.basePool.GetInfo().Tokens)
	bVPrice, D, err := t.basePool.GetVirtualPrice()
	vPrice, _ := uint256.FromBig(bVPrice)
	if err != nil {
		return nil, nil, err
	}
	rates := []*uint256.Int{t.RateMultiplier, vPrice}
	xp, err := t._xp_mem(t.Reserves, vPrice)
	if err != nil {
		return nil, nil, err
	}
	baseI := i - maxCoin
	baseJ := j - maxCoin
	metaI := maxCoin
	metaJ := maxCoin
	if baseI < 0 {
		metaI = i
	}
	if baseJ < 0 {
		metaJ = j
	}
	var x uint256.Int
	if baseI < 0 {
		x.Add(xp[i], x.Mul(_dx, x.Div(rates[i], Precision)))
	} else if baseJ < 0 {
		baseInputs := make([]*big.Int, baseNCoins)
		for k := range baseNCoins {
			baseInputs[k] = bignumber.ZeroBI
		}
		baseInputs[baseI] = _dx.ToBig()
		tokenAmt, err := t.basePool.CalculateTokenAmount(baseInputs, true)
		if err != nil {
			return nil, nil, err
		}
		var tmp, tmp2 uint256.Int
		tmp.SetFromBig(tokenAmt)
		x.MulDivOverflow(&tmp, rates[maxCoin], Precision)
		tmp.SetFromBig(t.basePool.GetInfo().SwapFee)
		// x = new(uint256.Int).Sub(x,
		// 	new(uint256.Int).Div(new(uint256.Int).Mul(x, &tmp), new(uint256.Int).Mul(big256.U2, FeeDenominator)))
		// x = new(uint256.Int).Add(x, xp[maxCoin])
		tmp.MulDivOverflow(&x, &tmp, tmp2.Mul(big256.U2, FeeDenominator))
		x.Add(x.Sub(&x, &tmp), xp[maxCoin])
	} else {
		bAmtOut, bFee, err := t.basePool.GetDy(baseI, baseJ, _dx.ToBig(), D)
		if err != nil {
			return nil, nil, err
		}
		amtOut, _ := uint256.FromBig(bAmtOut)
		fee, _ := uint256.FromBig(bFee)
		return amtOut, fee, nil
	}
	y, err := t._get_y(metaI, metaJ, &x, xp)
	if err != nil {
		return nil, nil, err
	}
	var dy, dyFee uint256.Int
	if _, overflow := dy.SubOverflow(xp[metaJ], y); overflow {
		return nil, nil, number.ErrUnderflow
	}
	if _, overflow := dy.SubOverflow(&dy, big256.U1); overflow {
		return nil, nil, number.ErrUnderflow
	}
	dyFee.MulDivOverflow(t.SwapFee, &dy, FeeDenominator)
	dy.Sub(&dy, &dyFee)
	if baseJ < 0 {
		dy.MulDivOverflow(&dy, Precision, rates[j])
		dyFee.MulDivOverflow(&dyFee, Precision, rates[j])
	} else {
		dy.MulDivOverflow(&dy, Precision, rates[maxCoin])
		bDy, bdyFee, err := t.basePool.CalculateWithdrawOneCoin(dy.ToBig(), baseJ)
		if err != nil {
			return nil, nil, err
		}
		dy.SetFromBig(bDy)
		dyFee.SetFromBig(bdyFee)
	}
	return &dy, &dyFee, err
}

func (t *PoolSimulator) Exchange(i int, j int, dx *uint256.Int) (*uint256.Int, error) {
	nCoins := len(t.Info.Tokens)
	bVPrice, _, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, err
	}
	vPrice, _ := uint256.FromBig(bVPrice)
	rates := []*uint256.Int{t.RateMultiplier, vPrice}
	oldBalances := make([]*uint256.Int, nCoins)
	for k := range nCoins {
		oldBalances[k] = t.Reserves[k]
	}
	xp, err := t._xp_mem(oldBalances, vPrice)
	if err != nil {
		return nil, err
	}
	var x uint256.Int
	x.MulDivOverflow(dx, rates[i], Precision)
	x.Add(xp[i], &x)
	y, err := t._get_y(i, j, &x, xp)
	if err != nil {
		return nil, err
	}
	var dy, dyFee uint256.Int
	if _, overflow := dy.SubOverflow(xp[j], y); overflow {
		return nil, number.ErrUnderflow
	}
	if _, overflow := dy.SubOverflow(&dy, big256.U1); overflow {
		return nil, number.ErrUnderflow
	}
	dyFee.MulDivOverflow(&dy, t.SwapFee, FeeDenominator)
	dy.MulDivOverflow(dy.Sub(&dy, &dyFee), Precision, rates[j])
	dyAdminFee, _ := dy.MulDivOverflow(&dyFee, t.AdminFee, FeeDenominator)
	dyAdminFee.MulDivOverflow(dyAdminFee, Precision, rates[j])
	t.Reserves[j] = x.Sub(x.Sub(oldBalances[j], &dy), dyAdminFee)
	t.Reserves[i] = dyAdminFee.Add(oldBalances[i], dx)
	t.Info.Reserves[i], t.Info.Reserves[j] = t.Reserves[i].ToBig(), t.Reserves[j].ToBig()
	return &dy, nil
}

func (t *PoolSimulator) ExchangeUnderlying(i int, j int, dx *uint256.Int) (*uint256.Int, error) {
	nCoins := len(t.Info.Tokens)
	maxCoins := nCoins - 1
	baseNCoins := len(t.basePool.GetInfo().Tokens)
	bVPrice, _, err := t.basePool.GetVirtualPrice()
	if err != nil {
		return nil, err
	}
	vPrice, _ := uint256.FromBig(bVPrice)
	rates := []*uint256.Int{t.RateMultiplier, vPrice}
	baseI, baseJ, metaI := i-maxCoins, j-maxCoins, maxCoins
	metaJ := maxCoins
	if baseI < 0 {
		metaI = i
	}
	if baseJ < 0 {
		metaJ = j
	}
	var dy uint256.Int
	if baseI < 0 || baseJ < 0 {
		oldBalances := make([]*uint256.Int, nCoins)
		for k := range nCoins {
			oldBalances[k] = t.Reserves[k]
		}
		xp, err := t._xp_mem(oldBalances, vPrice)
		if err != nil {
			return nil, err
		}
		var x uint256.Int
		if baseI < 0 {
			x.MulDivOverflow(dx, rates[i], Precision)
			x.Add(xp[i], &x)
		} else {
			baseInputs := make([]*big.Int, baseNCoins)
			for k := range baseNCoins {
				baseInputs[k] = bignumber.ZeroBI
			}
			baseInputs[baseI] = dx.ToBig()
			temp, err := t.basePool.AddLiquidity(baseInputs)
			if err != nil {
				return nil, err
			}
			dx, _ = uint256.FromBig(temp)
			x.MulDivOverflow(dx, rates[maxCoins], Precision)
			x.Add(&x, xp[maxCoins])
		}
		y, err := t._get_y(metaI, metaJ, &x, xp)
		if err != nil {
			return nil, err
		}
		var dyFee uint256.Int
		if _, overflow := dy.SubOverflow(xp[metaJ], y); overflow {
			return nil, number.ErrUnderflow
		}
		if _, overflow := dy.SubOverflow(&dy, big256.U1); overflow {
			return nil, number.ErrUnderflow
		}
		dyFee.MulDivOverflow(&dy, t.SwapFee, FeeDenominator)
		dy.MulDivOverflow(dy.Sub(&dy, &dyFee), Precision, rates[metaJ])
		dyAdminFee, _ := dyFee.MulDivOverflow(&dyFee, t.AdminFee, FeeDenominator)
		dyAdminFee.MulDivOverflow(dyAdminFee, Precision, rates[metaJ])
		t.Reserves[metaJ] = x.Sub(x.Sub(oldBalances[metaJ], &dy), dyAdminFee)
		t.Reserves[metaI] = dyAdminFee.Add(oldBalances[metaI], dx)
		t.Info.Reserves[metaI], t.Info.Reserves[metaJ] = t.Reserves[metaI].ToBig(), t.Reserves[metaJ].ToBig()

		if baseJ >= 0 {
			bDy, err := t.basePool.RemoveLiquidityOneCoin(dy.ToBig(), baseJ)
			if err != nil {
				return nil, err
			}
			dy.SetFromBig(bDy)
		}
	} else {
		return nil, ErrBasePoolExchangeNotSupported
	}
	return &dy, nil
}

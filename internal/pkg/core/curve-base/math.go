package curveBase

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	errors "github.com/KyberNetwork/router-service/internal/pkg/core/errors"
)

func _xpMem(
	balances []*big.Int,
	rates []*big.Int,
) ([]*big.Int, error) {
	var numTokens = len(balances)
	if numTokens != len(rates) {
		return nil, errors.ErrBalancesMustMatchMultipliers
	}
	xp := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		xp[i] = new(big.Int).Div(new(big.Int).Mul(rates[i], balances[i]), Precision)
	}
	return xp, nil
}

func (t *Pool) _xp() []*big.Int {
	var nTokens = len(t.Info.Tokens)
	result := make([]*big.Int, nTokens)
	for i := 0; i < nTokens; i += 1 {
		result[i] = new(big.Int).Div(new(big.Int).Mul(t.Rates[i], t.Info.Reserves[i]), Precision)
	}
	return result
}

func (t *Pool) get_D_mem(balances []*big.Int, amp *big.Int) (*big.Int, error) {
	var xp, err = _xpMem(balances, t.Rates)
	if err != nil {
		return nil, err
	}
	return t.getD(xp, amp)
}

func (t *Pool) _A() *big.Int {
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

func (t *Pool) A() *big.Int {
	var a = t._A()
	return new(big.Int).Div(a, t.APrecision)
}

func (t *Pool) APrecise() *big.Int {
	return t._A()
}

func (t *Pool) getD(xp []*big.Int, a *big.Int) (*big.Int, error) {
	var numTokens = len(xp)
	var s = new(big.Int).SetInt64(0)
	for i := 0; i < numTokens; i++ {
		s = new(big.Int).Add(s, xp[i])
	}
	if s.Cmp(big.NewInt(0)) == 0 {
		return s, nil
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var prevD *big.Int
	var d = new(big.Int).Set(s)
	var nA = new(big.Int).Mul(a, numTokensBI)
	for i := 0; i < MaxLoopLimit; i++ {
		var dP = new(big.Int).Set(d)
		for j := 0; j < numTokens; j++ {
			dP = new(big.Int).Div(
				new(big.Int).Mul(dP, d),
				new(big.Int).Add(new(big.Int).Mul(xp[j], numTokensBI), constant.One), // +1 is to prevent /0 (https://github.com/curvefi/curve-contract/blob/d4e8589/contracts/pools/aave/StableSwapAave.vy#L299)
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
	return nil, errors.ErrDDoesNotConverge
}

func (t *Pool) getY(
	tokenIndexFrom int,
	tokenIndexTo int,
	x *big.Int,
	xp []*big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, errors.ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, errors.ErrTokenIndexesOutOfRange
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var a = t._A()
	if a == nil {
		return nil, ErrInvalidAValue
	}

	var d, err = t.getD(xp, a)
	if err != nil {
		return nil, err
	}
	var c = new(big.Int).Set(d)
	var s = big.NewInt(0)
	var nA = new(big.Int).Mul(a, numTokensBI)
	var _x *big.Int
	for i := 0; i < numTokens; i++ {
		if i == tokenIndexFrom {
			_x = new(big.Int).Set(x)
		} else if i != tokenIndexTo {
			_x = new(big.Int).Set(xp[i])
		} else {
			continue
		}
		if _x.Cmp(constant.Zero) == 0 {
			return nil, errors.ErrZero
		}
		s = new(big.Int).Add(s, _x)
		c = new(big.Int).Div(
			new(big.Int).Mul(c, d),
			new(big.Int).Mul(_x, numTokensBI),
		)
	}
	if nA.Cmp(constant.Zero) == 0 {
		return nil, errors.ErrZero
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).Mul(c, d), t.APrecision),
		new(big.Int).Mul(nA, numTokensBI),
	)
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(new(big.Int).Mul(d, t.APrecision), nA),
	)
	var yPrev *big.Int
	var y = new(big.Int).Set(d)
	for i := 0; i < MaxLoopLimit; i++ {
		yPrev = new(big.Int).Set(y)
		y = new(big.Int).Div(
			new(big.Int).Add(new(big.Int).Mul(y, y), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(y, big.NewInt(2)), b), d),
		)
		if new(big.Int).Sub(y, yPrev).CmpAbs(constant.One) <= 0 {
			return y, nil
		}
	}
	return nil, errors.ErrAmountOutNotConverge
}

func (t *Pool) GetDy(
	i int,
	j int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	var xp = t._xp()
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x = new(big.Int).Add(xp[i], new(big.Int).Div(new(big.Int).Mul(dx, t.Rates[i]), Precision))

	// y: uint256 = self.get_y(i, j, x, xp)
	var y, err = t.getY(i, j, x, xp)
	if err != nil {
		return nil, nil, err
	}

	// dy: uint256 = xp[j] - y - 1
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[j], y), constant.One)

	// fee: uint256 = self.fee * dy / FEE_DENOMINATOR
	var fee = new(big.Int).Div(new(big.Int).Mul(t.Info.SwapFee, dy), FeeDenominator)

	// (dy - fee) * PRECISION / rates[j]
	dy = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(dy, fee), Precision), t.Rates[j])

	// fee * PRECISION / rates[j]
	fee = new(big.Int).Div(new(big.Int).Mul(fee, Precision), t.Rates[j])
	return dy, fee, nil
}

func (t *Pool) getYD(
	a *big.Int,
	tokenIndex int,
	xp []*big.Int,
	d *big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, errors.ErrTokenNotFound
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var c = new(big.Int).Set(d)
	var s = big.NewInt(0)
	var nA = new(big.Int).Mul(a, numTokensBI)
	for i := 0; i < numTokens; i++ {
		if i != tokenIndex {
			s = new(big.Int).Add(s, xp[i])
			c = new(big.Int).Div(
				new(big.Int).Mul(c, d),
				new(big.Int).Mul(xp[i], numTokensBI),
			)
		}
	}
	if nA.Cmp(constant.Zero) == 0 {
		return nil, errors.ErrZero
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).Mul(c, d), t.APrecision),
		new(big.Int).Mul(nA, numTokensBI),
	)
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(new(big.Int).Mul(d, t.APrecision), nA),
	)
	var yPrev *big.Int
	var y = new(big.Int).Set(d)
	for i := 0; i < MaxLoopLimit; i++ {
		yPrev = new(big.Int).Set(y)
		y = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(y, y),
				c,
			),
			new(big.Int).Sub(
				new(big.Int).Add(
					new(big.Int).Mul(y, constant.Two),
					b,
				),
				d,
			),
		)
		if new(big.Int).Sub(y, yPrev).CmpAbs(constant.One) <= 0 {
			return y, nil
		}
	}
	return nil, errors.ErrAmountOutNotConverge
}

/**
 * @notice internal helper function to calculate fee per token multiplier used in
 * swap fee calculations
 */
//func _feePerToken(
//	swapFee *big.Int,
//	numTokens int,
//) *big.Int {
//	var numTokensBI = big.NewInt(int64(numTokens))
//	return new(big.Int).Div(
//		new(big.Int).Mul(
//			swapFee,
//			numTokensBI,
//		),
//		new(big.Int).Mul(
//			new(big.Int).Sub(numTokensBI, constant.One),
//			constant.Four,
//		),
//	)
//}

func (t *Pool) CalculateWithdrawOneCoin(
	tokenAmount *big.Int,
	i int,
) (*big.Int, *big.Int, error) {
	var amp = t._A()
	var xp = t._xp()
	D0, err := t.getD(xp, amp)
	if err != nil {
		return nil, nil, err
	}
	var totalSupply = t.LpSupply
	var D1 = new(big.Int).Sub(D0, new(big.Int).Div(new(big.Int).Mul(tokenAmount, D0), totalSupply))
	newY, err := t.getYD(amp, i, xp, D1)
	if err != nil {
		return nil, nil, err
	}
	var nCoins = len(t.Info.Reserves)
	var xpReduced = make([]*big.Int, nCoins)
	var nCoinBI = big.NewInt(int64(nCoins))
	var fee = new(big.Int).Div(new(big.Int).Mul(t.Info.SwapFee, nCoinBI), new(big.Int).Mul(constant.Four, new(big.Int).Sub(nCoinBI, constant.One)))
	for j := 0; j < nCoins; j += 1 {
		var dxExpected = constant.Zero
		if j == i {
			dxExpected = new(big.Int).Sub(new(big.Int).Div(new(big.Int).Mul(xp[j], D1), D0), newY)
		} else {
			dxExpected = new(big.Int).Sub(xp[j], new(big.Int).Div(new(big.Int).Mul(xp[j], D1), D0))
		}
		xpReduced[j] = new(big.Int).Sub(xp[j], new(big.Int).Div(new(big.Int).Mul(fee, dxExpected), FeeDenominator))
	}
	newYD, err := t.getYD(amp, i, xpReduced, D1)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(xpReduced[i], newYD)
	dy = new(big.Int).Div(new(big.Int).Sub(dy, constant.One), t.Multipliers[i])
	var dy0 = new(big.Int).Div(new(big.Int).Sub(xp[i], newY), t.Multipliers[i])
	return dy, new(big.Int).Sub(dy0, dy), nil
}

func (t *Pool) CalculateTokenAmount(
	amounts []*big.Int,
	deposit bool,
) (*big.Int, error) {
	var numTokens = len(t.Info.Tokens)
	var a = t._A()
	d0, err := t.get_D_mem(t.Info.Reserves, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		if deposit {
			balances1[i] = new(big.Int).Add(t.Info.Reserves[i], amounts[i])
		} else {
			if t.Info.Reserves[i].Cmp(amounts[i]) < 0 {
				return nil, errors.ErrWithdrawMoreThanAvailable
			}
			balances1[i] = new(big.Int).Sub(t.Info.Reserves[i], amounts[i])
		}
	}
	d1, err := t.get_D_mem(balances1, a)
	if err != nil {
		return nil, err
	}
	var totalSupply = t.LpSupply
	var diff = constant.Zero
	if deposit {
		diff = new(big.Int).Sub(d1, d0)
	} else {
		diff = new(big.Int).Sub(d0, d1)
	}
	return new(big.Int).Div(new(big.Int).Mul(diff, totalSupply), d0), nil
}

func (t *Pool) CalculateAddLiquidityOneToken(
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	var numTokens = len(t.Info.Reserves)
	var amounts = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		amounts[i] = big.NewInt(0)
	}
	amounts[tokenIndex] = new(big.Int).Set(tokenAmount)
	amount, err := t.CalculateTokenAmount(
		amounts,
		true)
	return amount, constant.Zero, err
}

func (t *Pool) AddLiquidity(amounts []*big.Int) (*big.Int, error) {
	var nCoins = len(amounts)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var amp = t._A()
	var old_balances = make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		old_balances[i] = t.Info.Reserves[i]
	}
	D0, err := t.get_D_mem(old_balances, amp)
	if err != nil {
		return nil, err
	}
	var token_supply = t.LpSupply
	var new_balances = make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		new_balances[i] = new(big.Int).Add(old_balances[i], amounts[i])
	}
	D1, err := t.get_D_mem(new_balances, amp)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(D0) <= 0 {
		return nil, errors.ErrD1LowerThanD0
	}
	var D2 = D1
	var mint_amount = constant.Zero
	if token_supply.Cmp(constant.Zero) > 0 {
		var _fee = new(big.Int).Div(new(big.Int).Mul(t.Info.SwapFee, nCoinsBi),
			new(big.Int).Mul(constant.Four, big.NewInt(int64(nCoins-1))))
		var _admin_fee = t.AdminFee
		for i := 0; i < nCoins; i += 1 {
			var ideal_balance = new(big.Int).Div(new(big.Int).Mul(D1, old_balances[i]), D0)
			var difference = constant.Zero
			if ideal_balance.Cmp(new_balances[i]) > 0 {
				difference = new(big.Int).Sub(ideal_balance, new_balances[i])
			} else {
				difference = new(big.Int).Sub(new_balances[i], ideal_balance)
			}
			var fee = new(big.Int).Div(new(big.Int).Mul(_fee, difference), FeeDenominator)
			t.Info.Reserves[i] = new(big.Int).Sub(new_balances[i], new(big.Int).Div(new(big.Int).Mul(fee, _admin_fee), FeeDenominator))
			new_balances[i] = new(big.Int).Sub(new_balances[i], fee)
		}
		D2, _ = t.get_D_mem(new_balances, amp)
		mint_amount = new(big.Int).Div(new(big.Int).Mul(token_supply, new(big.Int).Sub(D2, D0)), D0)
	} else {
		for i := 0; i < nCoins; i += 1 {
			t.Info.Reserves[i] = new_balances[i]
		}
		mint_amount = D1
	}
	t.LpSupply = new(big.Int).Add(t.LpSupply, mint_amount)
	return mint_amount, nil
}

func (t *Pool) RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error) {
	var dy, dy_fee, err = t.CalculateWithdrawOneCoin(tokenAmount, i)
	if err != nil {
		return nil, err
	}
	t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], new(big.Int).Add(dy, new(big.Int).Div(new(big.Int).Mul(dy_fee, t.AdminFee), FeeDenominator)))
	t.LpSupply = new(big.Int).Sub(t.LpSupply, tokenAmount)
	return dy, nil
}

func (t *Pool) GetVirtualPrice() (*big.Int, error) {
	var xp = t._xp()
	var A = t._A()
	var D, err = t.getD(xp, A)
	if err != nil {
		return nil, err
	}
	if t.LpSupply.Cmp(constant.Zero) == 0 {
		return nil, errors.ErrDenominatorZero
	}
	return new(big.Int).Div(new(big.Int).Mul(D, Precision), t.LpSupply), nil
}

package base

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

func _xpMem(
	balances []uint256.Int,
	rates []uint256.Int,
) ([]uint256.Int, error) {
	var numTokens = len(balances)
	if numTokens != len(rates) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	xp := make([]uint256.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		xp[i].Div(number.Mul(&rates[i], &balances[i]), Precision)
	}
	return xp, nil
}

func (t *PoolBaseSimulator) _xp() []uint256.Int {
	var result [MaxTokenCount]uint256.Int
	count := t._xp_inplace(result[:])
	return result[:count]
}

func (t *PoolBaseSimulator) _xp_inplace(result []uint256.Int) int {
	var nTokens = len(t.Info.Tokens)
	for i := 0; i < nTokens; i += 1 {
		result[i].Div(number.Mul(&t.Rates[i], &t.Reserves[i]), Precision)
	}
	return nTokens
}

func (t *PoolBaseSimulator) get_D_mem(balances []uint256.Int, amp *uint256.Int) (*uint256.Int, error) {
	var xp, err = _xpMem(balances, t.Rates)
	if err != nil {
		return nil, err
	}
	return t.getD(xp, amp)
}

func (t *PoolBaseSimulator) _A() *uint256.Int {
	var t1 = t.FutureATime
	var a1 = &t.FutureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = t.InitialATime
		var a0 = &t.InitialA
		if a1.Cmp(a0) > 0 {
			return number.Add(
				a0,
				number.Div(
					number.Mul(
						number.Sub(a1, a0),
						number.SetUint64(uint64(now-t0)),
					),
					number.SetUint64(uint64(t1-t0)),
				),
			)
		} else {
			return number.Sub(
				a0,
				number.Div(
					number.Mul(
						number.Sub(a0, a1),
						number.SetUint64(uint64(now-t0)),
					),
					number.SetUint64(uint64(t1-t0)),
				),
			)
		}
	}
	return a1
}

func (t *PoolBaseSimulator) A() *uint256.Int {
	var a = t._A()
	return number.Div(a, &t.APrecision)
}

func (t *PoolBaseSimulator) APrecise() *uint256.Int {
	return t._A()
}

func (t *PoolBaseSimulator) getD(xp []uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	var result uint256.Int
	err := t.getD_inplace(xp, a, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (t *PoolBaseSimulator) getD_inplace(xp []uint256.Int, a *uint256.Int, d *uint256.Int) error {
	var numTokens = len(xp)
	var s uint256.Int
	s.Clear()
	for i := 0; i < numTokens; i++ {
		s.Add(&s, &xp[i])
	}
	if s.IsZero() {
		d.Set(&s)
		return nil
	}

	// this is in a loop so should use local variable instead of allocating
	// writing like this will hurt readability so double check with the original SC code here if needed
	// https://github.com/curvefi/curve-contract/blob/d4e8589ac92c4019b3064b2f3a8a87dbc3281b46/contracts/pool-templates/base/SwapTemplateBase.vy#L217

	/*
		D: uint256 = S
		Ann: uint256 = amp * N_COINS
		for _i in range(255):
		  D_P: uint256 = D
		  for _x in xp:
		    D_P = D_P * D / (_x * N_COINS)  # If division by 0, this will be borked: only withdrawal will work. And that is good
		  Dprev = D
		  D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)
		  # Equality with the precision of 1
		  if D > Dprev:
		    if D - Dprev <= 1:
		      return D
		  else:
		    if Dprev - D <= 1:
		      return D
	*/
	var dP, numTokensPlus1, nA, nA_mul_s_div_APrec, nA_sub_APrec, prevD, tmp0, tmp1, tmp2, tmp3, tmp4, tmp5, tmp6, tmp7 uint256.Int
	numTokensPlus1.SetUint64(uint64(numTokens + 1))
	d.Set(&s)
	nA.Mul(a, &t.numTokensBI)
	nA_mul_s_div_APrec.Mul(&nA, &s)
	nA_mul_s_div_APrec.Div(&nA_mul_s_div_APrec, &t.APrecision)
	nA_sub_APrec.Sub(&nA, &t.APrecision)

	for i := 0; i < MaxLoopLimit; i++ {
		// D_P: uint256 = D
		dP.Set(d)

		for j := 0; j < numTokens; j++ {
			// D_P = D_P * D / (_x * N_COINS +1)
			// +1 is to prevent /0 (https://github.com/curvefi/curve-contract/blob/d4e8589/contracts/pools/aave/StableSwapAave.vy#L299)

			// nominator
			tmp0.Mul(&dP, d)

			// denominator
			tmp1.Mul(&xp[j], &t.numTokensBI)
			tmp1.AddUint64(&tmp1, 1)

			// update dP
			dP.Div(&tmp0, &tmp1)
		}
		// Dprev = D
		prevD.Set(d)

		// D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)

		// nominator
		tmp6.Add(&nA_mul_s_div_APrec, tmp3.Mul(&dP, &t.numTokensBI)) // (Ann * S / A_PRECISION + D_P * N_COINS)
		tmp2.Mul(&tmp6, d)                                           // (Ann * S / A_PRECISION + D_P * N_COINS) * D

		// denominator
		tmp7.Mul(&nA_sub_APrec, d)     // (Ann - A_PRECISION) * D
		tmp4.Div(&tmp7, &t.APrecision) // (Ann - A_PRECISION) * D / A_PRECISION
		tmp5.Mul(&dP, &numTokensPlus1) // (N_COINS + 1) * D_P
		tmp4.Add(&tmp4, &tmp5)         // (Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P

		// update d
		d.Div(&tmp2, &tmp4)

		// calc abs(D - Dprev) and compare against 1
		if withinDelta(d, &prevD, 1) {
			return nil
		}
	}
	return ErrDDoesNotConverge
}

func (t *PoolBaseSimulator) getY(
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []uint256.Int,
	dCached *uint256.Int,
	y *uint256.Int,
) error {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return ErrTokenIndexesOutOfRange
	}

	var a = t._A()
	if a == nil {
		return ErrInvalidAValue
	}

	var d uint256.Int
	if dCached != nil {
		d.Set(dCached)
	} else {
		err := t.getD_inplace(xp, a, &d)
		if err != nil {
			return err
		}
	}
	var c = number.Set(&d)
	var nA = number.Mul(a, &t.numTokensBI)
	var _x, s uint256.Int
	s.Clear()
	for i := 0; i < numTokens; i++ {
		if i == tokenIndexFrom {
			_x.Set(x)
		} else if i != tokenIndexTo {
			_x.Set(&xp[i])
		} else {
			continue
		}
		if _x.IsZero() {
			return ErrZero
		}
		s.Add(&s, &_x)
		c.Div(
			number.Mul(c, &d),
			number.Mul(&_x, &t.numTokensBI),
		)
	}
	if nA.IsZero() {
		return ErrZero
	}
	c.Div(
		number.Mul(number.Mul(c, &d), &t.APrecision),
		number.Mul(nA, &t.numTokensBI),
	)
	var b = number.Add(
		&s,
		number.Div(number.Mul(&d, &t.APrecision), nA),
	)

	// this is in a loop so should use local variable instead of allocating
	// writing like this will hurt readability so double check with the original SC code here if needed
	// https://github.com/curvefi/curve-contract/blob/d4e8589ac92c4019b3064b2f3a8a87dbc3281b46/contracts/pool-templates/base/SwapTemplateBase.vy#L408
	/*
		for _i in range(255):
			y_prev = y
			y = (y*y + c) / (2 * y + b - D)
			# Equality with the precision of 1
			if y > y_prev:
		  	if y - y_prev <= 1:
		    	return y
			else:
		  	if y_prev - y <= 1:
		    	return y
	*/
	var tmp, tmp1 uint256.Int
	var yPrev uint256.Int
	y.Set(&d)
	for i := 0; i < MaxLoopLimit; i++ {
		// y_prev = y
		yPrev.Set(y)

		// y = (y*y + c) / (2 * y + b - D)
		// first calc denominator into tmp
		tmp.Add(y, y) // 2 * y
		tmp.Add(&tmp, b)
		tmp.Sub(&tmp, &d)
		// then calc nominator into tmp1
		tmp1.Mul(y, y)
		tmp1.Add(&tmp1, c)
		// then the whole y
		y.Div(&tmp1, &tmp)

		// calc abs(y - y_prev) and compare against 1
		if withinDelta(y, &yPrev, 1) {
			return nil
		}
	}
	return ErrAmountOutNotConverge
}

func withinDelta(x, y *uint256.Int, delta uint64) bool {
	var diff uint256.Int
	if x.Cmp(y) > 0 {
		diff.Sub(x, y)
	} else {
		diff.Sub(y, x)
	}
	if diff.CmpUint64(delta) <= 0 {
		return true
	}
	return false
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) GetDy(
	i int,
	j int,
	dx *big.Int,
	dCached *big.Int,
) (*big.Int, *big.Int, error) {
	var dy, fee uint256.Int
	err := t.GetDyU256(i, j, number.SetFromBig(dx), number.SetFromBig(dCached), &dy, &fee)
	if err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), fee.ToBig(), err
}

func (t *PoolBaseSimulator) GetDyU256(
	i int,
	j int,
	dx *uint256.Int,
	dCached *uint256.Int,
	dy *uint256.Int,
	fee *uint256.Int,
) error {
	var xp = t._xp()
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x = number.Add(&xp[i], number.Div(number.Mul(dx, &t.Rates[i]), Precision))

	// y: uint256 = self.get_y(i, j, x, xp)
	var y uint256.Int
	var err = t.getY(i, j, x, xp, dCached, &y)
	if err != nil {
		return err
	}

	// dy: uint256 = xp[j] - y - 1
	dy.SubUint64(number.Sub(&xp[j], &y), 1)

	// fee: uint256 = self.fee * dy / FEE_DENOMINATOR
	fee.Div(number.Mul(&t.SwapFee, dy), FeeDenominator)

	// (dy - fee) * PRECISION / rates[j]
	dy.Div(number.Mul(dy.Sub(dy, fee), Precision), &t.Rates[j])

	// fee * PRECISION / rates[j]
	fee.Div(number.Mul(fee, Precision), &t.Rates[j])

	return nil
}

func (t *PoolBaseSimulator) getYD(
	a *uint256.Int,
	tokenIndex int,
	xp []uint256.Int,
	d *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, ErrTokenNotFound
	}
	var c, s uint256.Int
	c.Set(d)
	s.Clear()
	var nA = number.Mul(a, &t.numTokensBI)
	for i := 0; i < numTokens; i++ {
		if i != tokenIndex {
			s.Add(&s, &xp[i])
			c.Div(
				number.Mul(&c, d),
				number.Mul(&xp[i], &t.numTokensBI),
			)
		}
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Div(
		number.Mul(number.Mul(&c, d), &t.APrecision),
		number.Mul(nA, &t.numTokensBI),
	)
	var b = number.Add(
		&s,
		number.Div(number.Mul(d, &t.APrecision), nA),
	)
	var y, yPrev uint256.Int
	y.Set(d)
	for i := 0; i < MaxLoopLimit; i++ {
		yPrev.Set(&y)
		y.Div(
			number.Add(
				number.Mul(&y, &y),
				&c,
			),
			number.Sub(
				number.Add(
					number.Add(&y, &y),
					b,
				),
				d,
			),
		)
		if withinDelta(&y, &yPrev, 1) {
			return number.Set(&y), nil
		}
	}
	return nil, ErrAmountOutNotConverge
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
//			new(big.Int).Sub(numTokensBI, bignumber.One),
//			bignumber.Four,
//		),
//	)
//}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) CalculateWithdrawOneCoin(
	tokenAmount *big.Int,
	i int,
) (*big.Int, *big.Int, error) {
	dy, diff, err := t.CalculateWithdrawOneCoinU256(number.SetFromBig(tokenAmount), i)
	if err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), diff.ToBig(), nil
}

func (t *PoolBaseSimulator) CalculateWithdrawOneCoinU256(
	tokenAmount *uint256.Int,
	i int,
) (*uint256.Int, *uint256.Int, error) {
	var amp = t._A()
	var xp = t._xp()
	D0, err := t.getD(xp, amp)
	if err != nil {
		return nil, nil, err
	}
	var totalSupply = &t.LpSupply
	var D1 = number.Sub(D0, number.Div(number.Mul(tokenAmount, D0), totalSupply))
	newY, err := t.getYD(amp, i, xp, D1)
	if err != nil {
		return nil, nil, err
	}
	var nCoins = len(t.Info.Reserves)
	var xpReduced = make([]uint256.Int, nCoins)
	var nCoinBI = number.SetUint64(uint64(nCoins))
	var fee = number.Div(number.Mul(&t.SwapFee, nCoinBI), number.Mul(uint256.NewInt(4), number.SubUint64(nCoinBI, 1)))
	for j := 0; j < nCoins; j += 1 {
		var dxExpected uint256.Int
		if j == i {
			dxExpected.Sub(number.Div(number.Mul(&xp[j], D1), D0), newY)
		} else {
			dxExpected.Sub(&xp[j], number.Div(number.Mul(&xp[j], D1), D0))
		}
		xpReduced[j].Sub(&xp[j], number.Div(number.Mul(fee, &dxExpected), FeeDenominator))
	}
	newYD, err := t.getYD(amp, i, xpReduced, D1)
	if err != nil {
		return nil, nil, err
	}
	var dy = number.Sub(&xpReduced[i], newYD)
	dy.Div(number.SubUint64(dy, 1), &t.Multipliers[i])
	var dy0 = number.Div(number.Sub(&xp[i], newY), &t.Multipliers[i])
	return dy, number.Sub(dy0, dy), nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) CalculateTokenAmount(
	amounts []*big.Int,
	deposit bool,
) (*big.Int, error) {
	amountsU256 := make([]uint256.Int, len(amounts))
	for i, amount := range amounts {
		amountsU256[i].SetFromBig(amount)
	}
	res, err := t.CalculateTokenAmountU256(amountsU256, deposit)
	if err != nil {
		return nil, err
	}
	return res.ToBig(), nil
}

func (t *PoolBaseSimulator) CalculateTokenAmountU256(
	amounts []uint256.Int,
	deposit bool,
) (*uint256.Int, error) {
	var numTokens = len(t.Info.Tokens)
	var a = t._A()
	d0, err := t.get_D_mem(t.Reserves, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]uint256.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		if deposit {
			balances1[i].Add(&t.Reserves[i], &amounts[i])
		} else {
			if t.Reserves[i].Cmp(&amounts[i]) < 0 {
				return nil, ErrWithdrawMoreThanAvailable
			}
			balances1[i].Sub(&t.Reserves[i], &amounts[i])
		}
	}
	d1, err := t.get_D_mem(balances1, a)
	if err != nil {
		return nil, err
	}
	var totalSupply = t.LpSupply
	var diff uint256.Int
	if deposit {
		diff.Sub(d1, d0)
	} else {
		diff.Sub(d0, d1)
	}
	return number.Div(number.Mul(&diff, &totalSupply), d0), nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) AddLiquidity(amounts []*big.Int) (*big.Int, error) {
	amountsU256 := make([]uint256.Int, len(amounts))
	for i, amount := range amounts {
		amountsU256[i].SetFromBig(amount)
	}
	res, err := t.AddLiquidityU256(amountsU256)
	if err != nil {
		return nil, err
	}
	return res.ToBig(), err
}

func (t *PoolBaseSimulator) AddLiquidityU256(amounts []uint256.Int) (*uint256.Int, error) {
	var nCoins = len(amounts)
	var nCoinsBi = uint256.NewInt(uint64(nCoins))
	var amp = t._A()
	var old_balances = make([]uint256.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		old_balances[i].Set(&t.Reserves[i])
	}
	D0, err := t.get_D_mem(old_balances, amp)
	if err != nil {
		return nil, err
	}
	var token_supply = t.LpSupply
	var new_balances = make([]uint256.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		new_balances[i].Add(&old_balances[i], &amounts[i])
	}
	D1, err := t.get_D_mem(new_balances, amp)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(D0) <= 0 {
		return nil, ErrD1LowerThanD0
	}
	var mint_amount uint256.Int
	if !token_supply.IsZero() {
		var _fee = number.Div(number.Mul(&t.SwapFee, nCoinsBi),
			number.Mul(uint256.NewInt(4), uint256.NewInt(uint64(nCoins-1))))
		var _admin_fee = t.AdminFee
		for i := 0; i < nCoins; i += 1 {
			var ideal_balance = number.Div(number.Mul(D1, &old_balances[i]), D0)
			var difference uint256.Int
			if ideal_balance.Cmp(&new_balances[i]) > 0 {
				difference.Sub(ideal_balance, &new_balances[i])
			} else {
				difference.Sub(&new_balances[i], ideal_balance)
			}
			var fee = number.Div(number.Mul(_fee, &difference), FeeDenominator)
			t.Reserves[i].Sub(&new_balances[i], number.Div(number.Mul(fee, &_admin_fee), FeeDenominator))
			bignumber.FillBig(&t.Reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
			new_balances[i].Sub(&new_balances[i], fee)
		}
		D2, _ := t.get_D_mem(new_balances, amp)
		mint_amount.Div(number.Mul(&token_supply, number.Sub(D2, D0)), D0)
	} else {
		for i := 0; i < nCoins; i += 1 {
			t.Reserves[i].Set(&new_balances[i])
			bignumber.FillBig(&t.Reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
		}
		mint_amount.Set(D1)
	}
	t.LpSupply.Add(&t.LpSupply, &mint_amount)
	return &mint_amount, nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error) {
	dy, err := t.RemoveLiquidityOneCoinU256(number.SetFromBig(tokenAmount), i)
	if err != nil {
		return nil, err
	}
	return dy.ToBig(), nil
}

func (t *PoolBaseSimulator) RemoveLiquidityOneCoinU256(tokenAmount *uint256.Int, i int) (*uint256.Int, error) {
	var dy, dy_fee, err = t.CalculateWithdrawOneCoinU256(tokenAmount, i)
	if err != nil {
		return nil, err
	}
	t.Reserves[i].Sub(&t.Reserves[i], number.Add(dy, number.Div(number.Mul(dy_fee, &t.AdminFee), FeeDenominator)))
	bignumber.FillBig(&t.Reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	t.LpSupply.Sub(&t.LpSupply, tokenAmount)
	return dy, nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolBaseSimulator) GetVirtualPrice() (*big.Int, *big.Int, error) {
	vPrice, d, err := t.GetVirtualPriceU256()
	if err != nil {
		return nil, nil, err
	}
	return vPrice.ToBig(), d.ToBig(), err
}

func (t *PoolBaseSimulator) GetVirtualPriceU256() (*uint256.Int, *uint256.Int, error) {
	var xp = t._xp()
	var A = t._A()
	var D, err = t.getD(xp, A)
	if err != nil {
		return nil, nil, err
	}
	if t.LpSupply.IsZero() {
		return nil, nil, ErrDenominatorZero
	}
	return number.Div(number.Mul(D, Precision), &t.LpSupply), D, nil
}

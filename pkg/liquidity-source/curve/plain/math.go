package plain

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
)

// most of the code here will follow https://github.com/curvefi/curve-factory/blob/master/contracts/implementations/plain-3/Plain3Basic.vy
// with some modifications to work with other variants (see pool_simulator.go for completed list)
// also, some functions are modified to pass in the result pointer instead of allocating and returning result

func (t *PoolSimulator) _A() *uint256.Int {
	var t1 = t.extra.FutureATime
	var a1 = t.extra.FutureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = t.extra.InitialATime
		var a0 = t.extra.InitialA
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

func xpMem(
	rates []uint256.Int,
	balances []uint256.Int,
) []uint256.Int {
	// try to put `result` in caller's stack (this func will be inlined)
	var result [shared.MaxTokenCount]uint256.Int
	count := xpMem_inplace(rates, balances, result[:])
	return result[:count]
}

func xpMem_inplace(
	rates []uint256.Int,
	balances []uint256.Int,
	xp []uint256.Int,
) int {
	numTokens := len(rates)
	for i := 0; i < numTokens; i++ {
		xp[i].Div(number.Mul(&rates[i], &balances[i]), Precision)
	}
	return numTokens
}

// D invariant calculation in non-overflowing integer operations iteratively
// - `D`: output
func (t *PoolSimulator) getD(xp []uint256.Int, a *uint256.Int, D *uint256.Int) error {
	var S uint256.Int
	S.Clear()
	for i := range xp {
		if xp[i].IsZero() {
			// this will cause div by zero down below
			return ErrZero
		}
		S.Add(&S, &xp[i])
	}
	if S.IsZero() {
		D.Clear()
		return nil
	}

	var D_P, Ann, Ann_mul_S_div_APrec, Ann_sub_APrec, Dprev uint256.Int

	// D: uint256 = S
	D.Set(&S)

	// Ann: uint256 = amp * N_COINS
	Ann.Mul(a, &t.numTokensU256)

	// pre-calculate some values to be used in the loop
	// Ann * S / A_PRECISION
	Ann_mul_S_div_APrec.Div(number.Mul(&Ann, &S), t.staticExtra.APrecision)
	// Ann - A_PRECISION
	Ann_sub_APrec.Sub(&Ann, t.staticExtra.APrecision)

	numTokensPlus1 := uint256.NewInt(uint64(t.numTokens + 1))

	for i := 0; i < MaxLoopLimit; i++ {
		// D_P: uint256 = D
		D_P.Set(D)

		for j := range xp {
			// D_P = D_P * D / (_x * N_COINS)
			// some pools (very few) will divide by `(_x * N_COINS +1)` instead to avoid div by zero (https://github.com/curvefi/curve-contract/blob/d4e8589/contracts/pools/aave/StableSwapAave.vy#L299)
			// but we can't apply that to other pools because it will lead to incorrect result (return high amount while the pool cannot be used anymore)
			// so here we'll use the original formula and do the zero check at the beginning
			D_P.Div(
				number.SafeMul(&D_P, D),
				number.SafeMul(&xp[j], &t.numTokensU256),
			)
		}
		// Dprev = D
		Dprev.Set(D)

		// D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)
		D.Div(
			number.SafeMul(
				number.SafeAdd(&Ann_mul_S_div_APrec, number.SafeMul(&D_P, &t.numTokensU256)),
				D,
			),
			number.SafeAdd(
				number.Div(number.SafeMul(&Ann_sub_APrec, D), t.staticExtra.APrecision),
				number.SafeMul(&D_P, numTokensPlus1),
			),
		)

		// calc abs(D - Dprev) and compare against 1
		if withinDelta(D, &Dprev, 1) {
			return nil
		}
	}
	return ErrDDoesNotConverge
}

func (t *PoolSimulator) get_D_mem(rates []uint256.Int, balances []uint256.Int, amp *uint256.Int, D *uint256.Int) error {
	var xp = xpMem(rates, balances)
	return t.getD(xp, amp, D)
}

// Calculate x[j] if one makes x[i] = x
// - `dCached`: if `D` has been calculated before the reuse it here (use nil if not available)
// - `y`: output x[j]
func (t *PoolSimulator) getY(
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []uint256.Int,
	dCached *uint256.Int,
	y *uint256.Int,
) error {
	if tokenIndexFrom == tokenIndexTo {
		return ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= t.numTokens && tokenIndexTo >= t.numTokens {
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
		err := t.getD(xp, a, &d)
		if err != nil {
			return err
		}
	}
	var c = number.Set(&d)
	var nA = number.SafeMul(a, &t.numTokensU256)
	var _x, s uint256.Int
	s.Clear()
	for i := 0; i < t.numTokens; i++ {
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
			number.SafeMul(c, &d),
			number.SafeMul(&_x, &t.numTokensU256),
		)
	}
	if nA.IsZero() {
		return ErrZero
	}
	c.Div(
		number.SafeMul(number.SafeMul(c, &d), t.staticExtra.APrecision),
		number.SafeMul(nA, &t.numTokensU256),
	)
	var b = number.SafeAdd(
		&s,
		number.Div(number.SafeMul(&d, t.staticExtra.APrecision), nA),
	)

	var yPrev uint256.Int
	y.Set(&d)
	for i := 0; i < MaxLoopLimit; i++ {
		// y_prev = y
		yPrev.Set(y)

		// y = (y*y + c) / (2 * y + b - D)
		y.Div(
			number.SafeAdd(number.SafeMul(y, y), c),
			number.SafeSub(
				number.SafeAdd(
					number.SafeAdd(y, y), // 2 * y
					b),
				&d),
		)

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
func (t *PoolSimulator) GetDy(
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

// Calculate the current output dy given input dx
func (t *PoolSimulator) GetDyU256(
	i int,
	j int,
	dx *uint256.Int,
	dCached *uint256.Int,
	dy *uint256.Int,
	fee *uint256.Int,
) error {
	var xp = xpMem(t.extra.RateMultipliers, t.reserves)
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x = number.Add(&xp[i], number.Div(number.Mul(dx, &t.extra.RateMultipliers[i]), Precision))

	// y: uint256 = self.get_y(i, j, x, xp)
	var y uint256.Int
	var err = t.getY(i, j, x, xp, dCached, &y)
	if err != nil {
		return err
	}

	// in SC, `xp[j] - y - 1` will check for underflow and raise exception
	// here we're using uint256.Int so have to check manually
	yPlus1 := number.AddUint64(&y, 1)
	if xp[j].Cmp(yPlus1) < 0 {
		return ErrReserveTooSmall
	}

	// dy: uint256 = xp[j] - y - 1
	dy.SubUint64(number.Sub(&xp[j], &y), 1)

	// fee: uint256 = self.fee * dy / FEE_DENOMINATOR
	fee.Div(number.Mul(t.extra.SwapFee, dy), FeeDenominator)

	// (dy - fee) * PRECISION / rates[j]
	if dy.Cmp(fee) < 0 {
		return ErrReserveTooSmall
	}
	dy.Div(number.Mul(dy.Sub(dy, fee), Precision), &t.extra.RateMultipliers[j])

	// fee * PRECISION / rates[j]
	fee.Div(number.Mul(fee, Precision), &t.extra.RateMultipliers[j])

	return nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) GetDx(
	i int,
	j int,
	dy *big.Int,
	dCached *big.Int,
) (*big.Int, *big.Int, error) {
	var dx, fee uint256.Int
	err := t.GetDxU256(i, j, number.SetFromBig(dy), number.SetFromBig(dCached), &dx, &fee)
	if err != nil {
		return nil, nil, err
	}
	return dx.ToBig(), fee.ToBig(), err
}

// Calculate the current input dx given output dy
/*
def get_dx(i: int128, j: int128, dy: uint256) -> uint256:
   # dx and dy in c-units
   rates: uint256[N_COINS] = self._stored_rates()
   xp: uint256[N_COINS] = self._xp(rates)

   y: uint256 = xp[j] - (dy * FEE_DENOMINATOR / (FEE_DENOMINATOR - self.fee)) * rates[j] / PRECISION
   x: uint256 = self.get_y(j, i, y, xp)
   dx: uint256 = (x - xp[i]) * PRECISION / rates[i]
   return dx
*/
func (t *PoolSimulator) GetDxU256(
	i int,
	j int,
	dy *uint256.Int,
	dCached *uint256.Int,
	dx *uint256.Int,
	fee *uint256.Int,
) error {
	var xp = xpMem(t.extra.RateMultipliers, t.reserves)

	// in SC, `FEE_DENOMINATOR - self.fee` will check for underflow and raise exception
	// here we're using uint256.Int so have to check manually
	if FeeDenominator.Cmp(t.extra.SwapFee) < 0 {
		return ErrInvalidFee
	}

	// yOut = (dy * FEE_DENOMINATOR / (FEE_DENOMINATOR - self.fee)) * rates[j] / PRECISION
	yOut := number.Mul(
		number.Div(number.Mul(dy, FeeDenominator), number.Sub(FeeDenominator, t.extra.SwapFee)),
		number.Div(&t.extra.RateMultipliers[j], Precision),
	)

	// in SC, `xp[j] - yOut` will check for underflow and raise exception
	// here we're using uint256.Int so have to check manually
	if (&xp[j]).Cmp(yOut) < 0 {
		return ErrReserveTooSmall
	}

	// y: uint256 = xp[j] - (dy * FEE_DENOMINATOR / (FEE_DENOMINATOR - self.fee)) * rates[j] / PRECISION = xp[j] - yOut
	var y = number.Sub(&xp[j], yOut)

	// x: uint256 = self.get_y(j, i, y, xp)
	var x uint256.Int
	var err = t.getY(j, i, y, xp, dCached, &x)
	if err != nil {
		return err
	}

	// in SC, `x - xp[i]` will check for underflow and raise exception
	// here we're using uint256.Int so have to check manually
	// dx: uint256 = (x - xp[i]) * PRECISION / rates[i]
	if x.Cmp(&xp[i]) < 0 {
		return ErrNewReserveInvalid
	}

	dx.Div(number.Mul(number.Sub(&x, &xp[i]), Precision), &t.extra.RateMultipliers[i])

	// fee = yOut - dy
	fee.Sub(yOut, dy)

	return nil
}

func (t *PoolSimulator) getYD(
	a *uint256.Int,
	tokenIndex int,
	xp []uint256.Int,
	d *uint256.Int,

	//output
	y *uint256.Int,
) error {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return ErrTokenNotFound
	}
	var c, s uint256.Int
	c.Set(d)
	s.Clear()
	var nA = number.Mul(a, &t.numTokensU256)
	for i := 0; i < numTokens; i++ {
		if i != tokenIndex {
			s.Add(&s, &xp[i])
			c.Div(
				number.Mul(&c, d),
				number.Mul(&xp[i], &t.numTokensU256),
			)
		}
	}
	if nA.IsZero() {
		return ErrZero
	}
	c.Div(
		number.Mul(number.Mul(&c, d), t.staticExtra.APrecision),
		number.Mul(nA, &t.numTokensU256),
	)
	var b = number.Add(
		&s,
		number.Div(number.Mul(d, t.staticExtra.APrecision), nA),
	)
	var yPrev uint256.Int
	y.Set(d)
	for i := 0; i < MaxLoopLimit; i++ {
		yPrev.Set(y)
		y.Div(
			number.Add(
				number.Mul(y, y),
				&c,
			),
			number.Sub(
				number.Add(
					number.Add(y, y),
					b,
				),
				d,
			),
		)
		if withinDelta(y, &yPrev, 1) {
			return nil
		}
	}
	return ErrAmountOutNotConverge
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) CalculateWithdrawOneCoin(
	tokenAmount *big.Int,
	i int,
) (*big.Int, *big.Int, error) {
	var dy, dyFee uint256.Int
	err := t.CalculateWithdrawOneCoinU256(number.SetFromBig(tokenAmount), i, &dy, &dyFee)
	if err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), dyFee.ToBig(), nil
}

func (t *PoolSimulator) CalculateWithdrawOneCoinU256(
	tokenAmount *uint256.Int,
	i int,

	// output
	dy *uint256.Int, dyFee *uint256.Int,
) error {
	var amp = t._A()
	var xp = xpMem(t.extra.RateMultipliers, t.reserves)
	var D0, newY, newYD uint256.Int
	err := t.getD(xp, amp, &D0)
	if err != nil {
		return err
	}
	var totalSupply = &t.LpSupply
	var D1 = number.Sub(&D0, number.Div(number.Mul(tokenAmount, &D0), totalSupply))
	err = t.getYD(amp, i, xp, D1, &newY)
	if err != nil {
		return err
	}
	var nCoins = len(t.Info.Reserves)
	var xpReduced [shared.MaxTokenCount]uint256.Int
	var nCoinBI = number.SetUint64(uint64(nCoins))
	var fee = number.Div(number.Mul(t.extra.SwapFee, nCoinBI), number.Mul(uint256.NewInt(4), number.SubUint64(nCoinBI, 1)))
	for j := 0; j < nCoins; j += 1 {
		var dxExpected uint256.Int
		if j == i {
			dxExpected.Sub(number.Div(number.Mul(&xp[j], D1), &D0), &newY)
		} else {
			dxExpected.Sub(&xp[j], number.Div(number.Mul(&xp[j], D1), &D0))
		}
		xpReduced[j].Sub(&xp[j], number.Div(number.Mul(fee, &dxExpected), FeeDenominator))
	}
	err = t.getYD(amp, i, xpReduced[:nCoins], D1, &newYD)
	if err != nil {
		return err
	}
	dy.Sub(&xpReduced[i], &newYD)
	dy.Div(number.SubUint64(dy, 1), &t.precisionMultipliers[i])
	var dy0 = number.Div(number.Sub(&xp[i], &newY), &t.precisionMultipliers[i])
	dyFee.Sub(dy0, dy)
	return nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) CalculateTokenAmount(
	amounts []*big.Int,
	deposit bool,
) (*big.Int, error) {
	amountsU256 := make([]uint256.Int, len(amounts))
	for i, amount := range amounts {
		amountsU256[i].SetFromBig(amount)
	}
	var mintAmount uint256.Int
	var feeAmounts [shared.MaxTokenCount]uint256.Int
	err := t.CalculateTokenAmountU256(amountsU256, deposit, &mintAmount, feeAmounts[:t.numTokens])
	if err != nil {
		return nil, err
	}
	return mintAmount.ToBig(), nil
}

func (t *PoolSimulator) CalculateTokenAmountU256(
	amounts []uint256.Int,
	deposit bool,

	// output
	mintAmount *uint256.Int,
	feeAmounts []uint256.Int,
) error {
	var numTokens = len(t.Info.Tokens)
	var a = t._A()
	var d0, d1, d2 uint256.Int
	err := t.get_D_mem(t.extra.RateMultipliers, t.reserves, a, &d0)
	if err != nil {
		return err
	}
	var balances1 [shared.MaxTokenCount]uint256.Int
	for i := 0; i < numTokens; i++ {
		if deposit {
			balances1[i].Add(&t.reserves[i], &amounts[i])
		} else {
			if t.reserves[i].Cmp(&amounts[i]) < 0 {
				return ErrWithdrawMoreThanAvailable
			}
			balances1[i].Sub(&t.reserves[i], &amounts[i])
		}
	}
	err = t.get_D_mem(t.extra.RateMultipliers, balances1[:numTokens], a, &d1)
	if err != nil {
		return err
	}

	// in SC, this method won't take fee into account, so the result is different than the actual add_liquidity method
	// we'll copy that code here

	// We need to recalculate the invariant accounting for fees
	// to calculate fair user's share
	var totalSupply = t.LpSupply
	var difference uint256.Int
	if !totalSupply.IsZero() {
		var _fee = number.Div(number.Mul(t.extra.SwapFee, &t.numTokensU256),
			number.Mul(number.Number_4, uint256.NewInt(uint64(t.numTokens-1))))
		var _admin_fee = t.extra.AdminFee
		for i := 0; i < t.numTokens; i += 1 {
			var ideal_balance = number.Div(number.Mul(&d1, &t.reserves[i]), &d0)
			if ideal_balance.Cmp(&balances1[i]) > 0 {
				difference.Sub(ideal_balance, &balances1[i])
			} else {
				difference.Sub(&balances1[i], ideal_balance)
			}
			var fee = number.Div(number.Mul(_fee, &difference), FeeDenominator)
			feeAmounts[i].Set(number.Div(number.Mul(fee, _admin_fee), FeeDenominator))
			balances1[i].Sub(&balances1[i], fee)
		}
		_ = t.get_D_mem(t.extra.RateMultipliers, balances1[:t.numTokens], a, &d2)
		mintAmount.Div(number.Mul(&totalSupply, number.Sub(&d2, &d0)), &d0)
	} else {
		mintAmount.Set(&d1)
	}

	return nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) AddLiquidity(amounts []*big.Int) (*big.Int, error) {
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

func (t *PoolSimulator) AddLiquidityU256(amounts []uint256.Int) (*uint256.Int, error) {
	var nCoins = len(amounts)
	var nCoinsBi = uint256.NewInt(uint64(nCoins))
	var amp = t._A()
	var old_balances = make([]uint256.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		old_balances[i].Set(&t.reserves[i])
	}
	var D0, D1, D2 uint256.Int
	err := t.get_D_mem(t.extra.RateMultipliers, old_balances, amp, &D0)
	if err != nil {
		return nil, err
	}
	var token_supply = t.LpSupply
	var new_balances = make([]uint256.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		new_balances[i].Add(&old_balances[i], &amounts[i])
	}
	err = t.get_D_mem(t.extra.RateMultipliers, new_balances, amp, &D1)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(&D0) <= 0 {
		return nil, ErrD1LowerThanD0
	}
	var mint_amount uint256.Int
	if !token_supply.IsZero() {
		var _fee = number.Div(number.Mul(t.extra.SwapFee, nCoinsBi),
			number.Mul(uint256.NewInt(4), uint256.NewInt(uint64(nCoins-1))))
		var _admin_fee = t.extra.AdminFee
		for i := 0; i < nCoins; i += 1 {
			var ideal_balance = number.Div(number.Mul(&D1, &old_balances[i]), &D0)
			var difference uint256.Int
			if ideal_balance.Cmp(&new_balances[i]) > 0 {
				difference.Sub(ideal_balance, &new_balances[i])
			} else {
				difference.Sub(&new_balances[i], ideal_balance)
			}
			var fee = number.Div(number.Mul(_fee, &difference), FeeDenominator)
			t.reserves[i].Sub(&new_balances[i], number.Div(number.Mul(fee, _admin_fee), FeeDenominator))
			number.FillBig(&t.reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
			new_balances[i].Sub(&new_balances[i], fee)
		}
		_ = t.get_D_mem(t.extra.RateMultipliers, new_balances, amp, &D2)
		mint_amount.Div(number.Mul(&token_supply, number.Sub(&D2, &D0)), &D0)
	} else {
		for i := 0; i < nCoins; i += 1 {
			t.reserves[i].Set(&new_balances[i])
			number.FillBig(&t.reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
		}
		mint_amount.Set(&D1)
	}
	t.LpSupply.Add(&t.LpSupply, &mint_amount)
	return &mint_amount, nil
}

func (t *PoolSimulator) ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error {
	for i := 0; i < t.numTokens; i++ {
		number.SafeAddZ(&t.reserves[i], &amounts[i], &t.reserves[i])
		number.SafeSubZ(&t.reserves[i], &feeAmounts[i], &t.reserves[i])
		number.FillBig(&t.reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	}

	t.LpSupply.Add(&t.LpSupply, mintAmount)

	return nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error) {
	dy, err := t.RemoveLiquidityOneCoinU256(number.SetFromBig(tokenAmount), i)
	if err != nil {
		return nil, err
	}
	return dy.ToBig(), nil
}

func (t *PoolSimulator) RemoveLiquidityOneCoinU256(tokenAmount *uint256.Int, i int) (*uint256.Int, error) {
	var dy, dyFee uint256.Int
	var err = t.CalculateWithdrawOneCoinU256(tokenAmount, i, &dy, &dyFee)
	if err != nil {
		return nil, err
	}
	t.reserves[i].Sub(&t.reserves[i], number.Add(&dy, number.Div(number.Mul(&dyFee, t.extra.AdminFee), FeeDenominator)))
	number.FillBig(&t.reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	t.LpSupply.Sub(&t.LpSupply, tokenAmount)
	return &dy, nil
}

func (t *PoolSimulator) ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error {
	t.reserves[i].Sub(&t.reserves[i], number.Add(dy, number.Div(number.Mul(dyFee, t.extra.AdminFee), FeeDenominator)))
	number.FillBig(&t.reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	t.LpSupply.Sub(&t.LpSupply, tokenAmount)
	return nil
}

// need to keep big.Int for interface method, will be removed later
func (t *PoolSimulator) GetVirtualPrice() (*big.Int, *big.Int, error) {
	var vPrice, d uint256.Int
	err := t.GetVirtualPriceU256(&vPrice, &d)
	if err != nil {
		return nil, nil, err
	}
	return vPrice.ToBig(), d.ToBig(), err
}

func (t *PoolSimulator) GetVirtualPriceU256(vPrice, D *uint256.Int) error {
	if t.LpSupply.IsZero() {
		return ErrDenominatorZero
	}
	var xp = xpMem(t.extra.RateMultipliers, t.reserves)
	var A = t._A()
	var err = t.getD(xp, A, D)
	if err != nil {
		return err
	}
	vPrice.Div(number.Mul(D, Precision), &t.LpSupply)
	return nil
}

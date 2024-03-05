package stableng

import (
	"math"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/holiman/uint256"
)

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
		xp[i].Div(number.SafeMul(&rates[i], &balances[i]), Precision)
	}
	return numTokens
}

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
	numTokensPow := uint256.NewInt(uint64(math.Pow(float64(t.numTokens), float64(t.numTokens))))

	for i := 0; i < MaxLoopLimit; i++ {
		// D_P: uint256 = D
		D_P.Set(D)

		// for x in _xp: D_P = D_P * D / x
		for j := range xp {
			D_P.Div(
				number.SafeMul(&D_P, D),
				&xp[j],
			)
		}
		// D_P /= pow_mod256(N_COINS, N_COINS)
		D_P.Div(&D_P, numTokensPow)

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
		if number.WithinDelta(D, &Dprev, 1) {
			return nil
		}
	}
	return ErrDDoesNotConverge
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
	var Ann = number.Mul(a, &t.numTokensU256)
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
	if Ann.IsZero() {
		return ErrZero
	}
	c.Div(
		number.SafeMul(number.SafeMul(c, &d), t.staticExtra.APrecision),
		number.SafeMul(Ann, &t.numTokensU256),
	)
	var b = number.SafeAdd(
		&s,
		number.Div(number.SafeMul(&d, t.staticExtra.APrecision), Ann),
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
		if number.WithinDelta(y, &yPrev, 1) {
			return nil
		}
	}
	return ErrAmountOutNotConverge
}

// Calculate the current output dy given input dx
func (t *PoolSimulator) GetDy(
	i int,
	j int,
	dx *uint256.Int,
	dCached *uint256.Int,
	dy *uint256.Int,
	adminFee *uint256.Int,
) error {
	var xp = xpMem(t.extra.RateMultipliers, t.reserves)
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x = number.SafeAdd(&xp[i], number.Div(number.SafeMul(dx, &t.extra.RateMultipliers[i]), Precision))

	// y: uint256 = self.get_y(i, j, x, xp)
	var y uint256.Int
	var err = t.getY(i, j, x, xp, dCached, &y)
	if err != nil {
		return err
	}

	// dy: uint256 = _xp[j] - y - 1  # -1 just in case there were some rounding errors
	number.SafeSubZ(&xp[j], number.AddUint64(&y, 1), dy)

	// dy_fee: uint256 = unsafe_div(
	//   dy * self._dynamic_fee(
	//      unsafe_div(_xp[i] + x, 2), unsafe_div(_xp[j] + y, 2), self.fee
	//   ),
	//   FEE_DENOMINATOR
	// )
	var dynamicFee, dyFee uint256.Int
	t._dynamic_fee(
		number.Div(number.SafeAdd(&xp[i], x), number.Number_2),
		number.Div(number.SafeAdd(&xp[j], &y), number.Number_2),
		t.extra.SwapFee,
		&dynamicFee,
	)
	dyFee.Div(
		number.SafeMul(dy, &dynamicFee),
		FeeDenominator,
	)

	// # Convert all to real units
	// dy = (dy - dy_fee) * PRECISION / rates[j]
	dy.Div(number.SafeMul(number.SafeSub(dy, &dyFee), Precision), &t.extra.RateMultipliers[j])

	adminFee.Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(&dyFee, t.extra.AdminFee),
				FeeDenominator,
			),
			Precision,
		),
		&t.extra.RateMultipliers[j],
	)

	return nil
}

func (t *PoolSimulator) _dynamic_fee(xpi *uint256.Int, xpj *uint256.Int, swapFee *uint256.Int, feeOutput *uint256.Int) {
	_offpeg_fee_multiplier := t.staticExtra.OffpegFeeMultiplier
	if _offpeg_fee_multiplier.Cmp(FeeDenominator) <= 0 {
		feeOutput.Set(swapFee)
		return
	}

	// xps2: uint256 = (xpi + xpj) ** 2
	sum := number.SafeAdd(xpi, xpj)
	prod := number.SafeMul(xpi, xpj)
	xps2 := number.SafeMul(sum, sum)
	feeOutput.Div(
		number.Mul(_offpeg_fee_multiplier, swapFee),
		number.Add(
			number.Div(
				number.SafeMul(
					number.SafeMul(
						number.Sub(_offpeg_fee_multiplier, FeeDenominator),
						number.Number_4,
					),
					prod,
				), xps2,
			),
			FeeDenominator,
		),
	)
}

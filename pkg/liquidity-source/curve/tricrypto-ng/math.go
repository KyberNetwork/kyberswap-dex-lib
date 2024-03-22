package tricryptong

import (
	"errors"
	"fmt"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// from contracts/main/CurveCryptoMathOptimized3.vy

// only sort slice of 3 elements (update the input array directly)
func sort(x []uint256.Int) {
	if x[0].Cmp(&x[1]) < 0 {
		tmp := number.Set(&x[0])
		x[0].Set(&x[1])
		x[1].Set(tmp)
	}
	if x[0].Cmp(&x[2]) < 0 {
		tmp := number.Set(&x[0])
		x[0].Set(&x[2])
		x[2].Set(tmp)
	}
	if x[1].Cmp(&x[2]) < 0 {
		tmp := number.Set(&x[1])
		x[1].Set(&x[2])
		x[2].Set(tmp)
	}
}

func geometric_mean(_x []uint256.Int) *uint256.Int {
	var result uint256.Int
	_geometric_mean(_x, &result)
	return &result
}

// calculates a geometric mean for three numbers.
func _geometric_mean(_x []uint256.Int, result *uint256.Int) {
	prod := number.Div(
		number.SafeMul(number.Div(number.SafeMul(&_x[0], &_x[1]), U_1e18), &_x[2]),
		U_1e18,
	)

	if prod.IsZero() {
		result.Clear()
		return
	}

	_cbrt(prod, result)
}

func newton_D(ANN *uint256.Int, gamma *uint256.Int, x_unsorted []uint256.Int, K0_prev *uint256.Int) (*uint256.Int, error) {
	/*
		@notice Finding the invariant via newtons method using good initial guesses.
		@dev ANN is higher by the factor A_MULTIPLIER
		@dev ANN is already A * N**N
		@param ANN the A * N**N value
		@param gamma the gamma value
		@param x_unsorted the array of coin balances (not sorted)
		@param K0_prev apriori for newton's method derived from get_y_int. Defaults
						to zero (no apriori)
	*/

	var x [NumTokens]uint256.Int
	for i := range x_unsorted {
		x[i].Set(&x_unsorted[i])
	}
	sort(x[:])

	// assert x[0] < max_value(uint256) / 10**18 * N_COINS**N_COINS  # dev: out of limits
	// assert x[0] > 0  # dev: empty pool
	if x[0].IsZero() || x[0].Cmp(MaxX) >= 0 {
		return nil, errors.New("unsafe values x[0]")
	}

	// S: uint256 = unsafe_add(unsafe_add(x[0], x[1]), x[2])
	// D: uint256 = 0
	S := number.Add(number.Add(&x[0], &x[1]), &x[2])
	D := uint256.NewInt(0)

	if K0_prev.IsZero() {
		// # Geometric mean of 3 numbers cannot be larger than the largest number
		// # so the following is safe to do:
		D = number.Mul(NumTokensU256, geometric_mean(x[:]))
	} else {
		if S.Cmp(U_1e36) > 0 {
			_cbrt(
				number.Mul(
					number.Div(
						number.Mul(number.Div(number.Mul(&x[0], &x[1]), U_1e36), &x[2]),
						K0_prev,
					),
					U_27e12),
				D)
		} else if S.Cmp(U_1e24) > 0 {
			_cbrt(
				number.Mul(
					number.Div(
						number.Mul(number.Div(number.Mul(&x[0], &x[1]), U_1e24), &x[2]),
						K0_prev,
					),
					U_27e6),
				D)
		} else {
			_cbrt(
				number.Mul(
					number.Div(
						number.Mul(number.Div(number.Mul(&x[0], &x[1]), U_1e18), &x[2]),
						K0_prev,
					),
					U_27,
				),
				D)
		}
		// # D not zero here if K0_prev > 0, and we checked if x[0] is gt 0.
	}

	var D_prev, K0, diff uint256.Int
	for i := 0; i < 255; i += 1 {
		D_prev.Set(D)

		// # K0 = 10**18 * x[0] * N_COINS / D * x[1] * N_COINS / D * x[2] * N_COINS / D
		K0.Div(
			number.Mul(
				number.Mul(
					number.Div(
						number.Mul(
							number.Mul(
								number.Div(
									number.Mul(
										number.Mul(U_1e18, &x[0]), NumTokensU256,
									),
									D,
								),
								&x[1],
							),
							NumTokensU256,
						),
						D,
					),
					&x[2],
				),
				NumTokensU256,
			),
			D,
		)
		// # <-------- We can convert the entire expression using unsafe math.
		// #   since x_i is not too far from D, so overflow is not expected. Also
		// #      D > 0, since we proved that already. unsafe_div is safe. K0 > 0
		// #        since we can safely assume that D < 10**18 * x[0]. K0 is also
		// #                            in the range of 10**18 (it's a property).

		// _g1k0 = unsafe_add(gamma, 10**18)  # <--------- safe to do unsafe_add.
		// if _g1k0 > K0:  #       The following operations can safely be unsafe.
		// 		_g1k0 = unsafe_add(unsafe_sub(_g1k0, K0), 1)
		// else:
		// 		_g1k0 = unsafe_add(unsafe_sub(K0, _g1k0), 1)

		var _g1k0 = number.Add(gamma, U_1e18)
		if _g1k0.Cmp(&K0) > 0 {
			_g1k0.AddUint64(number.Sub(_g1k0, &K0), 1)
		} else {
			_g1k0.AddUint64(number.Sub(&K0, _g1k0), 1)
		}

		// # D / (A * N**N) * _g1k0**2 / gamma**2
		// # mul1 = 10**18 * D / gamma * _g1k0 / gamma * _g1k0 * A_MULTIPLIER / ANN
		var mul1 = number.Div(
			number.Mul(
				number.Mul(
					number.Div(
						number.Mul(
							number.Div(number.Mul(U_1e18, D), gamma), _g1k0,
						),
						gamma,
					),
					_g1k0,
				),
				AMultiplier,
			),
			ANN,
		)
		// # <------ Since D > 0, gamma is small, _g1k0 is small, the rest are
		// #        non-zero and small constants, and D has a cap in this method,
		// #                    we can safely convert everything to unsafe maths.

		// # 2*N*K0 / _g1k0
		// # mul2 = (2 * 10**18) * N_COINS * K0 / _g1k0
		var mul2 = number.Div(
			number.Mul(number.SafeMul(U_2e18, NumTokensU256), &K0), _g1k0,
		)
		// # <--------------- K0 is approximately around D, which has a cap of
		// #      10**15 * 10**18 + 1, since we get that in get_y which is called
		// #    with newton_D. _g1k0 > 0, so the entire expression can be unsafe.

		// # neg_fprime: uint256 = (S + S * mul2 / 10**18) + mul1 * N_COINS / K0 - mul2 * D / 10**18
		var neg_fprime = number.Sub(
			number.Add(
				number.Add(S, number.Div(number.Mul(S, mul2), U_1e18)),
				number.Div(number.Mul(mul1, NumTokensU256), &K0),
			),
			number.Div(number.Mul(mul2, D), U_1e18),
		)
		// # <--- mul1 is a big number but not huge: safe to unsafely multiply
		// # with N_coins. neg_fprime > 0 if this expression executes.
		// # mul2 is in the range of 10**18, since K0 is in that range, S * mul2
		// # is safe. The first three sums can be done using unsafe math safely
		// # and since the final expression will be small since mul2 is small, we
		// # can safely do the entire expression unsafely.

		// # D -= f / fprime
		// # D * (neg_fprime + S) / neg_fprime
		var D_plus = number.Div(number.SafeMul(D, number.Add(neg_fprime, S)), neg_fprime)

		// # D*D / neg_fprime
		var D_minus = number.Div(number.SafeMul(D, D), neg_fprime)

		// # Since we know K0 > 0, and neg_fprime > 0, several unsafe operations
		// # are possible in the following. Also, (10**18 - K0) is safe to mul.
		// # So the only expressions we keep safe are (D_minus + ...) and (D * ...)
		if U_1e18.Cmp(&K0) > 0 {
			// # D_minus += D * (mul1 / neg_fprime) / 10**18 * (10**18 - K0) / K0
			D_minus = number.SafeAdd(D_minus,
				number.Div(
					number.Mul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), U_1e18), number.Sub(U_1e18, &K0)),
					&K0,
				),
			)
		} else {
			// # D_minus -= D * (mul1 / neg_fprime) / 10**18 * (K0 - 10**18) / K0
			D_minus = number.SafeSub(D_minus,
				number.Div(
					number.Mul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), U_1e18), number.Sub(&K0, U_1e18)),
					&K0,
				),
			)
		}
		if D_plus.Cmp(D_minus) > 0 {
			// D = unsafe_sub(D_plus, D_minus)  # <--------- Safe since we check.
			D = number.Sub(D_plus, D_minus)
		} else {
			// D = unsafe_div(unsafe_sub(D_minus, D_plus), 2)
			D = number.Div(number.Sub(D_minus, D_plus), number.Number_2)
		}

		if D.Cmp(&D_prev) > 0 {
			diff.Sub(D, &D_prev)
		} else {
			diff.Sub(&D_prev, D)
		}

		temp := U_1e16
		if D.Cmp(U_1e16) > 0 {
			temp = D
		}
		if number.Mul(&diff, U_1e14).Cmp(temp) < 0 {
			// # Test that we are safe with the next get_y
			for i := range x {
				var frac = number.Div(number.Mul(&x[i], U_1e18), D)
				if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
					return nil, errors.New("unsafe values x[i]")
				}
			}
			return D, nil
		}
	}
	return nil, errors.New("did not converge")
}

func get_y(
	_ann, _gamma *uint256.Int, x []uint256.Int, _D *uint256.Int, i int,
	//output
	y, K0 *uint256.Int,
) error {
	if _ann.Cmp(MinA) < 0 || _ann.Cmp(MaxA) > 0 {
		return errors.New("unsafe values A")
	}

	if _gamma.Cmp(MinGamma) < 0 || _gamma.Cmp(MaxGamma) > 0 {
		return errors.New("unsafe values gamma")
	}

	if _D.Cmp(MinD) < 0 || _D.Cmp(MaxD) > 0 {
		return errors.New("unsafe values D")
	}

	for k := 0; k < NumTokens; k++ {
		if k == i {
			continue
		}
		frac := number.Div(number.Mul(&x[k], U_1e18), _D)
		if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
			return fmt.Errorf("unsafe values x[%d] %s", i, frac.Dec())
		}
	}

	j := 0
	k := 0
	if i == 0 {
		j = 1
		k = 2
	} else if i == 1 {
		j = 0
		k = 2
	} else if i == 2 {
		j = 0
		k = 1
	}

	// ANN: int256 = convert(_ANN, int256)
	// gamma: int256 = convert(_gamma, int256)
	// D: int256 = convert(_D, int256)
	// x_j: int256 = convert(x[j], int256)
	// x_k: int256 = convert(x[k], int256)
	// gamma2: int256 = unsafe_mul(gamma, gamma)

	ann := i256.SafeToInt256(_ann)
	gamma := i256.SafeToInt256(_gamma)
	D := i256.SafeToInt256(_D)
	x_j := i256.SafeToInt256(&x[j])
	x_k := i256.SafeToInt256(&x[k])
	gamma2 := i256.Mul(gamma, gamma)
	AMultiplier_ := i256.SafeToInt256(AMultiplier)

	// a: int256 = 10**36 / 27
	a := i256.Set(TenPow36Div27)

	// # 10**36/9 + 2*10**18*gamma/27 - D**2/x_j*gamma**2*ANN/27**2/convert(A_MULTIPLIER, int256)/x_k
	b := i256.Sub(
		i256.Add(TenPow36Div9, i256.Div(i256.Mul(I_2e18, gamma), I_27)),
		i256.Div(
			i256.Div(
				i256.Div(
					i256.Mul(
						i256.Mul(
							i256.Div(i256.Mul(D, D), x_j),
							gamma2,
						),
						ann,
					),
					I_27x27,
				),
				AMultiplier_,
			),
			x_k,
		),
	)

	// # 10**36/9 + gamma*(gamma + 4*10**18)/27 + gamma**2*(x_j+x_k-D)/D*ANN/27/convert(A_MULTIPLIER, int256)
	c := i256.Add(
		i256.Add(
			TenPow36Div9,
			i256.Div(i256.Mul(gamma, i256.Add(gamma, I_4e18)), I_27),
		),
		i256.Div(
			i256.Div(
				i256.Mul(
					i256.Div(
						i256.Mul(gamma2,
							i256.Sub(i256.Add(x_j, x_k), D)),
						D,
					),
					ann,
				),
				I_27,
			),
			AMultiplier_,
		),
	)

	// d: int256 = i256.Div(unsafe_add(10**18, gamma)**2, 27)
	tmp := i256.Add(I_1e18, gamma)
	d := i256.Div(i256.Mul(tmp, tmp), I_27)

	// d0: int256 = abs(unsafe_mul(3, a) * c / b - b)  # <------------ a is smol.
	d0 := i256.Abs(
		i256.Sub(
			i256.Div(
				i256.Mul(
					i256.Mul(i256.Number_3, a),
					c),
				b),
			b))

	var divider int256.Int
	divider.SetUint64(1)
	if d0.Cmp(I_1e48) > 0 {
		divider.Set(I_1e30)
	} else if d0.Cmp(I_1e44) > 0 {
		divider.Set(I_1e26)
	} else if d0.Cmp(I_1e40) > 0 {
		divider.Set(I_1e22)
	} else if d0.Cmp(I_1e36) > 0 {
		divider.Set(I_1e18)
	} else if d0.Cmp(I_1e32) > 0 {
		divider.Set(I_1e14)
	} else if d0.Cmp(I_1e28) > 0 {
		divider.Set(I_1e10)
	} else if d0.Cmp(I_1e24) > 0 {
		divider.Set(I_1e6)
	} else if d0.Cmp(I_1e20) > 0 {
		divider.Set(I_1e2)
	}

	var additional_prec *int256.Int
	if i256.Abs(a).Cmp(i256.Abs(b)) > 0 {
		additional_prec = i256.Abs(i256.Div(a, b))
		a = i256.Div(i256.Mul(a, additional_prec), &divider)
		b = i256.Div(i256.Mul(b, additional_prec), &divider)
		c = i256.Div(i256.Mul(c, additional_prec), &divider)
		d = i256.Div(i256.Mul(d, additional_prec), &divider)
	} else {
		additional_prec = i256.Abs(i256.Div(b, a))
		a = i256.Div(i256.Div(a, additional_prec), &divider)
		b = i256.Div(i256.Div(b, additional_prec), &divider)
		c = i256.Div(i256.Div(c, additional_prec), &divider)
		d = i256.Div(i256.Div(d, additional_prec), &divider)
	}

	// # 3*a*c/b - b
	_3ac := i256.Mul(i256.Mul(i256.Number_3, a), c)
	delta0 := i256.Sub(i256.Div(_3ac, b), b)

	// # 9*a*c/b - 2*b - 27*a**2/b*d/b
	delta1 := i256.Sub(
		i256.Sub(
			i256.Div(i256.Mul(i256.Number_3, _3ac), b),
			i256.Mul(i256.Number_2, b),
		),
		i256.Div(i256.Mul(
			i256.Div(
				i256.Mul(I_27, i256.Mul(a, a)),
				b),
			d),
			b),
	)

	// # delta1**2 + 4*delta0**2/b*delta0
	sqrt_arg := i256.Add(
		i256.Mul(delta1, delta1),
		i256.Mul(
			i256.Div(
				i256.Mul(i256.Number_4, i256.Mul(delta0, delta0)),
				b),
			delta0),
	)

	sqrt_val := int256.NewInt(0)
	if sqrt_arg.Sign() > 0 {
		sqrt_val.Sqrt(sqrt_arg)
	} else {
		return newton_y(_ann, _gamma, x, _D, i, y)
	}

	b_cbrt := new(int256.Int)
	if b.Sign() >= 0 {
		i256.SafeConvertToInt256(cbrt(i256.SafeConvertToUInt256(b)), b_cbrt)
	} else {
		i256.SafeConvertToInt256(cbrt(i256.SafeConvertToUInt256(i256.Neg(b))), b_cbrt)
		b_cbrt = i256.Neg(b_cbrt)
	}

	second_cbrt := new(int256.Int)
	if delta1.Sign() > 0 {
		// # convert(self._cbrt(convert((delta1 + sqrt_val), uint256)/2), int256)
		i256.SafeConvertToInt256(
			cbrt(number.Div(
				i256.SafeConvertToUInt256(i256.Add(delta1, sqrt_val)),
				number.Number_2)),
			second_cbrt)
	} else {
		i256.SafeConvertToInt256(
			cbrt(number.Div(
				i256.SafeConvertToUInt256(new(int256.Int).Neg(i256.Sub(delta1, sqrt_val))),
				number.Number_2)),
			second_cbrt)
		second_cbrt = new(int256.Int).Neg(second_cbrt)
	}

	// # b_cbrt*b_cbrt/10**18*second_cbrt/10**18
	C1 := i256.Div(
		i256.Mul(i256.Div(i256.Mul(b_cbrt, b_cbrt), I_1e18), second_cbrt),
		I_1e18,
	)

	// # (b + b*delta0/C1 - C1)/3
	root_K0 := i256.Div(
		i256.Sub(
			i256.Add(b,
				i256.Div(i256.Mul(b, delta0), C1)),
			C1),
		i256.Number_3)

	// # D*D/27/x_k*D/x_j*root_K0/a
	root := i256.Div(
		i256.Mul(
			i256.Div(
				i256.Mul(
					i256.Div(
						i256.Div(
							i256.Mul(D, D),
							I_27),
						x_k),
					D),
				x_j),
			root_K0),
		a,
	)

	y.Set(i256.SafeConvertToUInt256(root))
	K0.Set(i256.SafeConvertToUInt256(i256.Div(i256.Mul(I_1e18, root_K0), a)))

	frac := number.Div(number.Mul(y, U_1e18), _D)
	if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
		return ErrUnsafeY
	}

	return nil
}

func cbrt(x *uint256.Int) *uint256.Int {
	var res uint256.Int
	_cbrt(x, &res)
	return &res
}

func _cbrt(x *uint256.Int, a *uint256.Int) {
	var xx *uint256.Int
	if x.Cmp(CbrtConst1) >= 0 {
		xx = x
	} else if x.Cmp(CbrtConst2) >= 0 {
		xx = number.Mul(x, U_1e18)
	} else {
		xx = number.Mul(x, U_1e36)
	}

	// log2x: int256 = convert(self._snekmate_log_2(xx, False), int256)
	_log2x := i256.SafeToInt256(snekmate_log_2(xx, false))
	log2x := i256.SafeConvertToUInt256(_log2x)

	// # When we divide log2x by 3, the remainder is (log2x % 3).
	// # So if we just multiply 2**(log2x/3) and discard the remainder to calculate our
	// # guess, the newton method will need more iterations to converge to a solution,
	// # since it is missing that precision. It's a few more calculations now to do less
	// # calculations later:
	// # pow = log2(x) // 3
	// # remainder = log2(x) % 3
	// # initial_guess = 2 ** pow * cbrt(2) ** remainder
	// # substituting -> 2 = 1.26 ≈ 1260 / 1000, we get:
	// #
	// # initial_guess = 2 ** pow * 1260 ** remainder // 1000 ** remainder

	remainder := new(uint256.Int).Mod(log2x, number.Number_3)
	a.Div(
		number.Mul(
			pow_mod256(number.Number_2, number.Div(log2x, number.Number_3)), //# <- pow
			pow_mod256(uint256.NewInt(1260), remainder),
		),
		pow_mod256(uint256.NewInt(1000), remainder),
	)

	// # Because we chose good initial values for cube roots, 7 newton raphson iterations
	// # are just about sufficient. 6 iterations would result in non-convergences, and 8
	// # would be one too many iterations. Without initial values, the iteration count
	// # can go up to 20 or greater. The iterations are unrolled. This reduces gas costs
	// # but takes up more bytecode:
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)
	a.Div(number.Add(number.Mul(number.Number_2, a), number.Div(xx, number.Mul(a, a))), number.Number_3)

	if x.Cmp(CbrtConst1) >= 0 {
		a = number.Mul(a, U_1e12)
	} else if x.Cmp(CbrtConst2) >= 0 {
		a = number.Mul(a, U_1e6)
	}
}

func pow_mod256(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Exp(x, y)
}

func snekmate_log_2(x *uint256.Int, roundup bool) *uint256.Int {
	var result uint256.Int
	_snekmate_log_2(x, roundup, &result)
	return &result
}

func _snekmate_log_2(x *uint256.Int, roundup bool, result *uint256.Int) {
	/*
	   @notice An `internal` helper function that returns the log in base 2
	        of `x`, following the selected rounding direction.
	   @dev This implementation is derived from Snekmate, which is authored
	        by pcaversaccio (Snekmate), distributed under the AGPL-3.0 license.
	        https://github.com/pcaversaccio/snekmate
	   @dev Note that it returns 0 if given 0. The implementation is
	        inspired by OpenZeppelin's implementation here:
	        https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/utils/math/Math.sol.
	   @param x The 32-byte variable.
	   @param roundup The Boolean variable that specifies whether
	          to round up or not. The default `False` is round down.
	   @return uint256 The 32-byte calculation result.
	*/
	value := number.Set(x)
	result.Clear()

	// # The following lines cannot overflow because we have the well-known
	// # decay behaviour of `log_2(max_value(uint256)) < max_value(uint256)`.
	if !new(uint256.Int).Rsh(x, 128).IsZero() {
		value.Rsh(x, 128)
		result.SetUint64(128)
	}
	if !new(uint256.Int).Rsh(value, 64).IsZero() {
		value.Rsh(value, 64)
		result.Add(result, uint256.NewInt(64))
	}
	if !new(uint256.Int).Rsh(value, 32).IsZero() {
		value.Rsh(value, 32)
		result.Add(result, uint256.NewInt(32))
	}
	if !new(uint256.Int).Rsh(value, 16).IsZero() {
		value.Rsh(value, 16)
		result.Add(result, uint256.NewInt(16))
	}
	if !new(uint256.Int).Rsh(value, 8).IsZero() {
		value.Rsh(value, 8)
		result.Add(result, uint256.NewInt(8))
	}
	if !new(uint256.Int).Rsh(value, 4).IsZero() {
		value.Rsh(value, 4)
		result.Add(result, uint256.NewInt(4))
	}
	if !new(uint256.Int).Rsh(value, 2).IsZero() {
		value.Rsh(value, 2)
		result.Add(result, uint256.NewInt(2))
	}
	if !new(uint256.Int).Rsh(value, 1).IsZero() {
		result.Add(result, uint256.NewInt(1))
	}
	const1 := new(uint256.Int).Lsh(number.Number_1, uint(result.Uint64()))
	if roundup && const1.Cmp(x) < 0 {
		result.Add(result, uint256.NewInt(1))
	}
}

// Calculate x[i] given A, gamma, xp and D using newton's method.
func newton_y(
	ann, gamma *uint256.Int, x []uint256.Int, D *uint256.Int, i int,
	//output
	y *uint256.Int,
) error {
	y.Div(D, NumTokensU256)
	var K0i, Si uint256.Int
	K0i.Set(U_1e18)
	Si.Clear()

	var xSorted [NumTokens]uint256.Int
	for j := 0; j < NumTokens; j += 1 {
		xSorted[j].Set(&x[j])
	}
	xSorted[i].Clear()
	sort(xSorted[:])

	// convergence_limit: uint256 = max(max(x_sorted[0] / 10**14, D / 10**14), 100)
	var convergenceLimit = number.Div(&xSorted[0], U_1e14)
	var temp = number.Div(D, U_1e14)
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if uint256.NewInt(100).Cmp(convergenceLimit) > 0 {
		convergenceLimit.SetUint64(100)
	}

	for j := 2; j < NumTokens+1; j += 1 {
		var _x = &xSorted[NumTokens-j]
		if _x.IsZero() {
			return ErrZero
		}
		y.Div(number.SafeMul(y, D), number.SafeMul(_x, NumTokensU256))
		Si.Add(&Si, _x)
	}
	for j := 0; j < NumTokens-1; j += 1 {
		K0i.Div(number.SafeMul(number.SafeMul(&K0i, &xSorted[j]), NumTokensU256), D)
	}

	var yPrev, K0, S, _g1k0, mul1, yfprime uint256.Int
	De18 := number.SafeMul(D, U_1e18)

	for j := 0; j < 255; j += 1 {
		yPrev.Set(y)
		K0.Div(number.SafeMul(number.SafeMul(&K0i, y), NumTokensU256), D)
		S.Add(&Si, y)

		_g1k0.Add(gamma, U_1e18)
		if _g1k0.Cmp(&K0) > 0 {
			number.SafeAddZ(number.SafeSub(&_g1k0, &K0), number.Number_1, &_g1k0)
		} else {
			number.SafeAddZ(number.SafeSub(&K0, &_g1k0), number.Number_1, &_g1k0)
		}

		// mul1 = 10**18 * D / gamma * _g1k0 / gamma * _g1k0 * A_MULTIPLIER / ANN
		mul1.Div(
			number.SafeMul(
				number.Div(
					number.SafeMul(
						number.Div(De18, gamma),
						&_g1k0,
					), gamma,
				),
				number.SafeMul(&_g1k0, AMultiplier),
			), ann)

		// mul2 = 10**18 + (2 * 10**18) * K0 / _g1k0
		var mul2 = number.SafeAdd(
			U_1e18,
			number.Div(number.SafeMul(U_2e18, &K0), &_g1k0),
		)

		// yfprime = 10**18 * y + S * mul2 + mul1
		number.SafeAddZ(
			number.SafeAdd(number.SafeMul(U_1e18, y), number.SafeMul(&S, mul2)),
			&mul1, &yfprime)
		var _dyfprime = number.SafeMul(D, mul2)
		if yfprime.Cmp(_dyfprime) < 0 {
			y.Div(&yPrev, number.Number_2)
			continue
		} else {
			number.SafeSubZ(&yfprime, _dyfprime, &yfprime)
		}

		if y.IsZero() {
			return ErrZero
		}

		var fprime = number.Div(&yfprime, y)

		if fprime.IsZero() {
			return ErrZero
		}

		var yMinus = number.Div(&mul1, fprime)
		var yPlus = number.SafeAdd(number.Div(
			number.SafeAdd(&yfprime, De18),
			fprime),
			number.Div(number.SafeMul(yMinus, U_1e18), &K0))
		number.SafeAddZ(yMinus, number.Div(number.SafeMul(U_1e18, &S), fprime), yMinus)
		if yPlus.Cmp(yMinus) < 0 {
			y.Div(&yPrev, number.Number_2)
		} else {
			number.SafeSubZ(yPlus, yMinus, y)
		}
		var diff uint256.Int
		if y.Cmp(&yPrev) > 0 {
			diff.Sub(y, &yPrev)
		} else {
			diff.Sub(&yPrev, y)
		}
		var t = number.Div(y, U_1e14)
		if convergenceLimit.Cmp(t) > 0 {
			t = convergenceLimit
		}
		if diff.Cmp(t) < 0 {
			var frac = number.Div(number.SafeMul(y, U_1e18), D)
			if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
				return ErrUnsafeY
			}
			return nil
		}
	}
	return ErrYDoesNotConverge
}

func reductionCoefficient(x []uint256.Int, feeGamma *uint256.Int, K *uint256.Int) error {
	var S uint256.Int
	number.SafeAddZ(number.SafeAdd(&x[0], &x[1]), &x[2], &S)
	if S.IsZero() {
		return ErrZero
	}

	K.Div(number.SafeMul(number.SafeMul(U_1e18, NumTokensU256), &x[0]), &S)
	K.Div(number.SafeMul(number.SafeMul(K, NumTokensU256), &x[1]), &S)
	K.Div(number.SafeMul(number.SafeMul(K, NumTokensU256), &x[2]), &S)

	if !feeGamma.IsZero() {
		K.Div(
			number.SafeMul(feeGamma, U_1e18),
			number.SafeSub(number.SafeAdd(feeGamma, U_1e18), K))
	}
	return nil
}

func (t *PoolSimulator) _A_gamma() (*uint256.Int, *uint256.Int) {
	var A, gamma uint256.Int
	t._A_gamma_inplace(&A, &gamma)
	return &A, &gamma
}

func (t *PoolSimulator) _A_gamma_inplace(A, gamma *uint256.Int) {
	var t1 = t.Extra.FutureAGammaTime
	A.Set(t.Extra.FutureA)
	gamma.Set(t.Extra.FutureGamma)
	var now = time.Now().Unix()
	if now < t1 {
		var A0 = t.Extra.InitialA
		var gamma0 = t.Extra.InitialGamma
		var t0 = t.Extra.InitialAGammaTime
		t1 -= t0
		t0 = now - t0
		var t2 = t1 - t0
		A.Div(number.Add(
			number.Mul(A0, uint256.NewInt(uint64(t2))),
			number.Mul(A, uint256.NewInt(uint64(t0))),
		), uint256.NewInt(uint64(t1)))

		gamma.Div(number.Add(
			number.Mul(gamma0, uint256.NewInt(uint64(t2))),
			number.Mul(gamma, uint256.NewInt(uint64(t0))),
		), uint256.NewInt(uint64(t1)))
	}
}

func _snekmate_wad_exp(x *int256.Int) (*uint256.Int, error) {
	/*
	   @dev Calculates the natural exponential function of a signed integer with
	        a precision of 1e18.
	   @notice Note that this function consumes about 810 gas units. The implementation
	           is inspired by Remco Bloemen's implementation under the MIT license here:
	           https://xn--2-umb.com/22/exp-ln.
	   @dev This implementation is derived from Snekmate, which is authored
	        by pcaversaccio (Snekmate), distributed under the AGPL-3.0 license.
	        https://github.com/pcaversaccio/snekmate
	   @param x The 32-byte variable.
	   @return int256 The 32-byte calculation result.
	*/
	value := i256.Set(x)

	// # If the result is `< 0.5`, we return zero. This happens when we have the following:
	// # "x <= floor(log(0.5e18) * 1e18) ~ -42e18".
	if x.Cmp(i256.MustFromDecimal("-42139678854452767551")) <= 0 {
		return uint256.NewInt(0), nil
	}

	// # When the result is "> (2 ** 255 - 1) / 1e18" we cannot represent it as a signed integer.
	// # This happens when "x >= floor(log((2 ** 255 - 1) / 1e18) * 1e18) ~ 135".
	if x.Cmp(i256.MustFromDecimal("135305999368893231589")) >= 0 {
		return nil, errors.New("wad_exp overflow")
	}

	// # `x` is now in the range "(-42, 136) * 1e18". Convert to "(-42, 136) * 2 ** 96" for higher
	// # intermediate precision and a binary base. This base conversion is a multiplication with
	// # "1e18 / 2 ** 96 = 5 ** 18 / 2 ** 78".
	value = i256.Div(i256.Lsh(x, 78), i256.MustFromDecimal("3814697265625"))

	// # Reduce the range of `x` to "(-½ ln 2, ½ ln 2) * 2 ** 96" by factoring out powers of two
	// # so that "exp(x) = exp(x') * 2 ** k", where `k` is a signer integer. Solving this gives
	// # "k = round(x / log(2))" and "x' = x - k * log(2)". Thus, `k` is in the range "[-61, 195]".
	k := i256.Rsh(
		i256.Add(
			i256.Div(
				i256.Lsh(value, 96),
				i256.MustFromDecimal("54916777467707473351141471128")),
			i256.MustFromDecimal("39614081257132168796771975168")),
		96)
	value = i256.Sub(value, i256.Mul(k, i256.MustFromDecimal("54916777467707473351141471128")))

	// # Evaluate using a "(6, 7)"-term rational approximation. Since `p` is monic,
	// # we will multiply by a scaling factor later.
	y := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Add(value, i256.MustFromDecimal("1346386616545796478920950773328")), value),
			96),
		i256.MustFromDecimal("57155421227552351082224309758442"))
	p := i256.Add(
		i256.Mul(
			i256.Add(
				i256.Rsh(
					i256.Mul(
						i256.Sub(i256.Add(y, value),
							i256.MustFromDecimal("94201549194550492254356042504812")),
						y),
					96),
				i256.MustFromDecimal("28719021644029726153956944680412240")),
			value),
		i256.Lsh(i256.MustFromDecimal("4385272521454847904659076985693276"), 96),
	)

	// # We leave `p` in the "2 ** 192" base so that we do not have to scale it up
	// # again for the division.
	q := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Sub(value, i256.MustFromDecimal("2855989394907223263936484059900")), value),
			96),
		i256.MustFromDecimal("50020603652535783019961831881945"))
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("533845033583426703283633433725380"))
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("3604857256930695427073651918091429"))
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("14423608567350463180887372962807573"))
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("26449188498355588339934803723976023"))

	// # The polynomial `q` has no zeros in the range because all its roots are complex.
	// # No scaling is required, as `p` is already "2 ** 96" too large. Also,
	// # `r` is in the range "(0.09, 0.25) * 2**96" after the division.
	r := i256.Div(p, q)

	// # To finalise the calculation, we have to multiply `r` by:
	// #   - the scale factor "s = ~6.031367120",
	// #   - the factor "2 ** k" from the range reduction, and
	// #   - the factor "1e18 / 2 ** 96" for the base conversion.
	// # We do this all at once, with an intermediate result in "2**213" base,
	// # so that the final right shift always gives a positive value.

	// # Note that to circumvent Vyper's safecast feature for the potentially
	// # negative parameter value `r`, we first convert `r` to `bytes32` and
	// # subsequently to `uint256`. Remember that the EVM default behaviour is
	// # to use two's complement representation to handle signed integers.
	tmp := number.Mul(i256.UnsafeToUInt256(r), uint256.MustFromDecimal("3822833074963236453042738258902158003155416615667"))
	n := 195 - k.Int64()
	return new(uint256.Int).Rsh(tmp, uint(n)), nil
}

func get_p(_xp [NumTokens]uint256.Int, _D, A, gamma *uint256.Int, out []uint256.Int) error {
	/*
		@notice Calculates dx/dy.
		@dev Output needs to be multiplied with price_scale to get the actual value.
		@param _xp Balances of the pool.
		@param _D Current value of D.
		@param _A_gamma Amplification coefficient and gamma.
	*/

	// assert _D > 10**17 - 1 and _D < 10**15 * 10**18 + 1  # dev: unsafe D values
	if _D.Cmp(MinD) < 0 || _D.Cmp(MaxD) > 0 {
		return errors.New("unsafe D values")
	}

	// # K0 = P * N**N / D**N.
	// # K0 is dimensionless and has 10**36 precision:
	K0 := number.Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(
					number.Div(
						number.SafeMul(U_27, number.SafeMul(&_xp[0], &_xp[1])),
						_D),
					&_xp[2]),
				_D),
			U_1e36),
		_D,
	)

	// # GK0 is in 10**36 precision and is dimensionless.
	// # GK0 = (
	// #     2 * _K0 * _K0 / 10**36 * _K0 / 10**36
	// #     + (gamma + 10**18)**2
	// #     - (_K0 * _K0 / 10**36 * (2 * gamma + 3 * 10**18) / 10**18)
	// # )
	// # GK0 is always positive. So the following should never revert:
	GK0 := number.SafeSub(
		number.SafeAdd(
			number.Div(
				number.SafeMul(
					number.Div(
						number.SafeMul(number.Number_2, number.SafeMul(K0, K0)),
						U_1e36),
					K0),
				U_1e36),
			pow_mod256(number.Add(gamma, U_1e18), number.Number_2),
		),
		number.Div(
			number.SafeMul(
				number.Div(pow_mod256(K0, number.Number_2), U_1e36),
				number.Add(number.Mul(number.Number_2, gamma), U_3e18),
			),
			U_1e18,
		),
	)

	// # NNAG2 = N**N * A * gamma**2
	NNAG2 := number.Div(
		number.Mul(A, pow_mod256(gamma, number.Number_2)),
		AMultiplier)

	// # denominator = (GK0 + NNAG2 * x / D * _K0 / 10**36)
	denominator := number.SafeAdd(GK0,
		number.Div(number.SafeMul(number.Div(number.SafeMul(NNAG2, &_xp[0]), _D), K0), U_1e36),
	)

	// # p_xy = x * (GK0 + NNAG2 * y / D * K0 / 10**36) / y * 10**18 / denominator
	// # p_xz = x * (GK0 + NNAG2 * z / D * K0 / 10**36) / z * 10**18 / denominator
	// # p is in 10**18 precision.
	out[0].Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(
					&_xp[0],
					number.SafeAdd(GK0, number.Div(number.SafeMul(number.Div(number.SafeMul(NNAG2, &_xp[1]), _D), K0), U_1e36)),
				),
				&_xp[1],
			),
			U_1e18,
		),
		denominator,
	)
	out[1].Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(
					&_xp[0],
					number.SafeAdd(GK0, number.Div(number.SafeMul(number.Div(number.SafeMul(NNAG2, &_xp[2]), _D), K0), U_1e36)),
				),
				&_xp[2],
			),
			U_1e18,
		),
		denominator,
	)

	return nil
}

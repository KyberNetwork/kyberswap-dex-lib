package twocryptong

import (
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// from contracts/main/CurveCryptoMathOptimized3.vy

// only sort slice of 2 elements (update the input array directly)
func sort(x []uint256.Int) {
	if x[0].Cmp(&x[1]) < 0 {
		tmp := number.Set(&x[0])
		x[0].Set(&x[1])
		x[1].Set(tmp)
	}
}

// calculates a geometric mean for two numbers.
func geometric_mean(_x []uint256.Int) *uint256.Int {
	var result uint256.Int
	_geometric_mean(_x, &result)
	return &result
}

// calculates a geometric mean for two numbers into provided result.
func _geometric_mean(_x []uint256.Int, result *uint256.Int) {
	result.Sqrt(result.Mul(&_x[0], &_x[1]))
}

// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveCryptoMathOptimized2.vy#L376
func newton_D(ANN *uint256.Int, gamma *uint256.Int, x_unsorted []uint256.Int, K0_prev *uint256.Int) (*uint256.Int,
	error) {
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

	if ANN.Cmp(MinA) < 0 || ANN.Cmp(MaxA) > 0 {
		return nil, ErrUnsafeA
	}

	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
		return nil, ErrUnsafeGamma
	}

	var x [NumTokens]uint256.Int
	for i := range x_unsorted {
		x[i].Set(&x_unsorted[i])
	}
	sort(x[:])

	// assert x[0] > 10**9 - 1 and x[0] < 10**15 * 10**18 + 1  # dev: unsafe values x[0]
	if x[0].Cmp(MinX0) < 0 || x[0].Cmp(MaxX1) > 0 {
		return nil, ErrUnsafeX0
	}
	// assert unsafe_div(x[1] * 10**18, x[0]) > 10**14 - 1  # dev: unsafe values x[i] (input)
	if number.Div(number.Mul(&x[1], U_1e18), &x[0]).Cmp(U_1e14) < 0 {
		return nil, ErrUnsafeXi
	}

	// S: uint256 = unsafe_add(x[0], x[1])
	// D: uint256 = 0
	S := number.Add(&x[0], &x[1])
	D := new(uint256.Int)

	if K0_prev.IsZero() {
		// # Geometric mean of 3 numbers cannot be larger than the largest number
		// # so the following is safe to do:
		D = number.Mul(NumTokensU256, geometric_mean(x[:]))
	} else {
		// D = isqrt(unsafe_mul(unsafe_div(unsafe_mul(unsafe_mul(4, x[0]), x[1]), K0_prev), 10**18))
		// if S < D:
		//     D = S
		D.Sqrt(number.Mul(number.Div(number.Mul(number.Mul(number.Number_4, &x[0]), &x[1]), K0_prev), U_1e18))
		if S.Cmp(D) < 0 {
			D.Set(S)
		}
	}

	var __g1k0 = number.Add(gamma, U_1e18)
	var D_prev, K0, diff uint256.Int
	for i := 0; i < 255; i += 1 {
		if D.Sign() <= 0 {
			return nil, ErrUnsafeD
		}
		D_prev.Set(D)

		// # collapsed for 2 coins
		// K0: uint256 = unsafe_div(unsafe_div((10**18 * N_COINS**2) * x[0], D) * x[1], D)
		K0.Div(number.Mul(number.Div(
			number.Mul(number.Mul(U_1e18, number.Mul(NumTokensU256, NumTokensU256)), &x[0]), D), &x[1]), D)

		// if _g1k0 > K0:  #       The following operations can safely be unsafe.
		// 		_g1k0 = unsafe_add(unsafe_sub(_g1k0, K0), 1)
		// else:
		// 		_g1k0 = unsafe_add(unsafe_sub(K0, _g1k0), 1)

		var _g1k0 = new(uint256.Int).Set(__g1k0)
		if _g1k0.Cmp(&K0) > 0 {
			_g1k0.AddUint64(number.Sub(_g1k0, &K0), 1)
		} else {
			_g1k0.AddUint64(number.Sub(&K0, _g1k0), 1)
		}

		// # D / (A * N**N) * _g1k0**2 / gamma**2
		// mul1: uint256 = unsafe_div(unsafe_div(unsafe_div(10**18 * D, gamma) * _g1k0, gamma) * _g1k0 * A_MULTIPLIER, ANN)
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

		// # 2*N*K0 / _g1k0
		// mul2: uint256 = unsafe_div(((2 * 10**18) * N_COINS) * K0, _g1k0)
		var mul2 = number.Div(
			number.Mul(number.SafeMul(U_2e18, NumTokensU256), &K0), _g1k0,
		)

		// # calculate neg_fprime. here K0 > 0 is being validated (safediv).
		// neg_fprime: uint256 = (S + unsafe_div(S * mul2, 10**18)) + mul1 * N_COINS / K0 - unsafe_div(mul2 * D, 10**18)
		var neg_fprime = number.Sub(
			number.Add(
				number.Add(S, number.Div(number.Mul(S, mul2), U_1e18)),
				number.SafeDiv(number.Mul(mul1, NumTokensU256), &K0),
			),
			number.Div(number.Mul(mul2, D), U_1e18),
		)

		// # D -= f / fprime; neg_fprime safediv being validated
		// D_plus: uint256 = D * (neg_fprime + S) / neg_fprime
		var D_plus = number.SafeDiv(number.SafeMul(D, number.Add(neg_fprime, S)), neg_fprime)

		// D_minus: uint256 = unsafe_div(D * D,  neg_fprime)
		var D_minus = number.Div(number.SafeMul(D, D), neg_fprime)

		if U_1e18.Cmp(&K0) > 0 {
			// D_minus += unsafe_div(unsafe_div(D * unsafe_div(mul1, neg_fprime), 10**18) * unsafe_sub(10**18, K0), K0)
			D_minus = number.SafeAdd(D_minus,
				number.Div(
					number.Mul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), U_1e18),
						number.Sub(U_1e18, &K0)),
					&K0,
				),
			)
		} else {
			// D_minus -= unsafe_div(unsafe_div(D * unsafe_div(mul1, neg_fprime), 10**18) * unsafe_sub(K0, 10**18), K0)
			D_minus = number.SafeSub(D_minus,
				number.Div(
					number.Mul(number.Div(number.SafeMul(D, number.Div(mul1, neg_fprime)), U_1e18),
						number.Sub(&K0, U_1e18)),
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
				if frac.Cmp(number.Div(MinFrac, NumTokensU256)) < 0 || frac.Cmp(number.Div(MaxFrac,
					NumTokensU256)) > 0 {
					return nil, ErrUnsafeXi
				}
			}
			return D, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveCryptoMathOptimized2.vy#L229
func get_y(
	_ann, _gamma *uint256.Int, x []uint256.Int, _D *uint256.Int, i int,
	// output
	y, K0 *uint256.Int,
) error {
	if _ann.Cmp(MinA) < 0 || _ann.Cmp(MaxA) > 0 {
		zz["4"] = true
		return ErrUnsafeA
	}

	if _gamma.Cmp(MinGamma) < 0 || _gamma.Cmp(MaxGamma) > 0 {
		return ErrUnsafeGamma
	}

	if _D.Cmp(MinD) < 0 || _D.Cmp(MaxD) > 0 {
		zz["6"] = true
		return ErrUnsafeD
	}

	// lim_mul: uint256 = 100 * 10**18  # 100.0
	// if _gamma > MAX_GAMMA_SMALL:
	//     lim_mul = unsafe_div(unsafe_mul(lim_mul, MAX_GAMMA_SMALL), _gamma)  # smaller than 100.0
	// lim_mul_signed: int256 = convert(lim_mul, int256)
	lim_mul := U_1e20
	if _gamma.Cmp(MaxGammaSmall) > 0 {
		zz["7"] = true
		lim_mul = number.Div(number.Mul(lim_mul, MaxGammaSmall), _gamma)
	}
	lim_mul_signed := i256.SafeToInt256(lim_mul)

	// ANN: int256 = convert(_ANN, int256)
	// gamma: int256 = convert(_gamma, int256)
	// D: int256 = convert(_D, int256)
	// x_j: int256 = convert(x[j], int256)
	// x_k: int256 = convert(x[k], int256)
	// gamma2: int256 = unsafe_mul(gamma, gamma)
	ann := i256.SafeToInt256(_ann)
	gamma := i256.SafeToInt256(_gamma)
	D := i256.SafeToInt256(_D)
	x_j := i256.SafeToInt256(&x[1-i])
	gamma2 := i256.Mul(gamma, gamma)

	// # savediv by x_j done here:
	// y: int256 = D**2 / (x_j * N_COINS**2)

	// # K0_i: int256 = (10**18 * N_COINS) * x_j / D
	// K0_i: int256 = unsafe_div(10**18 * N_COINS * x_j, D)
	// assert (K0_i >= unsafe_div(10**36, lim_mul_signed)) and (K0_i <= lim_mul_signed)  # dev: unsafe values x[i]
	K0_i := i256.SafeToInt256(number.Div(number.Mul(number.Mul(U_1e18, NumTokensU256), &x[1-i]), _D))
	if K0_i.Cmp(i256.Div(I_1e36, lim_mul_signed)) < 0 || K0_i.Cmp(lim_mul_signed) > 0 {
		zz["8"] = true
		return ErrUnsafeXi
	}

	// ann_gamma2: int256 = ANN * gamma2
	ann_gamma2 := i256.Mul(ann, gamma2)

	// # a = 10**36 / N_COINS**2
	a := i256.Set(I_1e32)

	// # b = ANN*D*gamma2/4/10000/x_j/10**4 - 10**32*3 - 2*gamma*10**14
	b := i256.Sub(i256.Sub(
		i256.Div(i256.Div(i256.Mul(D, ann_gamma2), I_4e8), x_j),
		I_3e32),
		i256.Mul(gamma, I_2e14),
	)

	// # c = 10**32*3 + 4*gamma*10**14 + gamma2/10**4 + 4*ANN*gamma2*x_j/D/10000/4/10**4 - 4*ANN*gamma2/10000/4/10**4
	c := i256.Sub(i256.Add(i256.Add(i256.Add(
		I_3e32,
		i256.Mul(gamma, I_4e14)),
		i256.Div(gamma2, I_1e4)),
		i256.Div(i256.Mul(i256.Div(i256.Mul(i256.Number_4, ann_gamma2), I_4e8), x_j), D)),
		i256.Div(i256.Mul(i256.Number_4, ann_gamma2), I_4e8),
	)

	// # d = -(10**18+gamma)**2 / 10**4
	// d: int256 = -unsafe_div(unsafe_add(10**18, gamma) ** 2, 10**4)
	tmp := i256.Add(I_1e18, gamma)
	d := i256.Neg(i256.Div(i256.Mul(tmp, tmp), I_1e4))

	// # delta0: int256 = 3*a*c/b - b
	delta0 := i256.Sub(i256.Div(i256.Mul(i256.Mul(i256.Number_3, a), c), b), b)

	// # delta1: int256 = 9*a*c/b - 2*b - 27*a**2/b*d/b
	delta1 := i256.Sub(i256.Sub(i256.Div(i256.Mul(i256.Mul(I_9, a), c), b), i256.Mul(i256.Number_2, b)),
		i256.Div(i256.Mul(i256.Div(i256.Mul(I_27, i256.Mul(a, a)), b), d), b))

	var divider int256.Int
	divider.SetUint64(1)
	// threshold: int256 = min(min(abs(delta0), abs(delta1)), a)
	threshold := i256.Abs(delta0)
	if threshold.Cmp(i256.Abs(delta1)) > 0 {
		threshold = i256.Abs(delta1)
	}
	if threshold.Cmp(a) > 0 {
		threshold = a
	}
	if threshold.Cmp(I_1e48) > 0 {
		zz["9"] = true
		divider.Set(I_1e30)
	} else if threshold.Cmp(I_1e46) > 0 {
		zz["a"] = true
		divider.Set(I_1e28)
	} else if threshold.Cmp(I_1e44) > 0 {
		zz["b"] = true
		divider.Set(I_1e26)
	} else if threshold.Cmp(I_1e42) > 0 {
		zz["c"] = true
		divider.Set(I_1e24)
	} else if threshold.Cmp(I_1e40) > 0 {
		zz["d"] = true
		divider.Set(I_1e22)
	} else if threshold.Cmp(I_1e38) > 0 {
		zz["e"] = true
		divider.Set(I_1e20)
	} else if threshold.Cmp(I_1e36) > 0 {
		zz["f"] = true
		divider.Set(I_1e18)
	} else if threshold.Cmp(I_1e34) > 0 {
		zz["g"] = true
		divider.Set(I_1e16)
	} else if threshold.Cmp(I_1e32) > 0 {
		zz["h"] = true
		divider.Set(I_1e14)
	} else if threshold.Cmp(I_1e30) > 0 {
		divider.Set(I_1e12)
	} else if threshold.Cmp(I_1e28) > 0 {
		zz["i"] = true
		divider.Set(I_1e10)
	} else if threshold.Cmp(I_1e26) > 0 {
		zz["j"] = true
		divider.Set(I_1e8)
	} else if threshold.Cmp(I_1e24) > 0 {
		zz["k"] = true
		divider.Set(I_1e6)
	} else if threshold.Cmp(I_1e20) > 0 {
		zz["l"] = true
		divider.Set(I_1e2)
	}
	a = i256.Div(a, &divider)
	b = i256.Div(b, &divider)
	c = i256.Div(c, &divider)
	d = i256.Div(d, &divider)

	// # delta0 = 3*a*c/b - b
	_3ac := i256.Mul(i256.Mul(i256.Number_3, a), c)
	delta0 = i256.Sub(i256.Div(_3ac, b), b)

	// # delta1 = 9*a*c/b - 2*b - 27*a**2/b*d/b
	delta1 = i256.Sub(
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

	sqrt_val := new(int256.Int)
	if sqrt_arg.Sign() > 0 {
		sqrt_val.Sqrt(sqrt_arg)
	} else {
		zz["m"] = true
		return _newton_y(_ann, _gamma, x, _D, i, lim_mul, y)
	}

	var b_cbrt *int256.Int
	if b.Sign() >= 0 {
		zz["n"] = true
		b_cbrt = i256.SafeToInt256(cbrt(i256.SafeConvertToUInt256(b)))
	} else {
		b_cbrt = i256.Neg(i256.SafeToInt256(cbrt(i256.SafeConvertToUInt256(i256.Neg(b)))))
	}

	var second_cbrt *int256.Int
	if delta1.Sign() > 0 {
		// # convert(self._cbrt(convert((delta1 + sqrt_val), uint256)/2), int256)
		second_cbrt = i256.SafeToInt256(
			cbrt(number.Div(
				i256.SafeConvertToUInt256(i256.Add(delta1, sqrt_val)),
				number.Number_2)))
	} else {
		second_cbrt = i256.Neg(i256.SafeToInt256(
			cbrt(number.Div(
				i256.SafeConvertToUInt256(i256.Sub(sqrt_val, delta1)),
				number.Number_2))),
		)
	}

	// # C1: int256 = b_cbrt**2/10**18*second_cbrt/10**18
	C1 := i256.Div(
		i256.Mul(i256.Div(i256.Mul(b_cbrt, b_cbrt), I_1e18), second_cbrt),
		I_1e18,
	)

	// # root: int256 = (10**18*C1 - 10**18*b - 10**18*b*delta0/C1)/(3*a), keep 2 safe ops here.
	root := i256.Div(
		i256.Sub(i256.Sub(
			i256.Mul(I_1e18, C1),
			i256.Mul(I_1e18, b)),
			i256.Mul(i256.Div(i256.Mul(I_1e18, b), C1), delta0)),
		i256.Mul(i256.Number_3, a),
	)

	// # y_out: uint256[2] =  [
	// #     convert(D**2/x_j*root/4/10**18, uint256),   # <--- y
	// #     convert(root, uint256)  # <----------------------- K0Prev
	// # ]
	y.Set(i256.SafeConvertToUInt256(i256.Div(i256.Mul(i256.Div(i256.Mul(D, D), x_j), root), I_4e18)))
	K0.Set(i256.SafeConvertToUInt256(root))

	frac := number.Div(number.Mul(y, U_1e18), _D)
	// assert (frac >= unsafe_div(10**36 / N_COINS, lim_mul)) and (frac <= unsafe_div(lim_mul, N_COINS))  # dev: unsafe value for y
	if frac.Cmp(number.Div(number.Div(U_1e36, NumTokensU256), lim_mul)) < 0 || frac.Cmp(number.Div(lim_mul,
		NumTokensU256)) > 0 {
		zz["o"] = true
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
		zz["p"] = true
		xx = x
	} else if x.Cmp(CbrtConst2) >= 0 {
		zz["q"] = true
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
			pow_mod256(number.Number_2, number.Div(log2x, number.Number_3)), // # <- pow
			pow_mod256(U_1260, remainder),
		),
		pow_mod256(U_1e3, remainder),
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
		zz["r"] = true
		a.Mul(a, U_1e12)
	} else if x.Cmp(CbrtConst2) >= 0 {
		zz["s"] = true
		a.Mul(a, U_1e6)
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

// Calculate x[i] given A, gamma, xp and D using newton's method with safety checks.
// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveCryptoMathOptimized2.vy#L210
func newton_y(
	ann, gamma *uint256.Int, x []uint256.Int, D *uint256.Int, i int,
	// output
	y *uint256.Int,
) error {
	// assert ANN > MIN_A - 1 and ANN < MAX_A + 1  # dev: unsafe values A
	// assert gamma > MIN_GAMMA - 1 and gamma < MAX_GAMMA + 1  # dev: unsafe values gamma
	// assert D > 10**17 - 1 and D < 10**15 * 10**18 + 1 # dev: unsafe values D
	// lim_mul: uint256 = 100 * 10**18  # 100.0
	// if gamma > MAX_GAMMA_SMALL:
	//     lim_mul = unsafe_div(unsafe_mul(lim_mul, MAX_GAMMA_SMALL), gamma)  # smaller than 100.0
	if ann.Cmp(MinA) < 0 || ann.Cmp(MaxA) > 0 {
		return ErrUnsafeA
	}

	if gamma.Cmp(MinGamma) < 0 || gamma.Cmp(MaxGamma) > 0 {
		return ErrUnsafeGamma
	}

	if D.Cmp(MinD) < 0 || D.Cmp(MaxD) > 0 {
		return ErrUnsafeD
	}

	lim_mul := U_1e20
	if gamma.Cmp(MaxGammaSmall) > 0 {
		lim_mul = number.Div(number.Mul(lim_mul, MaxGammaSmall), gamma)
	}

	// y: uint256 = self._newton_y(ANN, gamma, x, D, i, lim_mul)
	if err := _newton_y(ann, gamma, x, D, i, lim_mul, y); err != nil {
		return err
	}

	// frac: uint256 = y * 10**18 / D
	frac := number.Mul(number.Mul(y, U_1e18), U_1e18)
	frac.Div(frac, D)
	// assert (frac >= unsafe_div(10**36 / N_COINS, lim_mul)) and (frac <= unsafe_div(lim_mul, N_COINS))  # dev: unsafe value for y
	if frac.Cmp(lim_mul) < 0 || frac.Cmp(U_1e20) > 0 {
		return ErrUnsafeY
	}

	return nil
}

// Calculate x[i] given A, gamma, xp and D using newton's method.
// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveCryptoMathOptimized2.vy#L143
func _newton_y(
	ann, gamma *uint256.Int, x []uint256.Int, D *uint256.Int, i int, lim_mul *uint256.Int,
	// output
	y *uint256.Int,
) error {
	x_j := &x[1-i]
	y.Div(number.Mul(D, D), number.Mul(x_j, number.Mul(NumTokensU256, NumTokensU256)))
	K0i := number.Div(number.Mul(number.Mul(U_1e18, NumTokensU256), x_j), D)

	// assert (K0_i >= unsafe_div(10**36, lim_mul)) and (K0_i <= lim_mul)  # dev: unsafe values x[i]
	if K0i.Cmp(number.Div(U_1e36, lim_mul)) < 0 || K0i.Cmp(lim_mul) > 0 {
		return ErrUnsafeXi
	}

	// convergence_limit: uint256 = max(max(x_j / 10**14, D / 10**14), 100)
	var convergenceLimit = number.Div(x_j, U_1e14)
	var temp = number.Div(D, U_1e14)
	if temp.Cmp(convergenceLimit) > 0 {
		convergenceLimit = temp
	}
	if convergenceLimit.CmpUint64(100) < 0 {
		convergenceLimit.SetUint64(100)
	}

	var yPrev, K0, S, _g1k0, mul1, yfprime uint256.Int
	De18 := number.SafeMul(D, U_1e18)

	for j := 0; j < 255; j += 1 {
		yPrev.Set(y)
		K0.Div(number.SafeMul(number.SafeMul(K0i, y), NumTokensU256), D)
		S.Add(x_j, y)

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
			return nil
		}
	}
	return ErrYDoesNotConverge
}

func reductionCoefficient(x []uint256.Int, feeGamma *uint256.Int, K *uint256.Int) error {
	var S uint256.Int
	number.SafeAddZ(&x[0], &x[1], &S)
	if S.IsZero() {
		return ErrZero
	}

	K.Mul(U_1e18, number.Mul(NumTokensU256, NumTokensU256))
	K.Div(number.SafeMul(K, &x[0]), &S)
	K.Div(number.SafeMul(K, &x[1]), &S)

	K.Div(
		number.SafeMul(feeGamma, U_1e18),
		number.SafeSub(number.SafeAdd(feeGamma, U_1e18), K))
	return nil
}

func (t *PoolSimulator) _A_gamma() (*uint256.Int, *uint256.Int) {
	var A, gamma uint256.Int
	t._A_gamma_inplace(&A, &gamma)
	return &A, &gamma
}

// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveTwocryptoOptimized.vy
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

func wad_exp(x *int256.Int) (*uint256.Int, error) {
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

	// # If the result is `< 0.5`, we return zero. This happens when we have the following:
	// # "x <= floor(log(0.5e18) * 1e18) ~ -42e18".
	if x.Cmp(i256.MustFromDecimal("-42139678854452767551")) <= 0 {
		return uint256.NewInt(0), nil
	}

	// # When the result is "> (2 ** 255 - 1) / 1e18" we cannot represent it as a signed integer.
	// # This happens when "x >= floor(log((2 ** 255 - 1) / 1e18) * 1e18) ~ 135".
	if x.Cmp(i256.MustFromDecimal("135305999368893231589")) >= 0 {
		return nil, ErrWadExpOverflow
	}

	// # `x` is now in the range "(-42, 136) * 1e18". Convert to "(-42, 136) * 2 ** 96" for higher
	// # intermediate precision and a binary base. This base conversion is a multiplication with
	// # "1e18 / 2 ** 96 = 5 ** 18 / 2 ** 78".
	value := i256.Div(i256.Lsh(x, 78), i256.MustFromDecimal("3814697265625"))

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
	tmp := number.Mul(i256.UnsafeToUInt256(r),
		uint256.MustFromDecimal("3822833074963236453042738258902158003155416615667"))
	n := 195 - k.Int64()
	return new(uint256.Int).Rsh(tmp, uint(n)), nil
}

// https://github.com/curvefi/twocrypto-ng/blob/d21b270/contracts/main/CurveCryptoMathOptimized2.vy#L465
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
		return ErrUnsafeD
	}

	// # K0 = P * N**N / D**N.
	// # K0 is dimensionless and has 10**36 precision:
	K0 := number.Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(number.Number_4, number.SafeMul(&_xp[0], &_xp[1])),
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
	// # p is in 10**18 precision.
	out[0].Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(
					&_xp[0],
					number.SafeAdd(GK0,
						number.Div(number.SafeMul(number.Div(number.SafeMul(NNAG2, &_xp[1]), _D), K0), U_1e36)),
				),
				&_xp[1],
			),
			U_1e18,
		),
		denominator,
	)

	return nil
}

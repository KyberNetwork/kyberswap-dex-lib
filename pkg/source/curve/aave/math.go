package aave

import (

	// "errors"
	"math/big"
	"time"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var FeeDenominator = utils.NewBig10("10000000000")

const MaxLoopLimit = 256

var APrecision = big.NewInt(100)

/**
 * @notice Given a set of balances and precision multipliers, return the
 * precision-adjusted balances.
 *
 * @param balances an array of token balances, in their native precisions.
 * These should generally correspond with pooled tokens.
 *
 * @param precisionMultipliers an array of multipliers, corresponding to
 * the amounts in the balances array. When multiplied together they
 * should yield amounts at the pool's precision.
 *
 * @return an array of amounts "scaled" to the pool's precision
 */
func _xp(
	balances []*big.Int,
	precisionMultipliers []*big.Int,
) ([]*big.Int, error) {
	xp := make([]*big.Int, 0)
	var numTokens = len(balances)
	if numTokens != len(precisionMultipliers) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	for i := 0; i < numTokens; i += 1 {
		xp = append(xp, new(big.Int).Mul(balances[i], precisionMultipliers[i]))
	}
	return xp, nil
}

/**
 * @notice Calculates and returns A based on the ramp settings
 * @dev See the StableSwap paper for details
 * @param self Swap struct to read from
 * @return A parameter in its raw precision form
 */
func _getAPrecise(
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
) *big.Int {
	var t1 = futureATime
	var a1 = futureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = initialATime
		var a0 = initialA
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

/**
 * @notice Get D, the StableSwap invariant, based on a set of balances and a particular A.
 * @param xp a precision-adjusted set of pool balances. Array should be the same cardinality
 * as the pool.
 * @param a the amplification coefficient * n * (n - 1) in A_PRECISION.
 * See the StableSwap paper for details
 * @return the invariant, at the precision of the pool
 */
func getD(xp []*big.Int, a *big.Int) (*big.Int, error) {
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
					new(big.Int).Div(new(big.Int).Mul(nA, s), APrecision),
					new(big.Int).Mul(dP, numTokensBI),
				),
				d,
			),
			new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(nA, APrecision), d), APrecision),
				new(big.Int).Mul(dP, big.NewInt(int64(numTokens+1))),
			),
		)
		if new(big.Int).Sub(d, prevD).CmpAbs(big.NewInt(1)) <= 0 {
			return d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

/**
 * @notice Calculate the new balances of the tokens given the indexes of the token
 * that is swapped from (FROM) and the token that is swapped to (TO).
 * This function is used as a helper function to calculate how much TO token
 * the user should receive on swap.
 *
 * @param self Swap struct to read from
 * @param tokenIndexFrom index of FROM token
 * @param tokenIndexTo index of TO token
 * @param x the new total amount of FROM token
 * @param xp balances of the tokens in the pool
 * @return the amount of TO token that should remain in the pool
 */
func getY(
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	x *big.Int,
	xp []*big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var a = _getAPrecise(futureATime, futureA, initialATime, initialA)
	var d, err = getD(xp, a)
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
		if _x.Cmp(constant.ZeroBI) == 0 {
			return nil, ErrZero
		}
		s = new(big.Int).Add(s, _x)
		c = new(big.Int).Div(
			new(big.Int).Mul(c, d),
			new(big.Int).Mul(_x, numTokensBI),
		)
	}
	if nA.Cmp(constant.ZeroBI) == 0 {
		return nil, ErrZero
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).Mul(c, d), APrecision),
		new(big.Int).Mul(nA, numTokensBI),
	)
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(new(big.Int).Mul(d, APrecision), nA),
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
	return nil, ErrAmountOutNotConverge
}

/**
 * @notice Internally calculates a swap between two tokens.
 *
 * @dev The caller is expected to transfer the actual amounts (dx and dy)
 * using the token contracts.
 *
 * @param self Swap struct to read from
 * @param tokenIndexFrom the token to sell
 * @param tokenIndexTo the token to buy
 * @param dx the number of tokens to sell. If the token charges a fee on transfers,
 * use the amount that gets transferred after the fee.
 * @return dy the number of tokens the user will get
 * @return dyFee the associated fee
 */
func _calculateSwap(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	xp, err := _xp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(new(big.Int).Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom]), xp[tokenIndexFrom])
	y, err := getY(futureATime, futureA, initialATime, initialA, tokenIndexFrom, tokenIndexTo, x, xp)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[tokenIndexTo], y), constant.One)
	var dyFee = new(big.Int).Div(new(big.Int).Mul(dy, swapFee), FeeDenominator)
	dy = new(big.Int).Div(new(big.Int).Sub(dy, dyFee), tokenPrecisionMultipliers[tokenIndexTo])
	return dy, dyFee, nil
}

// CalculateSwap /**
func CalculateSwap(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	var dy, fee, err = _calculateSwap(
		balances,
		tokenPrecisionMultipliers,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		tokenIndexFrom,
		tokenIndexTo,
		dx,
	)
	return dy, fee, err
}

/**
 * @notice Calculate the price of a token in the pool with given
 * precision-adjusted balances and a particular D.
 *
 * @dev This is accomplished via solving the invariant iteratively.
 * See the StableSwap paper and Curve.fi implementation for further details.
 *
 * x_1**2 + x1 * (sum' - (A*n**n - 1) * D / (A * n**n)) = D ** (n + 1) / (n ** (2 * n) * prod' * A)
 * x_1**2 + b*x_1 = c
 * x_1 = (x_1**2 + c) / (2*x_1 + b)
 *
 * @param a the amplification coefficient * n * (n - 1). See the StableSwap paper for details.
 * @param tokenIndex Index of token we are calculating for.
 * @param xp a precision-adjusted set of pool balances. Array should be
 * the same cardinality as the pool.
 * @param d the stableswap invariant
 * @return the price of the token, in the same precision as in xp
 */
func getYD(
	a *big.Int,
	tokenIndex int,
	xp []*big.Int,
	d *big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, ErrTokenNotFound
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
	if nA.Cmp(constant.ZeroBI) == 0 {
		return nil, ErrZero
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(c, d),
			APrecision,
		),
		new(big.Int).Mul(nA, numTokensBI),
	)
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(
			new(big.Int).Mul(d, APrecision),
			nA,
		),
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
	return nil, ErrAmountOutNotConverge
}

/**
 * @notice internal helper function to calculate fee per token multiplier used in
 * swap fee calculations
 */
func _feePerToken(
	swapFee *big.Int,
	numTokens int,
) *big.Int {
	var numTokensBI = big.NewInt(int64(numTokens))
	return new(big.Int).Div(
		new(big.Int).Mul(
			swapFee,
			numTokensBI,
		),
		new(big.Int).Mul(
			new(big.Int).Sub(numTokensBI, constant.One),
			constant.Four,
		),
	)
}

/**
 * @notice Calculate the dy of withdrawing in one token
 * @param tokenIndex which token will be withdrawn
 * @param tokenAmount the amount to withdraw in the pools precision
 * @return the d and the new y after withdrawing one token
 */
func calculateWithdrawOneTokenDy(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	var numTokens = len(balances)
	if tokenIndex >= numTokens {
		return nil, nil, ErrTokenIndexesOutOfRange
	}
	xp, err := _xp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var preciseA = _getAPrecise(futureATime, futureA, initialATime, initialA)
	d0, err := getD(xp, preciseA)
	if err != nil {
		return nil, nil, err
	}
	var d1 = new(big.Int).Sub(
		d0,
		new(big.Int).Div(
			new(big.Int).Mul(tokenAmount, d0),
			lpSupply,
		),
	)

	if tokenAmount.Cmp(xp[tokenIndex]) > 0 {
		return nil, nil, ErrWithdrawMoreThanAvailable
	}

	newY, err := getYD(preciseA, tokenIndex, xp, d1)
	if err != nil {
		return nil, nil, err
	}
	var feePerToken = _feePerToken(swapFee, numTokens)
	var xpReduced = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		var norm *big.Int
		if i == tokenIndex {
			norm = new(big.Int).Sub(new(big.Int).Div(new(big.Int).Mul(xp[i], d1), d0), newY)
		} else {
			norm = new(big.Int).Sub(xp[i], new(big.Int).Div(new(big.Int).Mul(xp[i], d1), d0))
		}
		xpReduced[i] = new(big.Int).Sub(
			xp[i],
			new(big.Int).Div(
				new(big.Int).Mul(
					norm,
					feePerToken,
				),
				FeeDenominator,
			),
		)
	}
	yd, err := getYD(preciseA, tokenIndex, xpReduced, d1)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(xpReduced[tokenIndex], yd)
	dy = new(big.Int).Div(
		new(big.Int).Sub(dy, constant.One),
		tokenPrecisionMultipliers[tokenIndex],
	)
	return dy, newY, nil
}

/**
 * @notice Calculate the dy, the amount of selected token that user receives and
 * the fee of withdrawing in one token
 * @param account the address that is withdrawing
 * @param tokenAmount the amount to withdraw in the pool's precision
 * @param tokenIndex which token will be withdrawn
 * @return the amount of token user will receive and the associated swap fee
 */
func calculateWithdrawOneToken(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	var dy, newY, err = calculateWithdrawOneTokenDy(
		balances,
		tokenPrecisionMultipliers,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		lpSupply,
		tokenIndex,
		tokenAmount,
	)
	if err != nil {
		return nil, nil, err
	}
	xp, err := _xp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var dySwapFee = new(big.Int).Sub(
		new(big.Int).Div(new(big.Int).Sub(xp[tokenIndex], newY), tokenPrecisionMultipliers[tokenIndex]), dy)
	dy = new(big.Int).Div(
		new(big.Int).Mul(dy, new(big.Int).Sub(FeeDenominator, withdrawFee)),
		FeeDenominator,
	)
	return dy, dySwapFee, nil
}

// CalculateRemoveLiquidityOneToken /**
func CalculateRemoveLiquidityOneToken(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	amount, fee, err := calculateWithdrawOneToken(
		balances,
		tokenPrecisionMultipliers,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		withdrawFee,
		lpSupply,
		tokenIndex,
		tokenAmount)
	return amount, fee, err
}

/**
 * @notice A simple method to calculate prices from deposits or
 * withdrawals, excluding fees but including slippage. This is
 * helpful as an input into the various "min" parameters on calls
 * to fight front-running
 *
 * @param amounts an array of token amounts to deposit or withdrawal,
 * corresponding to pooledTokens. The amount should be in each
 * pooled token's native precision. If a token charges a fee on transfers,
 * use the amount that gets transferred after the fee.
 * @param deposit whether this is a deposit or a withdrawal
 * @return if deposit was true, total amount of lp token that will be minted and if
 * deposit was false, total amount of lp token that will be burned
 */
func calculateTokenAmount(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	amounts []*big.Int,
	deposit bool,
) (*big.Int, error) {
	var numTokens = len(balances)
	var a = _getAPrecise(futureATime, futureA, initialATime, initialA)
	xp, err := _xp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, err
	}
	d0, err := getD(xp, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		if deposit {
			balances1[i] = new(big.Int).Add(balances[i], amounts[i])
		} else {
			if balances[i].Cmp(amounts[i]) < 0 {
				return nil, ErrWithdrawMoreThanAvailable
			}
			balances1[i] = new(big.Int).Sub(balances[i], amounts[i])
		}
	}
	xp1, err := _xp(balances1, tokenPrecisionMultipliers)
	if err != nil {
		return nil, err
	}
	d1, err := getD(xp1, a)
	if err != nil {
		return nil, err
	}
	var totalSupply = new(big.Int).Set(lpSupply)
	if deposit {
		return new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Sub(d1, d0),
				totalSupply,
			),
			d0,
		), nil
	} else {
		return new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(d0, d1), totalSupply), d0), FeeDenominator),
			new(big.Int).Sub(FeeDenominator, withdrawFee),
		), nil
	}
}

func CalculateAddLiquidityOneToken(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	var numTokens = len(balances)
	var amounts = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		amounts[i] = big.NewInt(0)
	}
	amounts[tokenIndex] = new(big.Int).Set(tokenAmount)
	amount, err := calculateTokenAmount(
		balances,
		tokenPrecisionMultipliers,
		futureATime,
		futureA,
		initialATime,
		initialA,
		withdrawFee,
		lpSupply,
		amounts,
		true)
	return amount, constant.ZeroBI, err
}

func _dynamicFee(
	xpi *big.Int,
	xpj *big.Int,
	_fee *big.Int,
	_feemul *big.Int,
) *big.Int {
	if _feemul.Cmp(FeeDenominator) <= 0 {
		return _fee
	} else {
		var xps2 = new(big.Int).Add(xpi, xpj)
		xps2 = new(big.Int).Mul(xps2, xps2)
		return new(big.Int).Div(
			new(big.Int).Mul(_feemul, _fee),
			new(big.Int).Add(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Mul(
							new(big.Int).Mul(
								new(big.Int).Sub(_feemul, FeeDenominator),
								constant.Four),
							xpi),
						xpj),
					xps2,
				),
				FeeDenominator,
			),
		)
	}
}

func GetDyUnderlying(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	offPegFeeMultiplier *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	xp, err := _xp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(xp[tokenIndexFrom], new(big.Int).Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom]))
	y, err := getY(
		futureATime,
		futureA,
		initialATime,
		initialA,
		tokenIndexFrom,
		tokenIndexTo,
		x,
		xp,
	)
	if err != nil {
		return nil, nil, err
	}
	dy := new(big.Int).Div(
		new(big.Int).Sub(xp[tokenIndexTo], y),
		tokenPrecisionMultipliers[tokenIndexTo],
	)
	var dynamicFee = _dynamicFee(
		new(big.Int).Div(new(big.Int).Add(xp[tokenIndexFrom], x), constant.Two),
		new(big.Int).Div(new(big.Int).Add(xp[tokenIndexTo], y), constant.Two),
		swapFee,
		offPegFeeMultiplier,
	)
	var _fee = new(big.Int).Div(new(big.Int).Mul(dynamicFee, dy), FeeDenominator)
	return new(big.Int).Sub(dy, _fee), _fee, nil
}

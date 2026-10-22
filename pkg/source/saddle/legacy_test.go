package saddle

// The big.Int implementation this package had before the uint256 port (math.go and pool_simulator.go
// at origin/main 12be3057e, identifiers prefixed legacy). Tests compare the port with it; it is not
// used outside tests.

import (
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

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
func legacyXp(
	balances []*big.Int,
	precisionMultipliers []*big.Int,
) ([]*big.Int, error) {
	xp := make([]*big.Int, 0)
	var numTokens = len(balances)
	if numTokens != len(precisionMultipliers) {
		return nil, errors.New("balances must match multipliers")
	}
	for i := range numTokens {
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
func legacyGetAPrecise(
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
func legacyGetD(xp []*big.Int, a *big.Int) (*big.Int, error) {
	var numTokens = len(xp)
	var s = new(big.Int).SetInt64(0)
	for i := range numTokens {
		s = new(big.Int).Add(s, xp[i])
	}
	if s.Sign() == 0 {
		return s, nil
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var prevD *big.Int
	var d = new(big.Int).Set(s)
	var nA = new(big.Int).Mul(a, numTokensBI)
	for range MaxLoopLimit {
		var dP = new(big.Int).Set(d)
		for j := range numTokens {
			dP = new(big.Int).Div(
				new(big.Int).Mul(dP, d),
				new(big.Int).Mul(xp[j], numTokensBI),
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
	return nil, errors.New("d does not converge")
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
func legacyGetY(
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
		return nil, errors.New("can't compare token to itself")
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, errors.New("tokens must be in pool")
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var a = legacyGetAPrecise(futureATime, futureA, initialATime, initialA)
	var d, err = legacyGetD(xp, a)
	if err != nil {
		return nil, err
	}
	var c = new(big.Int).Set(d)
	var s = big.NewInt(0)
	var nA = new(big.Int).Mul(a, numTokensBI)
	var _x *big.Int
	for i := range numTokens {
		if i == tokenIndexFrom {
			_x = new(big.Int).Set(x)
		} else if i != tokenIndexTo {
			_x = new(big.Int).Set(xp[i])
		} else {
			continue
		}
		if _x.Cmp(bignumber.ZeroBI) == 0 {
			return nil, errors.New("zero")
		}
		s = new(big.Int).Add(s, _x)
		c = new(big.Int).Div(
			new(big.Int).Mul(c, d),
			new(big.Int).Mul(_x, numTokensBI),
		)
	}
	if nA.Cmp(bignumber.ZeroBI) == 0 {
		return nil, errors.New("zero")
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
	for range MaxLoopLimit {
		yPrev = new(big.Int).Set(y)
		y = new(big.Int).Div(
			new(big.Int).Add(new(big.Int).Mul(y, y), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(y, big.NewInt(2)), b), d),
		)
		if new(big.Int).Sub(y, yPrev).CmpAbs(bignumber.One) <= 0 {
			return y, nil
		}
	}
	return nil, errors.New("approximation did not converge")
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
func legacyCalculateSwapInner(
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
	xp, err := legacyXp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(new(big.Int).Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom]), xp[tokenIndexFrom])
	y, err := legacyGetY(futureATime, futureA, initialATime, initialA, tokenIndexFrom, tokenIndexTo, x, xp)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(new(big.Int).Sub(xp[tokenIndexTo], y), bignumber.One)
	var dyFee = new(big.Int).Div(new(big.Int).Mul(dy, swapFee), FeeDenominator)
	dy = new(big.Int).Div(new(big.Int).Sub(dy, dyFee), tokenPrecisionMultipliers[tokenIndexTo])
	return dy, dyFee, nil
}

// legacyCalculateSwap /**
func legacyCalculateSwap(
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
	var dy, fee, err = legacyCalculateSwapInner(
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
func legacyGetYD(
	a *big.Int,
	tokenIndex int,
	xp []*big.Int,
	d *big.Int,
) (*big.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, errors.New("token not found")
	}
	var numTokensBI = big.NewInt(int64(numTokens))
	var c = new(big.Int).Set(d)
	var s = big.NewInt(0)
	var nA = new(big.Int).Mul(a, numTokensBI)
	for i := range numTokens {
		if i != tokenIndex {
			s = new(big.Int).Add(s, xp[i])
			c = new(big.Int).Div(
				new(big.Int).Mul(c, d),
				new(big.Int).Mul(xp[i], numTokensBI),
			)
		}
	}
	c = new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(c, d),
			APrecision,
		),
		new(big.Int).Mul(nA, numTokensBI),
	)
	if nA.Cmp(bignumber.ZeroBI) == 0 {
		return nil, errors.New("zero")
	}
	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(
			new(big.Int).Mul(d, APrecision),
			nA,
		),
	)
	var yPrev *big.Int
	var y = new(big.Int).Set(d)
	for range MaxLoopLimit {
		yPrev = new(big.Int).Set(y)
		y = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Mul(y, y),
				c,
			),
			new(big.Int).Sub(
				new(big.Int).Add(
					new(big.Int).Mul(y, bignumber.Two),
					b,
				),
				d,
			),
		)
		if new(big.Int).Sub(y, yPrev).CmpAbs(bignumber.One) <= 0 {
			return y, nil
		}
	}
	return nil, errors.New("approximation did not converge")
}

/**
 * @notice internal helper function to calculate fee per token multiplier used in
 * swap fee calculations
 */
func legacyFeePerToken(
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
			new(big.Int).Sub(numTokensBI, bignumber.One),
			bignumber.Four,
		),
	)
}

/**
 * @notice Calculate the dy of withdrawing in one token
 * @param tokenIndex which token will be withdrawn
 * @param tokenAmount the amount to withdraw in the pools precision
 * @return the d and the new y after withdrawing one token
 */
func legacyCalculateWithdrawOneTokenDy(
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
		return nil, nil, errors.New("token index out of range")
	}
	xp, err := legacyXp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var preciseA = legacyGetAPrecise(futureATime, futureA, initialATime, initialA)
	d0, err := legacyGetD(xp, preciseA)
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
		return nil, nil, errors.New("withdraw exceeds available")
	}

	newY, err := legacyGetYD(preciseA, tokenIndex, xp, d1)
	if err != nil {
		return nil, nil, err
	}
	var feePerToken = legacyFeePerToken(swapFee, numTokens)
	var xpReduced = make([]*big.Int, numTokens)
	for i := range numTokens {
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
	yd, err := legacyGetYD(preciseA, tokenIndex, xpReduced, d1)
	if err != nil {
		return nil, nil, err
	}
	var dy = new(big.Int).Sub(xpReduced[tokenIndex], yd)
	dy = new(big.Int).Div(
		new(big.Int).Sub(dy, bignumber.One),
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
func legacyCalculateWithdrawOneToken(
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
	var dy, newY, err = legacyCalculateWithdrawOneTokenDy(
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
	xp, err := legacyXp(balances, tokenPrecisionMultipliers)
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

// legacyCalculateRemoveLiquidityOneToken /**
func legacyCalculateRemoveLiquidityOneToken(
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
	amount, fee, err := legacyCalculateWithdrawOneToken(
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
func legacyCalculateTokenAmount(
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
	var a = legacyGetAPrecise(futureATime, futureA, initialATime, initialA)
	xp, err := legacyXp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, err
	}
	d0, err := legacyGetD(xp, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]*big.Int, numTokens)
	for i := range numTokens {
		if deposit {
			balances1[i] = new(big.Int).Add(balances[i], amounts[i])
		} else {
			if balances[i].Cmp(amounts[i]) < 0 {
				return nil, errors.New("cannot withdraw more than available")
			}
			balances1[i] = new(big.Int).Sub(balances[i], amounts[i])
		}
	}
	xp1, err := legacyXp(balances1, tokenPrecisionMultipliers)
	if err != nil {
		return nil, err
	}
	d1, err := legacyGetD(xp1, a)
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

func legacyCalculateAddLiquidityOneToken(
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
	for i := range numTokens {
		amounts[i] = big.NewInt(0)
	}
	amounts[tokenIndex] = new(big.Int).Set(tokenAmount)
	amount, err := legacyCalculateTokenAmount(
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
	return amount, bignumber.ZeroBI, err
}

func legacyDynamicFee(
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
								bignumber.Four),
							xpi),
						xpj),
					xps2,
				),
				FeeDenominator,
			),
		)
	}
}

func legacyGetDyUnderlying(
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
	xp, err := legacyXp(balances, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(xp[tokenIndexFrom], new(big.Int).Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom]))
	y, err := legacyGetY(
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
	var dynamicFee = legacyDynamicFee(
		new(big.Int).Div(new(big.Int).Add(xp[tokenIndexFrom], x), bignumber.Two),
		new(big.Int).Div(new(big.Int).Add(xp[tokenIndexTo], y), bignumber.Two),
		swapFee,
		offPegFeeMultiplier,
	)
	var _fee = new(big.Int).Div(new(big.Int).Mul(dynamicFee, dy), FeeDenominator)
	return new(big.Int).Sub(dy, _fee), _fee, nil
}

type legacyPoolSimulator struct {
	pool.Pool
	Multipliers []*big.Int
	// extra fields
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       int64
	FutureATime        int64
	AdminFee           *big.Int
	DefaultWithdrawFee *big.Int
	LpToken            string
	LpSupply           *big.Int
	Paused             bool
	gas                Gas
}

func newLegacyPoolSimulator(entityPool entity.Pool) (*legacyPoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		multipliers[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	swapFee := bignumber.NewBig10(extra.SwapFee)

	// only have withdrawFee in saddle v1, default to 0
	defaultWithdrawFee := bignumber.NewBig10(extra.DefaultWithdrawFee)
	if defaultWithdrawFee == nil {
		defaultWithdrawFee = bignumber.ZeroBI
	}

	return &legacyPoolSimulator{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			SwapFee:     swapFee,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
		Multipliers:        multipliers,
		InitialA:           bignumber.NewBig10(extra.InitialA),
		FutureA:            bignumber.NewBig10(extra.FutureA),
		InitialATime:       extra.InitialATime,
		FutureATime:        extra.FutureATime,
		AdminFee:           bignumber.NewBig10(extra.AdminFee),
		DefaultWithdrawFee: defaultWithdrawFee,
		LpToken:            staticExtra.LpToken,
		LpSupply:           bignumber.NewBig10(entityPool.Reserves[numTokens]),
		Paused:             extra.Paused,
		gas:                DefaultGas,
	}, nil
}

func (t *legacyPoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if t.Paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var balances = t.Info.Reserves
	var tokenPrecisionMultipliers = t.Multipliers
	if tokenAmountIn.Token == t.LpToken {
		// withdraw
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexTo >= 0 {
			amountOut, fee, err := legacyCalculateRemoveLiquidityOneToken(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.Info.SwapFee,
				t.DefaultWithdrawFee,
				t.LpSupply,
				tokenIndexTo,
				tokenAmountIn.Amount,
			)
			if err != nil {
				return nil, err
			}
			if amountOut.Cmp(bignumber.ZeroBI) > 0 {
				return &pool.CalcAmountOutResult{
					TokenAmountOut: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: amountOut,
					},
					Fee: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: fee,
					},
					Gas: t.gas.RemoveLiquidity,
				}, nil
			}
		}
	} else if tokenOut == t.LpToken {
		// deposit
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		if tokenIndexFrom >= 0 {
			amountOut, fee, err := legacyCalculateAddLiquidityOneToken(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.DefaultWithdrawFee,
				t.LpSupply,
				tokenIndexFrom,
				tokenAmountIn.Amount,
			)
			if err != nil {
				return nil, err
			}
			if amountOut.Cmp(bignumber.ZeroBI) > 0 {
				return &pool.CalcAmountOutResult{
					TokenAmountOut: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: amountOut,
					},
					Fee: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: fee,
					},
					Gas: t.gas.AddLiquidity,
				}, nil
			}
		}
	} else {
		// swap from token to token
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
			amountOut, fee, err := legacyCalculateSwap(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.Info.SwapFee,
				tokenIndexFrom,
				tokenIndexTo,
				tokenAmountIn.Amount,
			)
			if err != nil {
				return nil, err
			}
			if amountOut.Cmp(bignumber.ZeroBI) > 0 {
				return &pool.CalcAmountOutResult{
					TokenAmountOut: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: amountOut,
					},
					Fee: &pool.TokenAmount{
						Token:  tokenOut,
						Amount: fee,
					},
					Gas: t.gas.Swap,
				}, nil
			}
		}
	}
	return nil, errors.New("i'm dead here")
}

func (t *legacyPoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.LpSupply = new(big.Int).Set(t.LpSupply)
	cloned.Info.Reserves = lo.Map(t.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})
	return &cloned
}

func (t *legacyPoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	if input.Token == t.LpToken {
		// withdraw
		outputAmount = new(big.Int).Add(
			outputAmount,
			new(big.Int).Div(
				new(big.Int).Mul(fee.Amount, t.AdminFee),
				FeeDenominator,
			),
		)
		t.LpSupply = new(big.Int).Sub(t.LpSupply, inputAmount)
	} else if output.Token == t.LpToken {
		// deposit
		t.LpSupply = new(big.Int).Add(t.LpSupply, outputAmount)
	} else {
		// swap fee
		// output = output + output * swapFee * adminFee
		outputAmount = new(big.Int).Add(
			outputAmount,
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
					t.AdminFee,
				),
				FeeDenominator,
			),
		)
	}
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}

func (t *legacyPoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *legacyPoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 && address != t.LpToken {
		return nil
	}
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	if address != t.LpToken {
		ret = append(ret, t.LpToken)
	}
	return ret
}

func (t *legacyPoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		PoolLength:    len(t.Info.Tokens),
		LpToken:       t.LpToken,
		BlockNumber:   t.Info.BlockNumber,
	}
}

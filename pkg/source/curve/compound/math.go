package compound

import (
	"math/big"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func _xp(
	balances []*big.Int,
	rates []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
) ([]*big.Int, error) {
	var xp []*big.Int
	var numTokens = len(balances)
	if numTokens != len(rates) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	for i := 0; i < numTokens; i += 1 {
		xp = append(xp, new(big.Int).Div(new(big.Int).Mul(balances[i], new(big.Int).Mul(rates[i], tokenPrecisionMultipliers[i])), constant.BONE))
	}

	return xp, nil
}

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
					new(big.Int).Mul(nA, s),
					new(big.Int).Mul(dP, numTokensBI),
				),
				d,
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).Sub(nA, new(big.Int).SetInt64(1)), d),
				new(big.Int).Mul(dP, big.NewInt(int64(numTokens+1))),
			),
		)
		if new(big.Int).Sub(d, prevD).CmpAbs(big.NewInt(1)) <= 0 {
			return d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

func getY(
	APrecise *big.Int,
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
	var a = APrecise
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
		new(big.Int).Mul(c, d),
		new(big.Int).Mul(nA, numTokensBI),
	)

	var b = new(big.Int).Add(
		s,
		new(big.Int).Div(d, nA),
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

func GetDyUnderlying(
	balances []*big.Int,
	rates []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	APrecise *big.Int,
	swapFee *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {

	xp, err := _xp(balances, rates, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x = new(big.Int).Add(xp[tokenIndexFrom], new(big.Int).Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom]))
	y, err := getY(
		APrecise,
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
	var _fee = new(big.Int).Div(new(big.Int).Mul(swapFee, dy), FeeDenominator)
	return new(big.Int).Sub(dy, _fee), _fee, nil
}

func GetDxUnderlying(
	balances []*big.Int,
	rates []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	APrecise *big.Int,
	swapFee *big.Int,
	i int,
	j int,
	dy *big.Int,
) (*big.Int, *big.Int, error) {
	xp, err := _xp(balances, rates, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}

	fee := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Mul(dy, FeeDenominator),
			new(big.Int).Sub(FeeDenominator, swapFee),
		),
		dy,
	)

	y := new(big.Int).Sub(
		xp[j],
		new(big.Int).Mul(
			new(big.Int).Div(
				new(big.Int).Mul(dy, FeeDenominator),
				new(big.Int).Sub(FeeDenominator, swapFee),
			),
			tokenPrecisionMultipliers[j],
		),
	)

	x, err := getY(APrecise, j, i, y, xp)
	if err != nil {
		return nil, nil, err
	}

	dx := new(big.Int).Div(
		new(big.Int).Sub(x, xp[i]),
		tokenPrecisionMultipliers[i],
	)

	return dx, fee, nil
}

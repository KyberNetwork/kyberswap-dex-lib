package balancerstable

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var One *big.Int
var AmpPrecision = bignumber.NewBig10("1000")

func init() {
	One = new(big.Int).Set(bignumber.BONE)
}

func _upscale(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return mulDown(amount, scalingFactor)
}

func _computeScalingFactor(tokenDecimals uint) *big.Int {
	var decimalsDiff = 36 - tokenDecimals
	return bignumber.TenPowInt(uint8(decimalsDiff))
}

func mulDown(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	return new(big.Int).Div(ret, One)
}

func mulUp(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	if ret.Cmp(bignumber.ZeroBI) == 0 {
		return ret
	}
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(ret, bignumber.One), One), bignumber.One)
}

func div(a *big.Int, b *big.Int, roundUp bool) *big.Int {
	if roundUp {
		return _divUp(a, b)
	}
	return _divDown(a, b)
}

func _divDown(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Div(a, b)
}

func _divUp(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(a, bignumber.One), b), bignumber.One)
}

func divDown(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}
	var ret = new(big.Int).Mul(a, One)
	return new(big.Int).Div(ret, b)
}

//func divUp(a *big.Int, b *big.Int) *big.Int {
//	if a.Cmp(bignumber.ZeroBI) == 0 {
//		return bignumber.ZeroBI
//	}
//	var ret = new(big.Int).Mul(a, One)
//	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(ret, bignumber.One), b), bignumber.One)
//}

func _downscaleDown(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return divDown(amount, scalingFactor)
}

func _calcOutGivenIn(
	a *big.Int,
	balances []*big.Int,
	tokenIndexIn int,
	tokenIndexOut int,
	tokenAmountIn *big.Int,
	invariant *big.Int,
) *big.Int {
	balances[tokenIndexIn] = new(big.Int).Add(balances[tokenIndexIn], tokenAmountIn)
	var finalBalanceOut = _getTokenBalanceGivenInvariantAndAllOtherBalances(a, balances, invariant, tokenIndexOut)
	if finalBalanceOut == nil {
		return nil
	}
	balances[tokenIndexIn] = new(big.Int).Sub(balances[tokenIndexIn], bignumber.One)
	return new(big.Int).Sub(new(big.Int).Sub(balances[tokenIndexOut], finalBalanceOut), bignumber.One)
}

func _getTokenBalanceGivenInvariantAndAllOtherBalances(
	a *big.Int,
	balances []*big.Int,
	invariant *big.Int,
	tokenIndex int,
) *big.Int {
	var nTokens = len(balances)
	var nTokensBi = big.NewInt(int64(nTokens))
	var ampTotal = new(big.Int).Mul(a, nTokensBi)
	var sum = balances[0]
	var P_D = new(big.Int).Mul(balances[0], nTokensBi)
	for j := 1; j < nTokens; j += 1 {
		P_D = _divDown(new(big.Int).Mul(new(big.Int).Mul(P_D, balances[j]), nTokensBi), invariant)
		sum = new(big.Int).Add(sum, balances[j])
	}
	sum = new(big.Int).Sub(sum, balances[tokenIndex])
	var inv2 = new(big.Int).Mul(invariant, invariant)
	var c = new(big.Int).Mul(
		new(big.Int).Mul(_divUp(inv2, new(big.Int).Mul(ampTotal, P_D)), AmpPrecision),
		balances[tokenIndex],
	)
	var b = new(big.Int).Add(sum, new(big.Int).Mul(_divDown(invariant, ampTotal), AmpPrecision))
	var prevTokenBalance *big.Int
	var tokenBalance = _divUp(new(big.Int).Add(inv2, c), new(big.Int).Add(invariant, b))
	for i := 0; i < 255; i += 1 {
		prevTokenBalance = tokenBalance
		tokenBalance = _divUp(
			new(big.Int).Add(new(big.Int).Mul(tokenBalance, tokenBalance), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(tokenBalance, bignumber.Two), b), invariant),
		)
		if tokenBalance.Cmp(prevTokenBalance) > 0 {
			if new(big.Int).Sub(tokenBalance, prevTokenBalance).Cmp(bignumber.One) <= 0 {
				return tokenBalance
			}
		} else if new(big.Int).Sub(prevTokenBalance, tokenBalance).Cmp(bignumber.One) <= 0 {
			return tokenBalance
		}
	}
	return nil
}

func _calculateInvariant(A *big.Int, balances []*big.Int, roundUp bool) *big.Int {
	var sum = bignumber.ZeroBI
	var numTokens = len(balances)
	var numTokensBi = big.NewInt(int64(numTokens))
	for i := 0; i < numTokens; i += 1 {
		sum = new(big.Int).Add(sum, balances[i])
	}
	if sum.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}
	var prevInvariant *big.Int
	var invariant = sum
	var ampTotal = new(big.Int).Mul(A, numTokensBi)
	for i := 0; i < 255; i += 1 {
		var P_D = new(big.Int).Mul(balances[0], numTokensBi)
		for j := 1; j < numTokens; j += 1 {
			P_D = div(new(big.Int).Mul(new(big.Int).Mul(P_D, balances[j]), numTokensBi), invariant, roundUp)
		}
		prevInvariant = invariant
		invariant = div(
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).Mul(numTokensBi, invariant), invariant),
				div(new(big.Int).Mul(new(big.Int).Mul(ampTotal, sum), P_D), AmpPrecision, roundUp),
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).Add(numTokensBi, bignumber.One), invariant),
				div(new(big.Int).Mul(new(big.Int).Sub(ampTotal, AmpPrecision), P_D), AmpPrecision, !roundUp),
			),
			roundUp,
		)
		if invariant.Cmp(prevInvariant) > 0 {
			if new(big.Int).Sub(invariant, prevInvariant).Cmp(bignumber.One) <= 0 {
				return invariant
			}
		} else if new(big.Int).Sub(prevInvariant, invariant).Cmp(bignumber.One) <= 0 {
			return invariant
		}
	}
	return nil
}

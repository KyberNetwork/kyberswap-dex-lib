package balancercomposablestable

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var One *big.Int
var AmpPrecision = utils.NewBig10("1000")

func init() {
	One = new(big.Int).Set(constant.BONE)
}

func Upscale(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return MulDownFixed(amount, scalingFactor)
}

func ComputeScalingFactor(tokenDecimals uint) *big.Int {
	var decimalsDiff = 36 - tokenDecimals
	return constant.TenPowInt(uint8(decimalsDiff))
}

func MulDownFixed(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	return new(big.Int).Div(ret, One)
}

func MulUpFixed(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	if ret.Cmp(constant.Zero) == 0 {
		return ret
	}
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(ret, constant.One), One), constant.One)
}

func div(a *big.Int, b *big.Int, roundUp bool) *big.Int {
	if roundUp {
		return DivUp(a, b)
	}
	return DivDown(a, b)
}

func DivDown(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Div(a, b)
}

func DivUp(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(a, constant.One), b), constant.One)
}

func DivUpFixed(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	aInflated := new(big.Int).Mul(a, One)
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(aInflated, constant.One), b), constant.One)
}

func DivDownFixed(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	var ret = new(big.Int).Mul(a, One)
	return new(big.Int).Div(ret, b)
}

//func divUp(a *big.Int, b *big.Int) *big.Int {
//	if a.Cmp(constant.Zero) == 0 {
//		return constant.Zero
//	}
//	var ret = new(big.Int).Mul(a, One)
//	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(ret, constant.One), b), constant.One)
//}

func DownscaleDown(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return DivDownFixed(amount, scalingFactor)
}

func CalcOutGivenIn(
	a *big.Int,
	balances []*big.Int,
	tokenIndexIn int,
	tokenIndexOut int,
	tokenAmountIn *big.Int,
	invariant *big.Int,
) *big.Int {
	balances[tokenIndexIn] = new(big.Int).Add(balances[tokenIndexIn], tokenAmountIn)
	var finalBalanceOut = GetTokenBalanceGivenInvariantAndAllOtherBalances(a, balances, invariant, tokenIndexOut)
	if finalBalanceOut == nil {
		return nil
	}
	balances[tokenIndexIn] = new(big.Int).Sub(balances[tokenIndexIn], tokenAmountIn)
	return new(big.Int).Sub(new(big.Int).Sub(balances[tokenIndexOut], finalBalanceOut), constant.One)
}

func GetTokenBalanceGivenInvariantAndAllOtherBalances(
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
		P_D = DivDown(new(big.Int).Mul(new(big.Int).Mul(P_D, balances[j]), nTokensBi), invariant)
		sum = new(big.Int).Add(sum, balances[j])
	}
	sum = new(big.Int).Sub(sum, balances[tokenIndex])
	var inv2 = new(big.Int).Mul(invariant, invariant)
	var c = new(big.Int).Mul(
		new(big.Int).Mul(DivUp(inv2, new(big.Int).Mul(ampTotal, P_D)), AmpPrecision),
		balances[tokenIndex],
	)
	var b = new(big.Int).Add(sum, new(big.Int).Mul(DivDown(invariant, ampTotal), AmpPrecision))
	var prevTokenBalance *big.Int
	var tokenBalance = DivUp(new(big.Int).Add(inv2, c), new(big.Int).Add(invariant, b))
	for i := 0; i < 255; i += 1 {
		prevTokenBalance = tokenBalance
		tokenBalance = DivUp(
			new(big.Int).Add(new(big.Int).Mul(tokenBalance, tokenBalance), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(tokenBalance, constant.Two), b), invariant),
		)
		if tokenBalance.Cmp(prevTokenBalance) > 0 {
			if new(big.Int).Sub(tokenBalance, prevTokenBalance).Cmp(constant.One) <= 0 {
				return tokenBalance
			}
		} else if new(big.Int).Sub(prevTokenBalance, tokenBalance).Cmp(constant.One) <= 0 {
			return tokenBalance
		}
	}
	return nil
}

func CalculateInvariant(A *big.Int, balances []*big.Int, roundUp bool) *big.Int {
	var sum = constant.Zero
	var numTokens = len(balances)
	var numTokensBi = big.NewInt(int64(numTokens))
	for i := 0; i < numTokens; i += 1 {
		sum = new(big.Int).Add(sum, balances[i])
	}
	if sum.Cmp(constant.Zero) == 0 {
		return constant.Zero
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
				new(big.Int).Mul(new(big.Int).Add(numTokensBi, constant.One), invariant),
				div(new(big.Int).Mul(new(big.Int).Sub(ampTotal, AmpPrecision), P_D), AmpPrecision, !roundUp),
			),
			roundUp,
		)
		if invariant.Cmp(prevInvariant) > 0 {
			if new(big.Int).Sub(invariant, prevInvariant).Cmp(constant.One) <= 0 {
				return invariant
			}
		} else if new(big.Int).Sub(prevInvariant, invariant).Cmp(constant.One) <= 0 {
			return invariant
		}
	}
	return nil
}

func ComplementFixed(x *big.Int) *big.Int {
	if x.Cmp(constant.BONE) < 0 {
		return new(big.Int).Sub(constant.BONE, x)
	}
	return big.NewInt(0)
}

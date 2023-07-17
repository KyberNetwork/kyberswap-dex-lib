package composablestable

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type ComposableStablePool struct {
	pool.Pool
	VaultAddress           string
	PoolId                 string
	ScalingFactors         []*big.Int
	ActualSupply           *big.Int
	BptIndex               int
	AmplificationParameter *big.Int
}

type droppedBpt struct {
	balances []*big.Int
	indexIn  int
	indexOut int
}

func (c ComposableStablePool) removeBpt(balances []*big.Int, tokenIndexIn, tokenIndexOut, bptIndex int) *droppedBpt {
	lenNewBalances := len(balances) - 1
	if lenNewBalances < 0 {
		lenNewBalances = 0
	}
	newBalances := make([]*big.Int, lenNewBalances)
	newTokenIndexIn := tokenIndexIn
	newTokenIndexOut := tokenIndexOut
	if bptIndex != -1 {
		// Remove the element at bptIndex
		newBalances = append(balances[:bptIndex], balances[bptIndex+1:]...)
		if bptIndex < tokenIndexIn {
			newTokenIndexIn = tokenIndexIn - 1
		}
		if bptIndex < tokenIndexOut {
			newTokenIndexOut = tokenIndexOut - 1
		}
	}
	return &droppedBpt{
		balances: newBalances,
		indexIn:  newTokenIndexIn,
		indexOut: newTokenIndexOut,
	}
}

// _subtractSwapFeeAmount
func (c ComposableStablePool) _subtractSwapFeeAmount(amount, _swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	feeAmount := balancer.MulUpFixed(amount, _swapFeePercentage)
	return new(big.Int).Sub(amount, feeAmount), feeAmount
}

func (c ComposableStablePool) _upscaleArray(amounts, scalingFactors []*big.Int) []*big.Int {
	result := make([]*big.Int, len(amounts))
	for i, amount := range amounts {
		result[i] = balancer.MulUpFixed(amount, scalingFactors[i])
	}
	return result
}

func (c ComposableStablePool) _upscale(amount, scalingFactor *big.Int) *big.Int {
	return balancer.MulUpFixed(amount, scalingFactor)
}

func (c ComposableStablePool) CalcAmountOut(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	var (
		pairType PairTypes
		indexIn  int
		indexOut int
	)
	tokens := c.Pool.GetTokens()
	for i, token := range tokens {
		if token == tokenAmountIn.Token {
			indexIn = i
		}
		if token == tokenOut {
			indexOut = i
		}
	}

	if tokenAmountIn.Token == tokens[c.BptIndex] {
		pairType = BptToToken
	} else if tokenOut == tokens[c.BptIndex] {
		pairType = TokenToBpt
	} else {
		pairType = TokenToToken
	}

	// Fees are subtracted before scaling, to reduce the complexity of the rounding direction analysis.
	tokenAmountsInWithFee, feeAmount := c._subtractSwapFeeAmount(tokenAmountIn.Amount, c.Info.SwapFee)
	balancesUpscaled := c._upscaleArray(c.Info.Reserves, c.ScalingFactors)
	tokenAmountInScaled := c._upscale(tokenAmountsInWithFee, c.ScalingFactors[indexIn])
	actualSupply := c.ActualSupply

	dropped := c.removeBpt(balancesUpscaled, indexIn, indexOut, c.BptIndex)

	amountOut := c._onSwapGivenIn(
		tokenAmountInScaled,
		dropped.balances,
		dropped.indexIn,
		dropped.indexOut,
		actualSupply,
		pairType,
	)

	// amountOut tokens are exiting the Pool, so we round down.
	amountOutScaleDown := balancer.DivDownFixed(amountOut, c.ScalingFactors[indexOut])

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: amountOutScaleDown,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: feeAmount,
		},
		Gas: balancer.DefaultGas.Swap,
	}, nil
}

func CalcBptOutGivenExactTokensIn(amp *big.Int, balances []*big.Int, amountsIn []*big.Int, bptTotalSupply *big.Int, invariant *big.Int) *big.Int {
	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	balanceRatiosWithFee := make([]*big.Int, len(amountsIn))
	invariantRatioWithFees := big.NewInt(0)
	for i, balance := range balances {
		currentWeight := balancer.DivDownFixed(balance, sumBalances)
		balanceRatiosWithFee[i] = balancer.DivDownFixed(new(big.Int).Add(balance, amountsIn[i]), balance)
		invariantRatioWithFees.Add(invariantRatioWithFees, balancer.MulDownFixed(balanceRatiosWithFee[i], currentWeight))
	}

	newBalances := make([]*big.Int, len(balances))
	for i, balance := range balances {
		var amountInWithoutFee *big.Int
		if balanceRatiosWithFee[i].Cmp(invariantRatioWithFees) > 0 {
			nonTaxableAmount := balancer.MulDownFixed(balance, new(big.Int).Sub(invariantRatioWithFees, balancer.One))
			taxableAmount := new(big.Int).Sub(amountsIn[i], nonTaxableAmount)
			amountInWithoutFee = new(big.Int).Add(nonTaxableAmount, taxableAmount)
		} else {
			amountInWithoutFee = amountsIn[i]
		}
		newBalances[i] = new(big.Int).Add(balance, amountInWithoutFee)
	}

	currentInvariant := balancer.CalculateInvariant(amp, balances, true)
	newInvariant := balancer.CalculateInvariant(amp, newBalances, false)
	invariantRatio := balancer.DivDownFixed(newInvariant, currentInvariant)

	if invariantRatio.Cmp(balancer.One) > 0 {
		return balancer.MulDownFixed(bptTotalSupply, new(big.Int).Sub(invariantRatio, balancer.One))
	} else {
		return big.NewInt(0)
	}
}

func CalcTokenOutGivenExactBptIn(amp *big.Int, balances []*big.Int, tokenIndex int, bptAmountIn *big.Int, bptTotalSupply *big.Int, invariant *big.Int) *big.Int {
	newInvariant := balancer.MulUpFixed(balancer.DivUp(new(big.Int).Sub(bptTotalSupply, bptAmountIn), bptTotalSupply), invariant)

	newBalanceTokenIndex := balancer.GetTokenBalanceGivenInvariantAndAllOtherBalances(amp, balances, newInvariant, tokenIndex)
	if newBalanceTokenIndex == nil {
		return nil
	}
	amountOutWithoutFee := new(big.Int).Sub(balances[tokenIndex], newBalanceTokenIndex)

	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	currentWeight := balancer.DivDownFixed(balances[tokenIndex], sumBalances)
	taxablePercentage := balancer.ComplementFixed(currentWeight)

	taxableAmount := balancer.MulUpFixed(amountOutWithoutFee, taxablePercentage)
	nonTaxableAmount := new(big.Int).Sub(amountOutWithoutFee, taxableAmount)

	return new(big.Int).Add(nonTaxableAmount, taxableAmount)
}

func (c ComposableStablePool) _onSwapGivenIn(
	tokenAmountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	indexOut int,
	virtualBptSupply *big.Int,
	pairType PairTypes,
) *big.Int {
	invariant := balancer.CalculateInvariant(c.AmplificationParameter, balances, true)
	var (
		amountOut *big.Int
	)
	switch pairType {
	case TokenToBpt:
		amountsIn := make([]*big.Int, len(balances))
		amountsIn[indexIn] = tokenAmountIn
		amountOut = CalcBptOutGivenExactTokensIn(c.AmplificationParameter, balances, amountsIn, virtualBptSupply, invariant)
	case BptToToken:
		amountOut = CalcTokenOutGivenExactBptIn(c.AmplificationParameter, balances, indexOut, tokenAmountIn, virtualBptSupply, invariant)
	default:
		amountOut = balancer.CalcOutGivenIn(c.AmplificationParameter, balances, indexIn, indexOut, tokenAmountIn, invariant)
	}
	return amountOut
}
func (c ComposableStablePool) UpdateBalance(params pool.UpdateBalanceParams) {
	//TODO implement me
	panic("implement me")
}

func (c ComposableStablePool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	//TODO implement me
	panic("implement me")
}

package composablestable

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type ComposableStablePool struct {
	pool.Pool
	VaultAddress                        string
	PoolId                              string
	ScalingFactors                      []*big.Int
	ActualSupply                        *big.Int
	BptIndex                            *big.Int
	AmplificationParameter              *big.Int
	TotalSupply                         *big.Int
	ProtocolFeePercentageCacheSwapType  *big.Int
	ProtocolFeePercentageCacheYieldType *big.Int

	LastJoinExit                     *balancer.LastJoinExitData
	RateProviders                    []string
	TokensExemptFromYieldProtocolFee []bool
	TokenRateCaches                  []*balancer.TokenRateCache
}

type droppedBpt struct {
	balances []*big.Int
	indexIn  int
	indexOut int
}

func NewPoolSimulator(entityPool entity.Pool) (*ComposableStablePool, error) {
	var staticExtra balancer.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra balancer.Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)
	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	totalSupply, _ := new(big.Int).SetString(entityPool.TotalSupply, 10)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	return &ComposableStablePool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    swapFee,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		VaultAddress:                        strings.ToLower(staticExtra.VaultAddress),
		PoolId:                              strings.ToLower(staticExtra.PoolId),
		AmplificationParameter:              extra.AmplificationParameter.Value,
		ScalingFactors:                      extra.ScalingFactors,
		BptIndex:                            extra.BptIndex,
		ActualSupply:                        extra.ActualSupply,
		LastJoinExit:                        extra.LastJoinExit,
		TotalSupply:                         totalSupply,
		RateProviders:                       extra.RateProviders,
		TokensExemptFromYieldProtocolFee:    extra.TokensExemptFromYieldProtocolFee,
		TokenRateCaches:                     extra.TokenRateCaches,
		ProtocolFeePercentageCacheSwapType:  extra.ProtocolFeePercentageCacheSwapType,
		ProtocolFeePercentageCacheYieldType: extra.ProtocolFeePercentageCacheYieldType,
	}, nil
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
		//pairType  PairTypes
		indexIn  int
		indexOut int
		//bptIndex  int64
		amountOut *big.Int
	)
	//bptIndex = c.BptIndex.Int64()
	tokens := c.Pool.GetTokens()
	for i, token := range tokens {
		if token == tokenAmountIn.Token {
			indexIn = i
		}
		if token == tokenOut {
			indexOut = i
		}
	}

	//if tokenAmountIn.Token == tokens[bptIndex] {
	//	pairType = BptToToken
	//} else if tokenOut == tokens[bptIndex] {
	//	pairType = TokenToBpt
	//} else {
	//	pairType = TokenToToken
	//}
	//
	//// Fees are subtracted before scaling, to reduce the complexity of the rounding direction analysis.
	//tokenAmountsInWithFee, feeAmount := c._subtractSwapFeeAmount(tokenAmountIn.Amount, c.Info.SwapFee)
	//balancesUpscaled := c._upscaleArray(c.Info.Reserves, c.ScalingFactors)
	//tokenAmountInScaled := c._upscale(tokenAmountsInWithFee, c.ScalingFactors[indexIn])
	//actualSupply := c.ActualSupply
	//
	//dropped := c.removeBpt(balancesUpscaled, indexIn, indexOut, int(bptIndex))
	//amountOut := c._onSwapGivenIn(
	//	tokenAmountInScaled,
	//	dropped.balances,
	//	dropped.indexIn,
	//	dropped.indexOut,
	//	actualSupply,
	//	pairType,
	//)
	//
	//// amountOut tokens are exiting the Pool, so we round down.
	//amountOutScaleDown := balancer.DivDownFixed(amountOut, c.ScalingFactors[indexOut])
	if tokenAmountIn.Token == c.Info.Address || tokenOut == c.Info.Address {
		amountOut = c._swapWithBptGivenIn(tokenAmountIn.Amount, c.Info.Reserves, indexIn, indexOut)
	} else {
		amountOut = c._swapGivenIn(tokenAmountIn.Amount, c.Info.Reserves, indexIn, indexOut)
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: balancer.DefaultGas.Swap,
	}, nil
}

// CalcBptOutGivenExactTokensIn https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L201
func CalcBptOutGivenExactTokensIn(amp *big.Int, balances []*big.Int, amountsIn []*big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) *big.Int {
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
			amountInWithoutFee = new(big.Int).Add(
				nonTaxableAmount,
				balancer.MulDownFixed(
					taxableAmount,
					new(big.Int).Sub(balancer.One, swapFeePercentage),
				),
			)
		} else {
			amountInWithoutFee = amountsIn[i]
		}
		newBalances[i] = new(big.Int).Add(balance, amountInWithoutFee)
	}

	newInvariant := balancer.CalculateInvariant(amp, newBalances, false)
	invariantRatio := balancer.DivDownFixed(newInvariant, invariant)

	if invariantRatio.Cmp(balancer.One) > 0 {
		return balancer.MulDownFixed(bptTotalSupply, new(big.Int).Sub(invariantRatio, balancer.One))
	} else {
		return big.NewInt(0)
	}
}

func CalcTokenOutGivenExactBptIn(amp *big.Int, balances []*big.Int, tokenIndex int, bptAmountIn *big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) *big.Int {
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

	return new(big.Int).Add(nonTaxableAmount,
		balancer.MulDownFixed(
			taxableAmount,
			new(big.Int).Sub(balancer.One, swapFeePercentage),
		),
	)
}

func (c ComposableStablePool) _onRegularSwap(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) *big.Int {
	droppedBalances := c._dropBptItem(registeredBalances)
	indexIn := c._skipBptIndex(registeredIndexIn)
	indexOut := c._skipBptIndex(registeredIndexOut)

	currentAmp := c.AmplificationParameter
	invariant := balancer.CalculateInvariant(currentAmp, droppedBalances, false)

	// given In
	return balancer.CalcOutGivenIn(currentAmp, droppedBalances, indexIn, indexOut, amountIn, invariant)
}

func (c ComposableStablePool) _onSwapGivenIn(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) *big.Int {
	return c._onRegularSwap(amountIn, registeredBalances, registeredIndexIn, registeredIndexOut)
}

func (c ComposableStablePool) _swapGivenIn(
	tokenAmountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	indexOut int,
) *big.Int {
	amountAfterFee, feeAmount := c._subtractSwapFeeAmount(tokenAmountIn, c.Info.SwapFee)

	upscaledBalances := c._upscaleArray(balances, c.ScalingFactors)
	amountUpScale := c._upscale(amountAfterFee, c.ScalingFactors[indexIn])

	amountOut := c._onSwapGivenIn(amountUpScale, upscaledBalances, indexIn, indexOut)
	fmt.Println(feeAmount)
	return balancer.DivDownFixed(amountOut, c.ScalingFactors[indexOut])
}

func (c ComposableStablePool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = c.GetTokenIndex(input.Token)
	var tokenOutIndex = c.GetTokenIndex(output.Token)
	if tokenInIndex >= 0 {
		c.Info.Reserves[tokenInIndex] = new(big.Int).Add(c.Info.Reserves[tokenInIndex], input.Amount)
	}
	if tokenOutIndex >= 0 {
		c.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(c.Info.Reserves[tokenOutIndex], output.Amount)
	}
}

func (c ComposableStablePool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return balancer.Meta{
		VaultAddress: c.VaultAddress,
		PoolId:       c.PoolId,
	}
}

// _swapWithBptGivenIn
// reference https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L314
func (c ComposableStablePool) _swapWithBptGivenIn(
	tokenAmountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn int,
	registeredIndexOut int,
) *big.Int {
	var (
		amountCalculated   *big.Int
		postJoinExitSupply *big.Int
	)

	balancesUpscaled := c._upscaleArray(registeredBalances, c.ScalingFactors)
	tokenAmountInScaled := c._upscale(tokenAmountIn, c.ScalingFactors[registeredIndexIn])

	preJoinExitSupply, balances, currentAmp, preJoinExitInvariant := c._beforeJoinExit(balancesUpscaled)

	if registeredIndexOut == int(c.BptIndex.Int64()) {
		amountCalculated, postJoinExitSupply = c._doJoinSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexIn),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
	} else {
		amountCalculated, postJoinExitSupply = c._doExitSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexOut),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
	}
	fmt.Println(amountCalculated, postJoinExitSupply)
	return balancer.DivDownFixed(amountCalculated, c.ScalingFactors[registeredIndexOut])
}

func (c ComposableStablePool) _exitSwapExactBptInForTokenOut(
	bptAmount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int) {
	amountOut := CalcTokenOutGivenExactBptIn(currentAmp, balances, indexOut, bptAmount, actualSupply, preJoinExitInvariant, c.Info.SwapFee)

	balances[indexOut].Sub(balances[indexOut], amountOut)
	postJoinExitSupply := new(big.Int).Sub(actualSupply, bptAmount)

	return amountOut, postJoinExitSupply

}
func (c ComposableStablePool) _doJoinSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int) {
	if isGivenIn {
		return c._joinSwapExactTokenInForBptOut(amount, balances, indexIn, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil
}

func (c ComposableStablePool) _doExitSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int) {
	if isGivenIn {
		return c._exitSwapExactBptInForTokenOut(amount, balances, indexOut, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil
}

/** _joinSwapExactTokenInForBptOut
 * @dev Since this is a join, we know the tokenOut is BPT. Since it is GivenIn, we know the tokenIn amount,
 * and must calculate the BPT amount out.
 * We are moving preminted BPT out of the Vault, which increases the virtual supply.
 * Ref: https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L409
 */
func (c ComposableStablePool) _joinSwapExactTokenInForBptOut(
	amountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int) {

	amountsIn := make([]*big.Int, len(balances))
	for i := range amountsIn {
		amountsIn[i] = new(big.Int)
	}
	amountsIn[indexIn] = amountIn
	bptOut := CalcBptOutGivenExactTokensIn(currentAmp, balances, amountsIn, actualSupply, preJoinExitInvariant, c.Info.SwapFee)

	balances[indexIn].Add(balances[indexIn], amountIn)
	postJoinExitSupply := new(big.Int).Add(actualSupply, bptOut)

	return bptOut, postJoinExitSupply
}

func (c ComposableStablePool) _beforeJoinExit(registeredBalances []*big.Int) (*big.Int, []*big.Int, *big.Int, *big.Int) {
	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant := c._payProtocolFeesBeforeJoinExit(registeredBalances)
	currentAmp := c.AmplificationParameter

	var preJoinExitInvariant *big.Int

	if currentAmp.Cmp(c.LastJoinExit.LastJoinExitAmplification) == 0 {
		preJoinExitInvariant = oldAmpPreJoinExitInvariant
	} else {
		preJoinExitInvariant = balancer.CalculateInvariant(currentAmp, balances, false)
	}

	return preJoinExitSupply, balances, currentAmp, preJoinExitInvariant

}

/** _payProtocolFeesBeforeJoinExit
 * @dev Calculates due protocol fees originating from accumulated swap fees and yield of non-exempt tokens, pays
 * them by minting BPT, and returns the actual supply and current balances.
 *
 * We also return the current invariant computed using the amplification factor at the last join or exit, which can
 * be useful to skip computations in scenarios where the amplification factor is not changing.
 * Ref: https://github.com/balancer/balancer-v2-monorepo/blob/3251913e63949f35be168b42987d0aae297a01b1/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L64
 */
func (c ComposableStablePool) _payProtocolFeesBeforeJoinExit(
	registeredBalances []*big.Int,
) (*big.Int, []*big.Int, *big.Int) {
	virtualSupply, droppedBalances := c._dropBptItemFromBalances(registeredBalances)
	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp := c._getProtocolPoolOwnershipPercentage(droppedBalances)

	protocolFeeAmount := c.bptForPoolOwnershipPercentage(virtualSupply, expectedProtocolOwnershipPercentage)

	return new(big.Int).Add(virtualSupply, protocolFeeAmount), droppedBalances, currentInvariantWithLastJoinExitAmp
}

// _getProtocolPoolOwnershipPercentage
// https://github.com/balancer/balancer-v2-monorepo/blob/3251913e63949f35be168b42987d0aae297a01b1/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L102
func (c ComposableStablePool) _getProtocolPoolOwnershipPercentage(balances []*big.Int) (*big.Int, *big.Int) {
	swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant := c._getGrowthInvariants(balances)

	// Calculate the delta for swap fee growth invariant
	swapFeeGrowthInvariantDelta := new(big.Int).Sub(swapFeeGrowthInvariant, c.LastJoinExit.LastPostJoinExitInvariant)
	if swapFeeGrowthInvariantDelta.Cmp(bignumber.ZeroBI) < 0 {
		swapFeeGrowthInvariantDelta.SetUint64(0)
	}

	// Calculate the delta for non-exempt yield growth invariant
	nonExemptYieldGrowthInvariantDelta := new(big.Int).Sub(totalNonExemptGrowthInvariant, swapFeeGrowthInvariant)
	if nonExemptYieldGrowthInvariantDelta.Cmp(bignumber.ZeroBI) < 0 {
		nonExemptYieldGrowthInvariantDelta.SetUint64(0)
	}

	//swapFeeGrowthInvariantDelta/totalGrowthInvariant*getProtocolFeePercentageCache
	protocolSwapFeePercentage := balancer.MulDownFixed(
		balancer.DivDownFixed(swapFeeGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheSwapType)

	protocolYieldPercentage := balancer.MulDownFixed(
		balancer.DivDownFixed(nonExemptYieldGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheYieldType)

	// Calculate the total protocol Pool ownership percentage
	protocolPoolOwnershipPercentage := new(big.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage)

	return protocolPoolOwnershipPercentage, totalGrowthInvariant
}

func (c ComposableStablePool) _getGrowthInvariants(balances []*big.Int) (*big.Int, *big.Int, *big.Int) {
	var (
		swapFeeGrowthInvariant        *big.Int
		totalNonExemptGrowthInvariant *big.Int
		totalGrowthInvariant          *big.Int
	)

	// This invariant result is calc by DivDown (round down)
	// DivDown https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L96
	swapFeeGrowthInvariant = balancer.CalculateInvariant(
		c.LastJoinExit.LastJoinExitAmplification,
		c.getAdjustedBalances(balances, true), false)

	// For the other invariants, we can potentially skip some work. In the edge cases where none or all of the
	// tokens are exempt from yield, there's one fewer invariant to compute.
	if c._areNoTokensExempt() {
		// If there are no tokens with fee-exempt yield, then the total non-exempt growth will equal the total
		// growth: all yield growth is non-exempt. There's also no point in adjusting balances, since we
		// already know none are exempt.
		totalNonExemptGrowthInvariant = balancer.CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
		totalGrowthInvariant = totalNonExemptGrowthInvariant
	} else if c._areAllTokensExempt() {
		// If no tokens are charged fees on yield, then the non-exempt growth is equal to the swap fee growth - no
		// yield fees will be collected.
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
		totalGrowthInvariant = balancer.CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
	} else {
		// In the general case, we need to calculate two invariants: one with some adjusted balances, and one with
		// the current balances.

		totalNonExemptGrowthInvariant = balancer.CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			c.getAdjustedBalances(balances, false), // Only adjust non-exempt balances
			false,
		)

		totalGrowthInvariant = balancer.CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			balances,
			false)
	}
	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant
}
func (c ComposableStablePool) _dropBptItemFromBalances(balances []*big.Int) (*big.Int, []*big.Int) {
	return c._getVirtualSupply(balances[c.BptIndex.Int64()]), c._dropBptItem(balances)
}

func (c ComposableStablePool) _getVirtualSupply(bptBalance *big.Int) *big.Int {
	return new(big.Int).Sub(c.TotalSupply, bptBalance)
}

func (c ComposableStablePool) _hasRateProvider(tokenIndex int) bool {
	return c.RateProviders[tokenIndex] != ""
}

func (c ComposableStablePool) isTokenExemptFromYieldProtocolFee(tokenIndex int) bool {
	return c.TokensExemptFromYieldProtocolFee[tokenIndex]
}

func (c ComposableStablePool) _areNoTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if exempt {
			return false
		}
	}
	return true
}

func (c ComposableStablePool) _areAllTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if exempt == false {
			return false
		}
	}
	return true
}

func (c ComposableStablePool) getAdjustedBalances(balances []*big.Int, ignoreExemptFlags bool) []*big.Int {
	totalTokensWithoutBpt := len(balances)
	adjustedBalances := make([]*big.Int, totalTokensWithoutBpt)

	for i := 0; i < totalTokensWithoutBpt; i++ {
		skipBptIndex := i
		if i >= int(c.BptIndex.Int64()) {
			skipBptIndex++
		}

		if c.isTokenExemptFromYieldProtocolFee(skipBptIndex) || (ignoreExemptFlags && c._hasRateProvider(skipBptIndex)) {
			adjustedBalances[i] = c._adjustedBalance(balances[i], c.TokenRateCaches[skipBptIndex])
		} else {
			adjustedBalances[i] = balances[i]
		}
	}

	return adjustedBalances
}

// _adjustedBalance Compute balance * oldRate/currentRate, doing division last to minimize rounding error.
func (c ComposableStablePool) _adjustedBalance(balance *big.Int, cache *balancer.TokenRateCache) *big.Int {
	return balancer.DivDown(new(big.Int).Mul(balance, cache.OldRate), cache.Rate)
}

// _dropBptItem Remove the item at `_bptIndex` from an arbitrary array (e.g., amountsIn).
func (c ComposableStablePool) _dropBptItem(amounts []*big.Int) []*big.Int {
	amountsWithoutBpt := make([]*big.Int, len(amounts)-1)
	bptIndex := int(c.BptIndex.Int64())

	for i := 0; i < len(amountsWithoutBpt); i++ {
		if i < bptIndex {
			amountsWithoutBpt[i] = new(big.Int).Set(amounts[i])
		} else {
			amountsWithoutBpt[i] = new(big.Int).Set(amounts[i+1])
		}
	}

	return amountsWithoutBpt
}

/**
 * @dev Calculates the amount of BPT necessary to give ownership of a given percentage of the Pool to an external
 * third party. In the case of protocol fees, this is the DAO, but could also be a pool manager, etc.
 * Note that this function reverts if `poolPercentage` >= 100%, it's expected that the caller will enforce this.
 * @param totalSupply - The total supply of the pool prior to minting BPT.
 * @param poolOwnershipPercentage - The desired ownership percentage of the pool to have as a result of minting BPT.
 * @return bptAmount - The amount of BPT to mint such that it is `poolPercentage` of the resultant total supply.
 */
func (c ComposableStablePool) bptForPoolOwnershipPercentage(totalSupply, poolOwnershipPercentage *big.Int) *big.Int {
	// If we mint some amount `bptAmount` of BPT then the percentage ownership of the pool this grants is given by:
	// `poolOwnershipPercentage = bptAmount / (totalSupply + bptAmount)`.
	// Solving for `bptAmount`, we arrive at:
	// `bptAmount = totalSupply * poolOwnershipPercentage / (1 - poolOwnershipPercentage)`.
	return balancer.DivDown(new(big.Int).Mul(totalSupply, poolOwnershipPercentage), balancer.ComplementFixed(poolOwnershipPercentage))
}

func (c ComposableStablePool) _skipBptIndex(index int) int {
	if index < int(c.BptIndex.Int64()) {
		return index
	} else {
		return index - 1
	}
}

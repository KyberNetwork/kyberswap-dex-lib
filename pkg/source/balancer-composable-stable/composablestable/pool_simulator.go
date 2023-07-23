package composablestable

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	composableStable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer-composable-stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Pool struct {
	pool.Pool
	VaultAddress                        string
	PoolId                              string
	ScalingFactors                      []*big.Int
	BptIndex                            *big.Int
	AmplificationParameter              *big.Int
	TotalSupply                         *big.Int
	ProtocolFeePercentageCacheSwapType  *big.Int
	ProtocolFeePercentageCacheYieldType *big.Int

	LastJoinExit                     *composableStable.LastJoinExitData
	RateProviders                    []string
	TokensExemptFromYieldProtocolFee []bool
	TokenRateCaches                  []composableStable.TokenRateCache
}

func NewPoolSimulator(entityPool entity.Pool) (*Pool, error) {
	var staticExtra composableStable.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra composableStable.Extra
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

	return &Pool{
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
		LastJoinExit:                        extra.LastJoinExit,
		TotalSupply:                         totalSupply,
		RateProviders:                       extra.RateProviders,
		TokensExemptFromYieldProtocolFee:    extra.TokensExemptFromYieldProtocolFee,
		TokenRateCaches:                     extra.TokenRateCaches,
		ProtocolFeePercentageCacheSwapType:  extra.ProtocolFeePercentageCacheSwapType,
		ProtocolFeePercentageCacheYieldType: extra.ProtocolFeePercentageCacheYieldType,
	}, nil
}

func (c Pool) CalcAmountOut(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	var (
		indexIn   int
		indexOut  int
		amountOut *big.Int
		fee       *pool.TokenAmount
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

	if tokenAmountIn.Token == c.Info.Address || tokenOut == c.Info.Address {
		amountOut, fee = c._swapWithBptGivenIn(tokenAmountIn.Amount, c.Info.Reserves, indexIn, indexOut)
	} else {
		amountOut, fee = c._swapGivenIn(tokenAmountIn.Amount, c.Info.Reserves, indexIn, indexOut)
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: fee,
		Gas: composableStable.DefaultGas.Swap,
	}, nil
}

func (c Pool) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (c Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return composableStable.Meta{
		VaultAddress: c.VaultAddress,
		PoolId:       c.PoolId,
	}
}

// calcBptOutGivenExactTokensIn https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L201
func calcBptOutGivenExactTokensIn(amp *big.Int, balances []*big.Int, amountsIn []*big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	feeAmountIn := big.NewInt(0)
	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	balanceRatiosWithFee := make([]*big.Int, len(amountsIn))
	invariantRatioWithFees := big.NewInt(0)
	for i, balance := range balances {
		currentWeight := composableStable.DivDownFixed(balance, sumBalances)
		balanceRatiosWithFee[i] = composableStable.DivDownFixed(new(big.Int).Add(balance, amountsIn[i]), balance)
		invariantRatioWithFees.Add(invariantRatioWithFees, composableStable.MulDownFixed(balanceRatiosWithFee[i], currentWeight))
	}

	newBalances := make([]*big.Int, len(balances))
	for i, balance := range balances {
		var amountInWithoutFee *big.Int
		if balanceRatiosWithFee[i].Cmp(invariantRatioWithFees) > 0 {
			nonTaxableAmount := composableStable.MulDownFixed(balance, new(big.Int).Sub(invariantRatioWithFees, composableStable.One))
			taxableAmount := new(big.Int).Sub(amountsIn[i], nonTaxableAmount)
			amountInWithoutFee = new(big.Int).Add(
				nonTaxableAmount,
				composableStable.MulDownFixed(
					taxableAmount,
					new(big.Int).Sub(composableStable.One, swapFeePercentage),
				),
			)
		} else {
			amountInWithoutFee = amountsIn[i]
		}
		feeAmountIn = feeAmountIn.Add(feeAmountIn, new(big.Int).Sub(amountsIn[i], amountInWithoutFee))
		newBalances[i] = new(big.Int).Add(balance, amountInWithoutFee)
	}

	newInvariant := composableStable.CalculateInvariant(amp, newBalances, false)
	invariantRatio := composableStable.DivDownFixed(newInvariant, invariant)

	if invariantRatio.Cmp(composableStable.One) > 0 {
		return composableStable.MulDownFixed(bptTotalSupply, new(big.Int).Sub(invariantRatio, composableStable.One)), feeAmountIn
	} else {
		return big.NewInt(0), feeAmountIn
	}
}

func calcTokenOutGivenExactBptIn(amp *big.Int, balances []*big.Int, tokenIndex int, bptAmountIn *big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	newInvariant := composableStable.MulUpFixed(composableStable.DivUpFixed(new(big.Int).Sub(bptTotalSupply, bptAmountIn), bptTotalSupply), invariant)
	newBalanceTokenIndex := composableStable.GetTokenBalanceGivenInvariantAndAllOtherBalances(amp, balances, newInvariant, tokenIndex)
	if newBalanceTokenIndex == nil {
		return nil, nil
	}
	amountOutWithoutFee := new(big.Int).Sub(balances[tokenIndex], newBalanceTokenIndex)

	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	currentWeight := composableStable.DivDownFixed(balances[tokenIndex], sumBalances)
	taxablePercentage := composableStable.ComplementFixed(currentWeight)

	taxableAmount := composableStable.MulUpFixed(amountOutWithoutFee, taxablePercentage)
	nonTaxableAmount := new(big.Int).Sub(amountOutWithoutFee, taxableAmount)

	feeOfTaxableAmount := composableStable.MulDownFixed(
		taxableAmount,
		new(big.Int).Sub(composableStable.One, swapFeePercentage),
	)

	feeAmount := new(big.Int).Sub(taxableAmount, feeOfTaxableAmount)
	return new(big.Int).Add(nonTaxableAmount, feeOfTaxableAmount), feeAmount
}

func (c Pool) _onRegularSwap(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) *big.Int {
	droppedBalances := c._dropBptItem(registeredBalances)
	indexIn := c._skipBptIndex(registeredIndexIn)
	indexOut := c._skipBptIndex(registeredIndexOut)

	currentAmp := c.AmplificationParameter
	invariant := composableStable.CalculateInvariant(currentAmp, droppedBalances, false)

	// given In
	return composableStable.CalcOutGivenIn(currentAmp, droppedBalances, indexIn, indexOut, amountIn, invariant)
}

func (c Pool) _onSwapGivenIn(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) *big.Int {
	return c._onRegularSwap(amountIn, registeredBalances, registeredIndexIn, registeredIndexOut)
}

func (c Pool) _swapGivenIn(
	tokenAmountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	indexOut int,
) (*big.Int, *pool.TokenAmount) {
	amountAfterFee, feeAmount := c._subtractSwapFeeAmount(tokenAmountIn, c.Info.SwapFee)

	upscaledBalances := c._upscaleArray(balances, c.ScalingFactors)
	amountUpScale := c._upscale(amountAfterFee, c.ScalingFactors[indexIn])

	amountOut := c._onSwapGivenIn(amountUpScale, upscaledBalances, indexIn, indexOut)
	return composableStable.DivDownFixed(amountOut, c.ScalingFactors[indexOut]),
		&pool.TokenAmount{Token: c.Info.Tokens[indexIn], Amount: feeAmount}
}

// _swapWithBptGivenIn
// reference https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L314
func (c Pool) _swapWithBptGivenIn(
	tokenAmountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn int,
	registeredIndexOut int,
) (*big.Int, *pool.TokenAmount) {
	var (
		amountCalculated *big.Int
		_                *big.Int
		feeAmount        *big.Int
		tokenAmount      pool.TokenAmount
	)
	balancesUpscaled := c._upscaleArray(registeredBalances, c.ScalingFactors)
	tokenAmountInScaled := c._upscale(tokenAmountIn, c.ScalingFactors[registeredIndexIn])
	preJoinExitSupply, balances, currentAmp, preJoinExitInvariant := c._beforeJoinExit(balancesUpscaled)

	if registeredIndexOut == int(c.BptIndex.Int64()) {
		amountCalculated, _, feeAmount = c._doJoinSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexIn),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		// charge fee amountIn
		tokenAmount.Token = c.Info.Tokens[registeredIndexIn]
		tokenAmount.Amount = feeAmount
	} else {
		amountCalculated, _, feeAmount = c._doExitSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexOut),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		// charge fee amountOut
		tokenAmount.Token = c.Info.Tokens[registeredIndexOut]
		tokenAmount.Amount = feeAmount
	}
	return composableStable.DivDownFixed(amountCalculated, c.ScalingFactors[registeredIndexOut]), &tokenAmount
}

func (c Pool) _exitSwapExactBptInForTokenOut(
	bptAmount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int) {
	amountOut, feeAmount := calcTokenOutGivenExactBptIn(currentAmp, balances, indexOut, bptAmount, actualSupply, preJoinExitInvariant, c.Info.SwapFee)

	balances[indexOut].Sub(balances[indexOut], amountOut)
	postJoinExitSupply := new(big.Int).Sub(actualSupply, bptAmount)

	return amountOut, postJoinExitSupply, feeAmount

}

func (c Pool) _doJoinSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int) {
	if isGivenIn {
		return c._joinSwapExactTokenInForBptOut(amount, balances, indexIn, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil
}

func (c Pool) _doExitSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int) {
	if isGivenIn {
		return c._exitSwapExactBptInForTokenOut(amount, balances, indexOut, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil
}

/** _joinSwapExactTokenInForBptOut
 * @dev Since this is a join, we know the tokenOut is BPT. Since it is GivenIn, we know the tokenIn amount,
 * and must calculate the BPT amount out.
 * We are moving preminted BPT out of the Vault, which increases the virtual supply.
 * Ref: https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L409
 */
func (c Pool) _joinSwapExactTokenInForBptOut(
	amountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int) {

	amountsIn := make([]*big.Int, len(balances))
	for i := range amountsIn {
		amountsIn[i] = new(big.Int)
	}
	amountsIn[indexIn] = amountIn
	bptOut, feeAmountIn := calcBptOutGivenExactTokensIn(currentAmp, balances, amountsIn, actualSupply, preJoinExitInvariant, c.Info.SwapFee)

	balances[indexIn].Add(balances[indexIn], amountIn)
	postJoinExitSupply := new(big.Int).Add(actualSupply, bptOut)

	return bptOut, postJoinExitSupply, feeAmountIn
}

func (c Pool) _beforeJoinExit(registeredBalances []*big.Int) (*big.Int, []*big.Int, *big.Int, *big.Int) {
	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant := c._payProtocolFeesBeforeJoinExit(registeredBalances)
	currentAmp := c.AmplificationParameter

	var preJoinExitInvariant *big.Int

	if currentAmp.Cmp(c.LastJoinExit.LastJoinExitAmplification) == 0 {
		preJoinExitInvariant = oldAmpPreJoinExitInvariant
	} else {
		preJoinExitInvariant = composableStable.CalculateInvariant(currentAmp, balances, false)
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
func (c Pool) _payProtocolFeesBeforeJoinExit(
	registeredBalances []*big.Int,
) (*big.Int, []*big.Int, *big.Int) {
	virtualSupply, droppedBalances := c._dropBptItemFromBalances(registeredBalances)
	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp := c._getProtocolPoolOwnershipPercentage(droppedBalances)

	protocolFeeAmount := c.bptForPoolOwnershipPercentage(virtualSupply, expectedProtocolOwnershipPercentage)

	return new(big.Int).Add(virtualSupply, protocolFeeAmount), droppedBalances, currentInvariantWithLastJoinExitAmp
}

// _getProtocolPoolOwnershipPercentage
// https://github.com/balancer/balancer-v2-monorepo/blob/3251913e63949f35be168b42987d0aae297a01b1/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L102
func (c Pool) _getProtocolPoolOwnershipPercentage(balances []*big.Int) (*big.Int, *big.Int) {
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
	protocolSwapFeePercentage := composableStable.MulDownFixed(
		composableStable.DivDownFixed(swapFeeGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheSwapType)

	protocolYieldPercentage := composableStable.MulDownFixed(
		composableStable.DivDownFixed(nonExemptYieldGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheYieldType)

	// Calculate the total protocol Pool ownership percentage
	protocolPoolOwnershipPercentage := new(big.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage)

	return protocolPoolOwnershipPercentage, totalGrowthInvariant
}

func (c Pool) _getGrowthInvariants(balances []*big.Int) (*big.Int, *big.Int, *big.Int) {
	var (
		swapFeeGrowthInvariant        *big.Int
		totalNonExemptGrowthInvariant *big.Int
		totalGrowthInvariant          *big.Int
	)

	// This invariant result is calc by DivDown (round down)
	// DivDown https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L96
	swapFeeGrowthInvariant = composableStable.CalculateInvariant(
		c.LastJoinExit.LastJoinExitAmplification,
		c.getAdjustedBalances(balances, true), false)

	// For the other invariants, we can potentially skip some work. In the edge cases where none or all of the
	// tokens are exempt from yield, there's one fewer invariant to compute.
	if c._areNoTokensExempt() {
		// If there are no tokens with fee-exempt yield, then the total non-exempt growth will equal the total
		// growth: all yield growth is non-exempt. There's also no point in adjusting balances, since we
		// already know none are exempt.
		totalNonExemptGrowthInvariant = composableStable.CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
		totalGrowthInvariant = totalNonExemptGrowthInvariant
	} else if c._areAllTokensExempt() {
		// If no tokens are charged fees on yield, then the non-exempt growth is equal to the swap fee growth - no
		// yield fees will be collected.
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
		totalGrowthInvariant = composableStable.CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
	} else {
		// In the general case, we need to calculate two invariants: one with some adjusted balances, and one with
		// the current balances.

		totalNonExemptGrowthInvariant = composableStable.CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			c.getAdjustedBalances(balances, false), // Only adjust non-exempt balances
			false,
		)

		totalGrowthInvariant = composableStable.CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			balances,
			false)
	}
	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant
}
func (c Pool) _dropBptItemFromBalances(balances []*big.Int) (*big.Int, []*big.Int) {
	return c._getVirtualSupply(balances[c.BptIndex.Int64()]), c._dropBptItem(balances)
}

func (c Pool) _getVirtualSupply(bptBalance *big.Int) *big.Int {
	return new(big.Int).Sub(c.TotalSupply, bptBalance)
}

func (c Pool) _hasRateProvider(tokenIndex int) bool {
	if c.RateProviders[tokenIndex] == "" || c.RateProviders[tokenIndex] == valueobject.ZeroAddress {
		return false
	}
	return true
}

func (c Pool) isTokenExemptFromYieldProtocolFee(tokenIndex int) bool {
	return c.TokensExemptFromYieldProtocolFee[tokenIndex]
}

func (c Pool) _areNoTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if exempt {
			return false
		}
	}
	return true
}

func (c Pool) _areAllTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if !exempt {
			return false
		}
	}
	return true
}

func (c Pool) getAdjustedBalances(balances []*big.Int, ignoreExemptFlags bool) []*big.Int {
	totalTokensWithoutBpt := len(balances)
	adjustedBalances := make([]*big.Int, totalTokensWithoutBpt)

	for i := 0; i < totalTokensWithoutBpt; i++ {
		skipBptIndex := i
		if i >= int(c.BptIndex.Int64()) {
			skipBptIndex++
		}

		if c.isTokenExemptFromYieldProtocolFee(skipBptIndex) || (ignoreExemptFlags && c._hasRateProvider(skipBptIndex)) {
			adjustedBalances[i] = c._adjustedBalance(balances[i], &c.TokenRateCaches[skipBptIndex])
		} else {
			adjustedBalances[i] = balances[i]
		}
	}

	return adjustedBalances
}

// _adjustedBalance Compute balance * oldRate/currentRate, doing division last to minimize rounding error.
func (c Pool) _adjustedBalance(balance *big.Int, cache *composableStable.TokenRateCache) *big.Int {
	return composableStable.DivDown(new(big.Int).Mul(balance, cache.OldRate), cache.Rate)
}

// _dropBptItem Remove the item at `_bptIndex` from an arbitrary array (e.g., amountsIn).
func (c Pool) _dropBptItem(amounts []*big.Int) []*big.Int {
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
func (c Pool) bptForPoolOwnershipPercentage(totalSupply, poolOwnershipPercentage *big.Int) *big.Int {
	// If we mint some amount `bptAmount` of BPT then the percentage ownership of the pool this grants is given by:
	// `poolOwnershipPercentage = bptAmount / (totalSupply + bptAmount)`.
	// Solving for `bptAmount`, we arrive at:
	// `bptAmount = totalSupply * poolOwnershipPercentage / (1 - poolOwnershipPercentage)`.
	return composableStable.DivDown(new(big.Int).Mul(totalSupply, poolOwnershipPercentage), composableStable.ComplementFixed(poolOwnershipPercentage))
}

func (c Pool) _skipBptIndex(index int) int {
	if index < int(c.BptIndex.Int64()) {
		return index
	} else {
		return index - 1
	}
}

// _subtractSwapFeeAmount
func (c Pool) _subtractSwapFeeAmount(amount, _swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	feeAmount := composableStable.MulUpFixed(amount, _swapFeePercentage)
	return new(big.Int).Sub(amount, feeAmount), feeAmount
}

func (c Pool) _upscaleArray(amounts, scalingFactors []*big.Int) []*big.Int {
	result := make([]*big.Int, len(amounts))
	for i, amount := range amounts {
		result[i] = composableStable.MulUpFixed(amount, scalingFactors[i])
	}
	return result
}

func (c Pool) _upscale(amount, scalingFactor *big.Int) *big.Int {
	return composableStable.MulUpFixed(amount, scalingFactor)
}

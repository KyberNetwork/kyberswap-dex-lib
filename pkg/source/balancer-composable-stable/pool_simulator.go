package balancercomposablestable

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	VaultAddress                        string
	PoolId                              string
	ScalingFactors                      []*big.Int
	BptIndex                            *big.Int
	AmplificationParameter              *big.Int
	TotalSupply                         *big.Int
	ProtocolFeePercentageCacheSwapType  *big.Int
	ProtocolFeePercentageCacheYieldType *big.Int

	LastJoinExit                     *LastJoinExitData
	RateProviders                    []string
	TokensExemptFromYieldProtocolFee []bool
	TokenRateCaches                  []TokenRateCache
	mapTokenAddressToIndex           map[string]int
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)
	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	totalSupply, _ := new(big.Int).SetString(entityPool.TotalSupply, 10)
	mapTokenAddressToIndex := make(map[string]int)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		mapTokenAddressToIndex[entityPool.Tokens[i].Address] = i
	}

	return &PoolSimulator{
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
		mapTokenAddressToIndex:              mapTokenAddressToIndex,
	}, nil
}

func (c *PoolSimulator) CalcAmountOut(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	var (
		indexIn   = c.mapTokenAddressToIndex[tokenAmountIn.Token]
		indexOut  = c.mapTokenAddressToIndex[tokenOut]
		amountOut *big.Int
		fee       *pool.TokenAmount
		err       error
	)

	if tokenAmountIn.Token == c.Info.Address || tokenOut == c.Info.Address {
		amountOut, fee, err = c._swapWithBptGivenIn(tokenAmountIn.Amount, indexIn, indexOut)
	} else {
		amountOut, fee, err = c._swapGivenIn(tokenAmountIn.Amount, indexIn, indexOut)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: fee,
		Gas: DefaultGas.Swap,
	}, nil
}

func (c *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (c *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return Meta{
		VaultAddress:           c.VaultAddress,
		PoolId:                 c.PoolId,
		MapTokenAddressToIndex: c.mapTokenAddressToIndex,
	}
}

func (c *PoolSimulator) _onRegularSwap(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) (*big.Int, error) {
	droppedBalances := c._dropBptItem(registeredBalances)
	indexIn := c._skipBptIndex(registeredIndexIn)
	indexOut := c._skipBptIndex(registeredIndexOut)

	currentAmp := c.AmplificationParameter
	invariant, err := CalculateInvariant(currentAmp, droppedBalances, false)
	if err != nil {
		return nil, err
	}
	// given In
	return CalcOutGivenIn(currentAmp, droppedBalances, indexIn, indexOut, amountIn, invariant)
}

func (c *PoolSimulator) _onSwapGivenIn(
	amountIn *big.Int,
	registeredBalances []*big.Int,
	registeredIndexIn,
	registeredIndexOut int,
) (*big.Int, error) {
	return c._onRegularSwap(amountIn, registeredBalances, registeredIndexIn, registeredIndexOut)
}

func (c *PoolSimulator) _swapGivenIn(
	tokenAmountIn *big.Int,
	indexIn int,
	indexOut int,
) (*big.Int, *pool.TokenAmount, error) {
	amountAfterFee, feeAmount := c._subtractSwapFeeAmount(tokenAmountIn, c.Info.SwapFee)

	upscaledBalances := c._upscaleArray(c.Info.Reserves, c.ScalingFactors)
	amountUpScale := c._upscale(amountAfterFee, c.ScalingFactors[indexIn])

	amountOut, err := c._onSwapGivenIn(amountUpScale, upscaledBalances, indexIn, indexOut)
	if err != nil {
		return nil, nil, err
	}
	return DivDownFixed(amountOut, c.ScalingFactors[indexOut]),
		&pool.TokenAmount{Token: c.Info.Tokens[indexIn], Amount: feeAmount}, nil
}

// _swapWithBptGivenIn
// reference https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L314
func (c *PoolSimulator) _swapWithBptGivenIn(
	tokenAmountIn *big.Int,
	registeredIndexIn int,
	registeredIndexOut int,
) (*big.Int, *pool.TokenAmount, error) {
	var (
		amountCalculated *big.Int
		_                *big.Int
		feeAmount        *big.Int
		tokenAmount      pool.TokenAmount
	)
	balancesUpscaled := c._upscaleArray(c.Info.Reserves, c.ScalingFactors)
	tokenAmountInScaled := c._upscale(tokenAmountIn, c.ScalingFactors[registeredIndexIn])

	preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, err := c._beforeJoinExit(balancesUpscaled)
	if err != nil {
		return nil, nil, err
	}
	if registeredIndexOut == int(c.BptIndex.Int64()) {
		amountCalculated, _, feeAmount, err = c._doJoinSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexIn),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		if err != nil {
			return nil, nil, err
		}
		// charge fee amountIn
		tokenAmount.Token = c.Info.Tokens[registeredIndexIn]
		tokenAmount.Amount = feeAmount
	} else {
		amountCalculated, _, feeAmount, err = c._doExitSwap(
			true,
			tokenAmountInScaled,
			balances,
			c._skipBptIndex(registeredIndexOut),
			currentAmp,
			preJoinExitSupply,
			preJoinExitInvariant,
		)
		if err != nil {
			return nil, nil, err
		}
		// charge fee amountOut
		tokenAmount.Token = c.Info.Tokens[registeredIndexOut]
		tokenAmount.Amount = feeAmount
	}
	if amountCalculated == nil {
		return nil, nil, ErrorInvalidAmountOutCalculated
	}
	return DivDownFixed(amountCalculated, c.ScalingFactors[registeredIndexOut]), &tokenAmount, nil
}

func (c *PoolSimulator) _exitSwapExactBptInForTokenOut(
	bptAmount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	amountOut, feeAmount, err := calcTokenOutGivenExactBptIn(currentAmp, balances, indexOut, bptAmount, actualSupply, preJoinExitInvariant, c.Info.SwapFee)
	if err != nil {
		return nil, nil, nil, err
	}

	balances[indexOut].Sub(balances[indexOut], amountOut)
	postJoinExitSupply := new(big.Int).Sub(actualSupply, bptAmount)

	return amountOut, postJoinExitSupply, feeAmount, nil

}

func (c *PoolSimulator) _doJoinSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	if isGivenIn {
		return c._joinSwapExactTokenInForBptOut(amount, balances, indexIn, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil, nil
}

func (c *PoolSimulator) _doExitSwap(
	isGivenIn bool,
	amount *big.Int,
	balances []*big.Int,
	indexOut int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	if isGivenIn {
		return c._exitSwapExactBptInForTokenOut(amount, balances, indexOut, currentAmp, actualSupply, preJoinExitInvariant)
	}
	// Currently ignore givenOut case
	return nil, nil, nil, nil
}

/** _joinSwapExactTokenInForBptOut
 * @dev Since this is a join, we know the tokenOut is BPT. Since it is GivenIn, we know the tokenIn amount,
 * and must calculate the BPT amount out.
 * We are moving preminted BPT out of the Vault, which increases the virtual supply.
 * Ref: https://github.com/balancer/balancer-v2-monorepo/blob/872342e060bfc31c3ab6a1deb7b1d3050ea7e19d/pkg/pool-stable/contracts/ComposableStablePool.sol#L409
 */
func (c *PoolSimulator) _joinSwapExactTokenInForBptOut(
	amountIn *big.Int,
	balances []*big.Int,
	indexIn int,
	currentAmp *big.Int,
	actualSupply *big.Int,
	preJoinExitInvariant *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {

	amountsIn := make([]*big.Int, len(balances))
	for i := range amountsIn {
		amountsIn[i] = new(big.Int)
	}
	amountsIn[indexIn] = amountIn
	bptOut, feeAmountIn, err := calcBptOutGivenExactTokensIn(currentAmp, balances, amountsIn, actualSupply, preJoinExitInvariant, c.Info.SwapFee)
	if err != nil {
		return nil, nil, nil, err
	}
	balances[indexIn].Add(balances[indexIn], amountIn)
	postJoinExitSupply := new(big.Int).Add(actualSupply, bptOut)

	return bptOut, postJoinExitSupply, feeAmountIn, nil
}

func (c *PoolSimulator) _beforeJoinExit(registeredBalances []*big.Int) (*big.Int, []*big.Int, *big.Int, *big.Int, error) {
	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant, err := c._payProtocolFeesBeforeJoinExit(registeredBalances)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	currentAmp := c.AmplificationParameter

	var (
		preJoinExitInvariant *big.Int
	)

	if currentAmp.Cmp(c.LastJoinExit.LastJoinExitAmplification) == 0 {
		preJoinExitInvariant = oldAmpPreJoinExitInvariant
	} else {
		preJoinExitInvariant, err = CalculateInvariant(currentAmp, balances, false)
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, nil

}

/** _payProtocolFeesBeforeJoinExit
 * @dev Calculates due protocol fees originating from accumulated swap fees and yield of non-exempt tokens, pays
 * them by minting BPT, and returns the actual supply and current balances.
 *
 * We also return the current invariant computed using the amplification factor at the last join or exit, which can
 * be useful to skip computations in scenarios where the amplification factor is not changing.
 * Ref: https://github.com/balancer/balancer-v2-monorepo/blob/3251913e63949f35be168b42987d0aae297a01b1/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L64
 */
func (c *PoolSimulator) _payProtocolFeesBeforeJoinExit(
	registeredBalances []*big.Int,
) (*big.Int, []*big.Int, *big.Int, error) {
	virtualSupply, droppedBalances := c._dropBptItemFromBalances(registeredBalances)
	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp, err := c._getProtocolPoolOwnershipPercentage(droppedBalances)
	if err != nil {
		return nil, nil, nil, err
	}
	protocolFeeAmount := c.bptForPoolOwnershipPercentage(virtualSupply, expectedProtocolOwnershipPercentage)

	return new(big.Int).Add(virtualSupply, protocolFeeAmount), droppedBalances, currentInvariantWithLastJoinExitAmp, nil
}

// _getProtocolPoolOwnershipPercentage
// https://github.com/balancer/balancer-v2-monorepo/blob/3251913e63949f35be168b42987d0aae297a01b1/pkg/pool-stable/contracts/ComposableStablePoolProtocolFees.sol#L102
func (c *PoolSimulator) _getProtocolPoolOwnershipPercentage(balances []*big.Int) (*big.Int, *big.Int, error) {
	swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, err := c._getGrowthInvariants(balances)
	if err != nil {
		return nil, nil, err
	}
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
	protocolSwapFeePercentage := MulDownFixed(
		DivDownFixed(swapFeeGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheSwapType)

	protocolYieldPercentage := MulDownFixed(
		DivDownFixed(nonExemptYieldGrowthInvariantDelta, totalGrowthInvariant),
		c.ProtocolFeePercentageCacheYieldType)

	// Calculate the total protocol PoolSimulator ownership percentage
	protocolPoolOwnershipPercentage := new(big.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage)

	return protocolPoolOwnershipPercentage, totalGrowthInvariant, nil
}

func (c *PoolSimulator) _getGrowthInvariants(balances []*big.Int) (*big.Int, *big.Int, *big.Int, error) {
	var (
		swapFeeGrowthInvariant        *big.Int
		totalNonExemptGrowthInvariant *big.Int
		totalGrowthInvariant          *big.Int
		err                           error
	)

	// This invariant result is calc by DivDown (round down)
	// DivDown https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L96
	swapFeeGrowthInvariant, err = CalculateInvariant(
		c.LastJoinExit.LastJoinExitAmplification,
		c.getAdjustedBalances(balances, true), false)
	if err != nil {
		return nil, nil, nil, err
	}

	// For the other invariants, we can potentially skip some work. In the edge cases where none or all of the
	// tokens are exempt from yield, there's one fewer invariant to compute.
	if c._areNoTokensExempt() {
		// If there are no tokens with fee-exempt yield, then the total non-exempt growth will equal the total
		// growth: all yield growth is non-exempt. There's also no point in adjusting balances, since we
		// already know none are exempt.
		totalNonExemptGrowthInvariant, err = CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant = totalNonExemptGrowthInvariant
	} else if c._areAllTokensExempt() {
		// If no tokens are charged fees on yield, then the non-exempt growth is equal to the swap fee growth - no
		// yield fees will be collected.
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
		totalGrowthInvariant, err = CalculateInvariant(c.LastJoinExit.LastJoinExitAmplification, balances, false)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		// In the general case, we need to calculate two invariants: one with some adjusted balances, and one with
		// the current balances.

		totalNonExemptGrowthInvariant, err = CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			c.getAdjustedBalances(balances, false), // Only adjust non-exempt balances
			false,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant, err = CalculateInvariant(
			c.LastJoinExit.LastJoinExitAmplification,
			balances,
			false)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, nil
}
func (c *PoolSimulator) _dropBptItemFromBalances(balances []*big.Int) (*big.Int, []*big.Int) {
	return c._getVirtualSupply(balances[c.BptIndex.Int64()]), c._dropBptItem(balances)
}

func (c *PoolSimulator) _getVirtualSupply(bptBalance *big.Int) *big.Int {
	return new(big.Int).Sub(c.TotalSupply, bptBalance)
}

func (c *PoolSimulator) _hasRateProvider(tokenIndex int) bool {
	if c.RateProviders[tokenIndex] == "" || c.RateProviders[tokenIndex] == valueobject.ZeroAddress {
		return false
	}
	return true
}

func (c *PoolSimulator) isTokenExemptFromYieldProtocolFee(tokenIndex int) bool {
	return c.TokensExemptFromYieldProtocolFee[tokenIndex]
}

func (c *PoolSimulator) _areNoTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if exempt {
			return false
		}
	}
	return true
}

func (c *PoolSimulator) _areAllTokensExempt() bool {
	for _, exempt := range c.TokensExemptFromYieldProtocolFee {
		if !exempt {
			return false
		}
	}
	return true
}

func (c *PoolSimulator) getAdjustedBalances(balances []*big.Int, ignoreExemptFlags bool) []*big.Int {
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
func (c *PoolSimulator) _adjustedBalance(balance *big.Int, cache *TokenRateCache) *big.Int {
	return DivDown(new(big.Int).Mul(balance, cache.OldRate), cache.Rate)
}

// _dropBptItem Remove the item at `_bptIndex` from an arbitrary array (e.g., amountsIn).
func (c *PoolSimulator) _dropBptItem(amounts []*big.Int) []*big.Int {
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
 * @dev Calculates the amount of BPT necessary to give ownership of a given percentage of the PoolSimulator to an external
 * third party. In the case of protocol fees, this is the DAO, but could also be a pool manager, etc.
 * Note that this function reverts if `poolPercentage` >= 100%, it's expected that the caller will enforce this.
 * @param totalSupply - The total supply of the pool prior to minting BPT.
 * @param poolOwnershipPercentage - The desired ownership percentage of the pool to have as a result of minting BPT.
 * @return bptAmount - The amount of BPT to mint such that it is `poolPercentage` of the resultant total supply.
 */
func (c *PoolSimulator) bptForPoolOwnershipPercentage(totalSupply, poolOwnershipPercentage *big.Int) *big.Int {
	// If we mint some amount `bptAmount` of BPT then the percentage ownership of the pool this grants is given by:
	// `poolOwnershipPercentage = bptAmount / (totalSupply + bptAmount)`.
	// Solving for `bptAmount`, we arrive at:
	// `bptAmount = totalSupply * poolOwnershipPercentage / (1 - poolOwnershipPercentage)`.
	return DivDown(new(big.Int).Mul(totalSupply, poolOwnershipPercentage), ComplementFixed(poolOwnershipPercentage))
}

func (c *PoolSimulator) _skipBptIndex(index int) int {
	if index < int(c.BptIndex.Int64()) {
		return index
	} else {
		return index - 1
	}
}

// _subtractSwapFeeAmount
func (c *PoolSimulator) _subtractSwapFeeAmount(amount, _swapFeePercentage *big.Int) (*big.Int, *big.Int) {
	feeAmount := MulUpFixed(amount, _swapFeePercentage)
	return new(big.Int).Sub(amount, feeAmount), feeAmount
}

func (c *PoolSimulator) _upscaleArray(amounts, scalingFactors []*big.Int) []*big.Int {
	result := make([]*big.Int, len(amounts))
	for i, amount := range amounts {
		result[i] = MulUpFixed(amount, scalingFactors[i])
	}
	return result
}

func (c *PoolSimulator) _upscale(amount, scalingFactor *big.Int) *big.Int {
	return MulUpFixed(amount, scalingFactor)
}

// calcBptOutGivenExactTokensIn https://github.com/balancer/balancer-v2-monorepo/blob/b46023f7c5deefaf58a0a42559a36df420e1639f/pkg/pool-stable/contracts/StableMath.sol#L201
func calcBptOutGivenExactTokensIn(amp *big.Int, balances []*big.Int, amountsIn []*big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) (*big.Int, *big.Int, error) {
	feeAmountIn := big.NewInt(0)
	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	balanceRatiosWithFee := make([]*big.Int, len(amountsIn))
	invariantRatioWithFees := big.NewInt(0)
	for i, balance := range balances {
		currentWeight := DivDownFixed(balance, sumBalances)
		balanceRatiosWithFee[i] = DivDownFixed(new(big.Int).Add(balance, amountsIn[i]), balance)
		invariantRatioWithFees.Add(invariantRatioWithFees, MulDownFixed(balanceRatiosWithFee[i], currentWeight))
	}

	newBalances := make([]*big.Int, len(balances))
	for i, balance := range balances {
		var amountInWithoutFee *big.Int
		if balanceRatiosWithFee[i].Cmp(invariantRatioWithFees) > 0 {
			nonTaxableAmount := MulDownFixed(balance, new(big.Int).Sub(invariantRatioWithFees, One))
			taxableAmount := new(big.Int).Sub(amountsIn[i], nonTaxableAmount)
			amountInWithoutFee = new(big.Int).Add(
				nonTaxableAmount,
				MulDownFixed(
					taxableAmount,
					new(big.Int).Sub(One, swapFeePercentage),
				),
			)
		} else {
			amountInWithoutFee = amountsIn[i]
		}
		feeAmountIn = feeAmountIn.Add(feeAmountIn, new(big.Int).Sub(amountsIn[i], amountInWithoutFee))
		newBalances[i] = new(big.Int).Add(balance, amountInWithoutFee)
	}

	newInvariant, err := CalculateInvariant(amp, newBalances, false)
	if err != nil {
		return nil, nil, err
	}

	invariantRatio := DivDownFixed(newInvariant, invariant)
	if invariantRatio.Cmp(One) > 0 {
		return MulDownFixed(bptTotalSupply, new(big.Int).Sub(invariantRatio, One)), feeAmountIn, nil
	} else {
		return big.NewInt(0), feeAmountIn, nil
	}
}

func calcTokenOutGivenExactBptIn(amp *big.Int, balances []*big.Int, tokenIndex int, bptAmountIn *big.Int, bptTotalSupply, invariant, swapFeePercentage *big.Int) (*big.Int, *big.Int, error) {
	newInvariant := MulUpFixed(DivUpFixed(new(big.Int).Sub(bptTotalSupply, bptAmountIn), bptTotalSupply), invariant)
	newBalanceTokenIndex, err := GetTokenBalanceGivenInvariantAndAllOtherBalances(amp, balances, newInvariant, tokenIndex)
	if err != nil {
		return nil, nil, err
	}
	amountOutWithoutFee := new(big.Int).Sub(balances[tokenIndex], newBalanceTokenIndex)

	sumBalances := big.NewInt(0)
	for _, balance := range balances {
		sumBalances.Add(sumBalances, balance)
	}

	currentWeight := DivDownFixed(balances[tokenIndex], sumBalances)
	taxablePercentage := ComplementFixed(currentWeight)

	taxableAmount := MulUpFixed(amountOutWithoutFee, taxablePercentage)
	nonTaxableAmount := new(big.Int).Sub(amountOutWithoutFee, taxableAmount)

	feeOfTaxableAmount := MulDownFixed(
		taxableAmount,
		new(big.Int).Sub(One, swapFeePercentage),
	)

	feeAmount := new(big.Int).Sub(taxableAmount, feeOfTaxableAmount)
	return new(big.Int).Add(nonTaxableAmount, feeOfTaxableAmount), feeAmount, nil
}

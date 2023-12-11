package composablestable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type bptSimulator struct {
	poolpkg.Pool

	bptIndex        int
	bptTotalSupply  *uint256.Int
	amp             *uint256.Int
	scalingFactors  []*uint256.Int
	lastJoinExit    LastJoinExitData
	rateProviders   []string
	tokenRateCaches []TokenRateCache

	swapFeePercentage               *uint256.Int
	protocolFeePercentageCache      map[int]*uint256.Int
	tokenExemptFromYieldProtocolFee []bool
	exemptFromYieldProtocolFee      bool // >= V5
	inRecoveryMode                  bool

	poolTypeVer int
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L301
func (s *bptSimulator) swap(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, *poolpkg.TokenAmount, *SwapInfo, error) {
	balances, err := _upscaleArray(balances, s.scalingFactors)
	if err != nil {
		return nil, nil, nil, err
	}

	amountIn, err = _upscale(amountIn, s.scalingFactors[indexIn])
	if err != nil {
		return nil, nil, nil, err
	}

	preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, err := s._beforeJoinExit(balances)
	if err != nil {
		return nil, nil, nil, err
	}

	var amountCalculated, postJoinExitSupply *uint256.Int
	if indexOut == s.bptIndex {
		amountCalculated, postJoinExitSupply, err = s._doJoinSwap(
			amountIn, balances, _skipBptIndex(indexIn, s.bptIndex), currentAmp, preJoinExitSupply, preJoinExitInvariant,
		)
	} else {
		amountCalculated, postJoinExitSupply, err = s._doExitSwap(
			amountIn, balances, _skipBptIndex(indexOut, s.bptIndex), currentAmp, preJoinExitSupply, preJoinExitInvariant,
		)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	amountOut, err := _downscaleDown(amountCalculated, s.scalingFactors[indexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	swapInfo, err := s.initSwapInfo(
		currentAmp,
		balances,
		preJoinExitInvariant,
		preJoinExitSupply,
		postJoinExitSupply,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return amountOut, &poolpkg.TokenAmount{}, swapInfo, nil
}

func (s *bptSimulator) _doJoinSwap(
	amount *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	currentAmp *uint256.Int,
	actualSupply *uint256.Int,
	preJoinExitInvariant *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	return s._joinSwapExactTokenInForBptOut(
		amount,
		balances,
		indexIn,
		currentAmp,
		actualSupply,
		preJoinExitInvariant,
	)
}

func (s *bptSimulator) _joinSwapExactTokenInForBptOut(
	amount *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	currentAmp *uint256.Int,
	actualSupply *uint256.Int,
	preJoinExitInvariant *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	amountsIn := make([]*uint256.Int, len(balances))
	for i := 0; i < len(balances); i++ {
		amountsIn[i] = uint256.NewInt(0)
	}
	amountsIn[indexIn] = amount

	bptOut, err := math.StableMath.CalcBptOutGivenExactTokensIn(
		currentAmp,
		balances,
		amountsIn,
		actualSupply,
		preJoinExitInvariant,
		s.swapFeePercentage,
	)
	if err != nil {
		return nil, nil, err
	}

	balances[indexIn], err = math.FixedPoint.Add(balances[indexIn], amount)
	if err != nil {
		return nil, nil, err
	}

	postJoinExitSupply, err := math.FixedPoint.Add(actualSupply, bptOut)
	if err != nil {
		return nil, nil, err
	}

	return bptOut, postJoinExitSupply, nil
}

func (s *bptSimulator) _doExitSwap(
	amount *uint256.Int,
	balances []*uint256.Int,
	indexOut int,
	currentAmp *uint256.Int,
	actualSupply *uint256.Int,
	preJoinExitInvariant *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	return s._exitSwapExactBptInForTokenOut(
		amount,
		balances,
		indexOut,
		currentAmp,
		actualSupply,
		preJoinExitInvariant,
	)
}

func (s *bptSimulator) _exitSwapExactBptInForTokenOut(
	bptAmount *uint256.Int,
	balances []*uint256.Int,
	indexOut int,
	currentAmp *uint256.Int,
	actualSupply *uint256.Int,
	preJoinExitInvariant *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	amountOut, err := math.StableMath.CalcTokenOutGivenExactBptIn(
		currentAmp,
		balances,
		indexOut,
		bptAmount,
		actualSupply,
		preJoinExitInvariant,
		s.swapFeePercentage,
	)
	if err != nil {
		return nil, nil, err
	}

	balances[indexOut], err = math.FixedPoint.Sub(balances[indexOut], amountOut)
	if err != nil {
		return nil, nil, err
	}

	postJoinExitSupply, err := math.FixedPoint.Sub(actualSupply, bptAmount)
	if err != nil {
		return nil, nil, err
	}

	return amountOut, postJoinExitSupply, nil
}

func (s *bptSimulator) _getVirtualSupply(bptBalance *uint256.Int) (*uint256.Int, error) {
	cir, err := math.FixedPoint.Sub(s.bptTotalSupply, bptBalance)
	if err != nil {
		return nil, err
	}
	return cir, nil
}

func (s *bptSimulator) _hasRateProvider(tokenIndex int) bool {
	if s.rateProviders[tokenIndex] == "" || s.rateProviders[tokenIndex] == valueobject.ZeroAddress {
		return false
	}
	return true
}

func (s *bptSimulator) _beforeJoinExit(
	registeredBalances []*uint256.Int,
) (*uint256.Int, []*uint256.Int, *uint256.Int, *uint256.Int, error) {
	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant, err := s._payProtocolFeesBeforeJoinExit(registeredBalances)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var preJoinExitInvariant *uint256.Int
	if s.amp.Eq(s.lastJoinExit.LastJoinExitAmplification) {
		preJoinExitInvariant = oldAmpPreJoinExitInvariant
	} else {
		preJoinExitInvariant, err = math.StableMath.CalculateInvariantV2(
			s.amp,
			balances,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return preJoinExitSupply, balances, s.amp, preJoinExitInvariant, nil
}

func (s *bptSimulator) _payProtocolFeesBeforeJoinExit(
	registeredBalances []*uint256.Int,
) (*uint256.Int, []*uint256.Int, *uint256.Int, error) {
	virtualSupply, balances, err := s._dropBptItemFromBalances(registeredBalances)
	if err != nil {
		return nil, nil, nil, err
	}

	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp, err := s._getProtocolPoolOwnershipPercentage(balances)
	if err != nil {
		return nil, nil, nil, err
	}

	protocolFeeAmount, err := s.protocolFeeAmount(virtualSupply, expectedProtocolOwnershipPercentage)
	if err != nil {
		return nil, nil, nil, err
	}

	return new(uint256.Int).Add(virtualSupply, protocolFeeAmount),
		balances,
		currentInvariantWithLastJoinExitAmp,
		nil
}

func (s *bptSimulator) _getProtocolPoolOwnershipPercentage(
	balances []*uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer5 {
		return s._getProtocolPoolOwnershipPercentageV2(balances)
	}
	return s._getProtocolPoolOwnershipPercentageV1(balances)
}

func (s *bptSimulator) _getProtocolPoolOwnershipPercentageV2(
	balances []*uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, err := s._getGrowthInvariantsV2(balances)
	if err != nil {
		return nil, nil, err
	}

	if totalGrowthInvariant.Cmp(s.lastJoinExit.LastPostJoinExitInvariant) <= 0 {
		return uint256.NewInt(0), totalGrowthInvariant, nil
	}

	swapFeeGrowthInvariantDelta := new(uint256.Int).Sub(
		swapFeeGrowthInvariant, s.lastJoinExit.LastPostJoinExitInvariant,
	)

	nonExemptYieldGrowthInvariantDelta := new(uint256.Int).Sub(
		totalNonExemptGrowthInvariant, swapFeeGrowthInvariant,
	)

	var protocolSwapFeePercentage *uint256.Int
	{
		percentage := s.getProtocolFeePercentageCache(feeTypeSwap)
		u, err := math.FixedPoint.DivDown(swapFeeGrowthInvariantDelta, totalGrowthInvariant)
		if err != nil {
			return nil, nil, err
		}
		u, err = math.FixedPoint.MulDown(u, percentage)
		if err != nil {
			return nil, nil, err
		}

		protocolSwapFeePercentage = u
	}

	var protocolYieldPercentage *uint256.Int
	{
		percentage := s.getProtocolFeePercentageCache(feeTypeYield)
		u, err := math.FixedPoint.DivDown(
			nonExemptYieldGrowthInvariantDelta,
			totalGrowthInvariant,
		)
		if err != nil {
			return nil, nil, err
		}

		u, err = math.FixedPoint.MulDown(u, percentage)
		if err != nil {
			return nil, nil, err
		}

		protocolYieldPercentage = u
	}

	return new(uint256.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage), totalGrowthInvariant, nil
}

func (s *bptSimulator) _getProtocolPoolOwnershipPercentageV1(
	balances []*uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, err := s._getGrowthInvariantsV1(balances)
	if err != nil {
		return nil, nil, err
	}

	swapFeeGrowthInvariantDelta := uint256.NewInt(0)
	if swapFeeGrowthInvariant.Gt(s.lastJoinExit.LastPostJoinExitInvariant) {
		swapFeeGrowthInvariantDelta = new(uint256.Int).Sub(
			swapFeeGrowthInvariant, s.lastJoinExit.LastPostJoinExitInvariant,
		)
	}

	nonExemptYieldGrowthInvariantDelta := uint256.NewInt(0)
	if totalNonExemptGrowthInvariant.Gt(swapFeeGrowthInvariant) {
		nonExemptYieldGrowthInvariantDelta = new(uint256.Int).Sub(
			totalNonExemptGrowthInvariant, swapFeeGrowthInvariant,
		)
	}

	var protocolSwapFeePercentage *uint256.Int
	{
		percentage := s.getProtocolFeePercentageCache(feeTypeSwap)
		u, err := math.FixedPoint.DivDown(swapFeeGrowthInvariantDelta, totalGrowthInvariant)
		if err != nil {
			return nil, nil, err
		}
		u, err = math.FixedPoint.MulDown(u, percentage)
		if err != nil {
			return nil, nil, err
		}

		protocolSwapFeePercentage = u
	}

	var protocolYieldPercentage *uint256.Int
	{
		percentage := s.getProtocolFeePercentageCache(feeTypeYield)
		u, err := math.FixedPoint.DivDown(
			nonExemptYieldGrowthInvariantDelta,
			totalGrowthInvariant,
		)
		if err != nil {
			return nil, nil, err
		}

		u, err = math.FixedPoint.MulDown(u, percentage)
		if err != nil {
			return nil, nil, err
		}

		protocolYieldPercentage = u
	}

	return new(uint256.Int).Add(protocolSwapFeePercentage, protocolYieldPercentage), totalGrowthInvariant, nil
}

func (s *bptSimulator) _isTokenExemptFromYieldProtocolFee(registeredTokenIndex int) bool {
	return s.tokenExemptFromYieldProtocolFee[registeredTokenIndex]
}

func (s *bptSimulator) _getGrowthInvariantsV1(
	balances []*uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	var (
		swapFeeGrowthInvariant        *uint256.Int
		totalNonExemptGrowthInvariant *uint256.Int
		totalGrowthInvariant          *uint256.Int
		err                           error
	)

	adjustedBalances, err := s._getAdjustedBalanceV1(balances, true)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFeeGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
		s.lastJoinExit.LastJoinExitAmplification,
		adjustedBalances,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	if s._areNoTokensExempt() {
		totalNonExemptGrowthInvariant, err := math.StableMath.CalculateInvariantV2(
			s.lastJoinExit.LastJoinExitAmplification,
			balances,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant = totalNonExemptGrowthInvariant
	} else if s._areAllTokensExempt() {
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
		totalGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
			s.lastJoinExit.LastJoinExitAmplification, balances,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		adjustedBalances, err := s._getAdjustedBalanceV1(balances, false)
		if err != nil {
			return nil, nil, nil, err
		}

		totalNonExemptGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
			s.lastJoinExit.LastJoinExitAmplification,
			adjustedBalances,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		totalGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
			s.lastJoinExit.LastJoinExitAmplification,
			balances,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, nil
}

func (s *bptSimulator) _getAdjustedBalanceV1(
	balances []*uint256.Int,
	ignoreExemptFlags bool,
) ([]*uint256.Int, error) {
	totalTokensWithoutBpt := len(balances)
	adjustedBalances := make([]*uint256.Int, totalTokensWithoutBpt)

	for i := 0; i < totalTokensWithoutBpt; i++ {
		skipBptIndex := i
		if i >= s.bptIndex {
			skipBptIndex++
		}

		if s._isTokenExemptFromYieldProtocolFee(skipBptIndex) ||
			(ignoreExemptFlags && s._hasRateProvider(skipBptIndex)) {
			var err error
			adjustedBalances[i], err = _adjustedBalance(balances[i], s.tokenRateCaches[skipBptIndex])
			if err != nil {
				return nil, err
			}

			continue
		}

		adjustedBalances[i] = balances[i]
	}

	return adjustedBalances, nil
}

func (s *bptSimulator) _getGrowthInvariantsV2(
	balances []*uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	var (
		swapFeeGrowthInvariant        *uint256.Int
		totalNonExemptGrowthInvariant *uint256.Int
		totalGrowthInvariant          *uint256.Int
		err                           error
	)

	totalGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
		s.lastJoinExit.LastJoinExitAmplification,
		balances,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	if totalGrowthInvariant.Cmp(s.lastJoinExit.LastPostJoinExitInvariant) <= 0 {
		return totalGrowthInvariant, totalGrowthInvariant, totalGrowthInvariant, nil
	}

	adjustedBalances, err := s._getAdjustedBalanceV2(balances)
	if err != nil {
		return nil, nil, nil, err
	}
	swapFeeGrowthInvariant, err = math.StableMath.CalculateInvariantV2(
		s.lastJoinExit.LastJoinExitAmplification,
		adjustedBalances,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFeeGrowthInvariant = math.Math.Min(totalGrowthInvariant, swapFeeGrowthInvariant)
	swapFeeGrowthInvariant = math.Math.Max(s.lastJoinExit.LastPostJoinExitInvariant, swapFeeGrowthInvariant)

	if s.isExemptFromYieldProtocolFee() {
		totalNonExemptGrowthInvariant = swapFeeGrowthInvariant
	} else {
		totalNonExemptGrowthInvariant = totalGrowthInvariant
	}

	return swapFeeGrowthInvariant, totalNonExemptGrowthInvariant, totalGrowthInvariant, nil
}

func (s *bptSimulator) _getAdjustedBalanceV2(balances []*uint256.Int) ([]*uint256.Int, error) {
	totalTokensWithoutBpt := len(balances)
	adjustedBalances := make([]*uint256.Int, totalTokensWithoutBpt)

	for i := 0; i < totalTokensWithoutBpt; i++ {
		skipBptIndex := i
		if i >= s.bptIndex {
			skipBptIndex++
		}

		if s._hasRateProvider(skipBptIndex) {
			var err error
			adjustedBalances[i], err = _adjustedBalance(balances[i], s.tokenRateCaches[skipBptIndex])
			if err != nil {
				return nil, err
			}
			continue
		}

		adjustedBalances[i] = balances[i]
	}

	return adjustedBalances, nil
}

func (s *bptSimulator) isExemptFromYieldProtocolFee() bool {
	return s.exemptFromYieldProtocolFee
}

func (s *bptSimulator) _areAllTokensExempt() bool {
	for _, exempt := range s.tokenExemptFromYieldProtocolFee {
		if !exempt {
			return false
		}
	}
	return true
}

func (s *bptSimulator) _areNoTokensExempt() bool {
	for _, exempt := range s.tokenExemptFromYieldProtocolFee {
		if exempt {
			return false
		}
	}
	return true
}

func (s *bptSimulator) getProtocolFeePercentageCache(feeType int) *uint256.Int {
	if s.inRecoveryMode {
		return uint256.NewInt(0)
	}

	return s.protocolFeePercentageCache[feeType]
}

func (s *bptSimulator) protocolFeeAmount(
	totalSupply *uint256.Int,
	poolOwnershipPercentage *uint256.Int,
) (*uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer1 {
		return s._calculateAdjustedProtocolFeeAmount(totalSupply, poolOwnershipPercentage)
	}

	return s.bptForPoolOwnershipPercentage(totalSupply, poolOwnershipPercentage)
}

func (s *bptSimulator) _calculateAdjustedProtocolFeeAmount(
	totalSupply *uint256.Int,
	poolOwnershipPercentage *uint256.Int,
) (*uint256.Int, error) {
	u, err := math.FixedPoint.MulDown(totalSupply, poolOwnershipPercentage)
	if err != nil {
		return nil, err
	}

	u, err = math.FixedPoint.DivDown(u, math.FixedPoint.Complement(poolOwnershipPercentage))
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *bptSimulator) bptForPoolOwnershipPercentage(
	totalSupply *uint256.Int,
	poolOwnershipPercentage *uint256.Int,
) (*uint256.Int, error) {
	u, err := math.Math.Mul(totalSupply, poolOwnershipPercentage)
	if err != nil {
		return nil, err
	}
	u, err = math.Math.DivDown(u, math.FixedPoint.Complement(poolOwnershipPercentage))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *bptSimulator) _dropBptItemFromBalances(
	registeredBalances []*uint256.Int,
) (*uint256.Int, []*uint256.Int, error) {
	virtualSupply, err := s._getVirtualSupply(registeredBalances[s.bptIndex])
	if err != nil {
		return nil, nil, err
	}

	balances := _dropBptItem(registeredBalances, s.bptIndex)

	return virtualSupply, balances, nil
}

func (s *bptSimulator) initSwapInfo(
	currentAmp *uint256.Int,
	balances []*uint256.Int,
	preJoinExitInvariant *uint256.Int,
	preJoinExitSupply *uint256.Int,
	postJoinExitSupply *uint256.Int,
) (*SwapInfo, error) {
	postJoinExitInvariant, err := math.StableMath.CalculateInvariantV2(currentAmp, balances)
	if err != nil {
		return nil, err
	}

	swapInfo := &SwapInfo{
		LastJoinExitData: LastJoinExitData{
			LastJoinExitAmplification: currentAmp,
			LastPostJoinExitInvariant: postJoinExitInvariant,
		},
	}

	return swapInfo, nil
}

func (s *bptSimulator) updateBalance(params poolpkg.UpdateBalanceParams) {
	for idx, token := range s.Info.Tokens {
		// update reserves

		if token == params.TokenAmountIn.Token {
			s.Info.Reserves[idx] = new(big.Int).Add(
				s.Info.Reserves[idx],
				params.TokenAmountIn.Amount,
			)
		}

		if token == params.TokenAmountOut.Token {
			s.Info.Reserves[idx] = new(big.Int).Sub(
				s.Info.Reserves[idx],
				params.TokenAmountOut.Amount,
			)
		}

		// update rates

		if s._hasRateProvider(idx) {
			s.tokenRateCaches[idx].OldRate = s.tokenRateCaches[idx].Rate
		}
	}

	swapInfo, ok := params.SwapInfo.(*SwapInfo)
	if !ok {
		return
	}
	s.lastJoinExit = swapInfo.LastJoinExitData
}

func _adjustedBalance(balance *uint256.Int, cache TokenRateCache) (*uint256.Int, error) {
	u, err := math.Math.Mul(balance, cache.OldRate)
	if err != nil {
		return nil, err
	}
	return math.Math.DivDown(u, cache.Rate)
}

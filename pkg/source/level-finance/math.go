package levelfinance

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
)

func swap(tokenIn, tokenOut string, amountIn *big.Int, state *PoolState) (*big.Int, error) {
	// Check allowSwap

	if tokenIn == tokenOut {
		return nil, ErrSameTokenSwap
	}
	if amountIn.Cmp(constants.Zero) == 0 {
		return nil, ErrZeroAmount
	}

	tokenInInfo, ok := state.TokenInfos[tokenIn]
	if !ok {
		return nil, ErrTokenInfoIsNotFound
	}
	tokenOutInfo, ok := state.TokenInfos[tokenOut]
	if !ok {
		return nil, ErrTokenInfoIsNotFound
	}

	amountOutAfterFee, swapFee, err := calcSwapOutput(tokenInInfo, tokenOutInfo, amountIn, state)
	if err != nil {
		return nil, err
	}

	daoFee := calcDaoFee(swapFee, state)

	rebalanceTranches(tokenInInfo, new(big.Int).Sub(amountIn, daoFee), tokenOutInfo, amountOutAfterFee, state)

	// _validateMaxLiquidity

	return amountOutAfterFee, nil
}

func calcSwapOutput(tokenIn, tokenOut *TokenInfo, amountIn *big.Int, state *PoolState) (*big.Int, *big.Int, error) {
	priceIn := new(big.Int).Set(tokenIn.MinPrice)
	priceOut := new(big.Int).Set(tokenOut.MaxPrice)
	valueChange := new(big.Int).Mul(amountIn, priceIn)
	isStableSwap := tokenIn.IsStableCoin && tokenOut.IsStableCoin
	feeIn := calcSwapFee(isStableSwap, tokenIn, priceIn, valueChange, true, state)
	feeOut := calcSwapFee(isStableSwap, tokenOut, priceOut, valueChange, false, state)

	fee := feeIn
	if feeIn.Cmp(feeOut) <= 0 {
		fee = feeOut
	}

	amountOutAfterFee := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(valueChange, new(big.Int).Sub(precision, fee)),
			priceOut,
		),
		precision,
	)
	feeAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(valueChange, fee),
			priceIn,
		),
		precision,
	)

	return amountOutAfterFee, feeAmount, nil
}

func calcSwapFee(isStableSwap bool, token *TokenInfo, tokenPrice, valueChange *big.Int, isSwapIn bool, state *PoolState) *big.Int {
	var baseSwapFee, taxBasicPoint *big.Int
	if isStableSwap {
		baseSwapFee = new(big.Int).Set(state.StableCoinBaseSwapFee)
		taxBasicPoint = new(big.Int).Set(state.StableCoinTaxBasisPoint)
	} else {
		baseSwapFee = new(big.Int).Set(state.BaseSwapFee)
		taxBasicPoint = new(big.Int).Set(state.TaxBasisPoint)
	}

	return calcFeeRate(token, tokenPrice, valueChange, baseSwapFee, taxBasicPoint, isSwapIn, state)
}

func calcFeeRate(token *TokenInfo, tokenPrice, valueChange, baseFee, taxBasicPoint *big.Int, isIncrease bool, state *PoolState) *big.Int {
	var targetValue *big.Int
	if state.TotalWeight.Cmp(constants.Zero) == 0 {
		targetValue = big.NewInt(0)
	} else {
		targetValue = new(big.Int).Div(
			new(big.Int).Mul(token.TargetWeight, state.VirtualPoolValue),
			state.TotalWeight,
		)
	}
	if targetValue.Cmp(constants.Zero) == 0 {
		return baseFee
	}

	currentValue := new(big.Int).Mul(
		tokenPrice,
		getPoolAsset(token).PoolAmount,
	)

	var nextValue *big.Int
	if isIncrease {
		nextValue = new(big.Int).Add(currentValue, valueChange)
	} else {
		nextValue = new(big.Int).Sub(currentValue, valueChange)
	}
	initDiff := diff(currentValue, targetValue)
	nextDiff := diff(nextValue, targetValue)

	if nextDiff.Cmp(initDiff) < 0 {
		feeAdjust := new(big.Int).Div(
			new(big.Int).Mul(taxBasicPoint, initDiff),
			targetValue,
		)
		rate := zeroCapSub(baseFee, feeAdjust)
		if rate.Cmp(minSwapFee) > 0 {
			return rate
		}
		return new(big.Int).Set(minSwapFee)
	} else {
		avgDiff := new(big.Int).Div(
			new(big.Int).Add(initDiff, nextDiff),
			bignumber.Two,
		)
		feeAdjust := new(big.Int).Set(taxBasicPoint)
		if avgDiff.Cmp(targetValue) <= 0 {
			feeAdjust = new(big.Int).Div(
				new(big.Int).Mul(taxBasicPoint, avgDiff),
				targetValue,
			)
		}
		return new(big.Int).Add(baseFee, feeAdjust)
	}
}

func getPoolAsset(token *TokenInfo) *AssetInfo {
	asset := &AssetInfo{
		PoolAmount:    big.NewInt(0),
		ReserveAmount: big.NewInt(0),
	}
	for _, tranche := range token.TrancheAssets {
		asset.PoolAmount = new(big.Int).Add(asset.PoolAmount, tranche.PoolAmount)
		asset.ReserveAmount = new(big.Int).Add(asset.ReserveAmount, tranche.ReserveAmount)
	}
	return asset
}

func calcDaoFee(feeAmount *big.Int, state *PoolState) *big.Int {
	return frac(feeAmount, state.DaoFee, precision)
}

func rebalanceTranches(tokenIn *TokenInfo, amountIn *big.Int, tokenOut *TokenInfo, amountOut *big.Int, state *PoolState) {
	outAmounts := calcTrancheSharesAmount(tokenIn, tokenOut, amountOut, false, state)
	for trancheAddress := range tokenIn.TrancheAssets {
		tokenOut.TrancheAssets[trancheAddress].PoolAmount = new(big.Int).Sub(
			tokenOut.TrancheAssets[trancheAddress].PoolAmount,
			outAmounts[trancheAddress],
		)
		tokenIn.TrancheAssets[trancheAddress].PoolAmount = new(big.Int).Add(
			tokenIn.TrancheAssets[trancheAddress].PoolAmount,
			frac(amountIn, outAmounts[trancheAddress], amountOut),
		)
	}
}

func calcTrancheSharesAmount(indexToken, collateralToken *TokenInfo, amount *big.Int, isInceasePoolAmount bool, _ *PoolState) map[string]*big.Int {
	nTranches := len(indexToken.TrancheAssets)
	reserves := make(map[string]*big.Int, len(indexToken.TrancheAssets))
	factors := make(map[string]*big.Int, len(indexToken.TrancheAssets))
	maxShare := make(map[string]*big.Int, len(indexToken.TrancheAssets))

	for trancheAddress := range indexToken.TrancheAssets {
		reserves[trancheAddress] = big.NewInt(0)
		asset := collateralToken.TrancheAssets[trancheAddress]
		factors[trancheAddress] = big.NewInt(1)
		if !indexToken.IsStableCoin {
			factors[trancheAddress] = indexToken.RiskFactor[trancheAddress]
		}
		maxShare[trancheAddress] = new(big.Int).Set(abi.MaxUint256)
		if !isInceasePoolAmount {
			maxShare[trancheAddress] = new(big.Int).Sub(asset.PoolAmount, asset.ReserveAmount)
		}
	}

	var totalFactor *big.Int
	if indexToken.IsStableCoin {
		totalFactor = big.NewInt(int64(nTranches))
	} else {
		totalFactor = new(big.Int).Set(indexToken.TotalRiskFactor)
	}

	newAmount := new(big.Int).Set(amount)
	for range indexToken.TrancheAssets {
		totalRiskFactorTmp := new(big.Int).Set(totalFactor)
		for i := range indexToken.TrancheAssets {
			riskFactorTmp := new(big.Int).Set(factors[i])
			if riskFactorTmp.Cmp(constants.Zero) != 0 {
				shareAmount := frac(amount, riskFactorTmp, totalRiskFactorTmp)
				availableAmount := new(big.Int).Sub(maxShare[i], reserves[i])
				if shareAmount.Cmp(availableAmount) >= 0 {
					shareAmount = availableAmount
					totalFactor = new(big.Int).Sub(totalFactor, riskFactorTmp)
					factors[i] = big.NewInt(0)
				}

				reserves[i] = new(big.Int).Add(reserves[i], shareAmount)
				newAmount = new(big.Int).Sub(newAmount, shareAmount)
				totalRiskFactorTmp = new(big.Int).Sub(totalRiskFactorTmp, riskFactorTmp)
				if newAmount.Cmp(constants.Zero) == 0 {
					return reserves
				}
			}
		}
	}

	return reserves
}

func diff(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Sub(a, b)
	}
	return new(big.Int).Sub(b, a)
}

func zeroCapSub(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Sub(a, b)
	}
	return big.NewInt(0)
}

func frac(amount, num, denom *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(amount, num),
		denom,
	)
}

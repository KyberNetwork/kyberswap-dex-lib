package levelfinance

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
)

func swap(tokenIn, tokenOut string, amountIn *big.Int, state *PoolState) (*big.Int, error) {
	if tokenIn == tokenOut {
		return nil, ErrSameTokenSwap
	}
	if amountIn.Cmp(constants.Zero) == 0 {
		return nil, ErrZeroAmount
	}

	tokenInInfo, ok := state.TokenInfos[tokenIn]
	if !ok {
		return nil, ErrTokenNotFound
	}
	tokenOutInfo, ok := state.TokenInfos[tokenOut]
	if !ok {
		return nil, ErrTokenNotFound
	}

	amountOutAfterFee, swapFee, err := calcSwapOutput(tokenInInfo, tokenOutInfo, amountIn, state)
	if err != nil {
		return nil, err
	}

	daoFee := calcDaoFee(swapFee, state)

	rebalanceTranches(tokenIn, new(big.Int).Sub(amountIn, daoFee), tokenOut, amountOutAfterFee, state)

	return amountOutAfterFee, nil
}

func calcSwapOutput(tokenIn, tokenOut *TokenInfo, amountIn *big.Int, state *PoolState) (*big.Int, *big.Int, error) {
	priceIn := new(big.Int).Set(tokenIn.MinPrice)
	priceOut := new(big.Int).Set(tokenOut.MaxPrice)
	valueChange := new(big.Int).Mul(amountIn, priceIn)
	feeIn := calcSwapFee(tokenIn, priceIn, valueChange, true, state)
	feeOut := calcSwapFee(tokenOut, priceOut, valueChange, false, state)

	fee := feeIn
	if feeIn.Cmp(feeOut) <= 0 {
		fee = feeOut
	}

	amountOutAfterFee := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(valueChange, new(big.Int).Sub(Precision, fee)),
			priceOut,
		),
		Precision,
	)
	feeAmount := new(big.Int).Div(
		new(big.Int).Div(
			new(big.Int).Mul(valueChange, fee),
			priceIn,
		),
		Precision,
	)

	return amountOutAfterFee, feeAmount, nil
}

func calcSwapFee(token *TokenInfo, tokenPrice, valueChange *big.Int, isSwapIn bool, state *PoolState) *big.Int {
	var baseSwapFee, taxBasicPoint *big.Int
	if token.IsStableCoin {
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
		return zeroCapSub(baseFee, feeAdjust)
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
	return frac(feeAmount, state.DaoFee, Precision)
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

func calcTrancheSharesAmount(indexToken, collateralToken *TokenInfo, amount *big.Int, isInceasePoolAmount bool, state *PoolState) map[string]*big.Int {
	resserves := make(map[string]*big.Int, len(indexToken.TrancheAssets))
	factors := make(map[string]*big.Int, len(indexToken.TrancheAssets))
	maxShare := make(map[string]*big.Int, len(indexToken.TrancheAssets))

	for trancheAddress := range indexToken.TrancheAssets {
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

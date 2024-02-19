package hillclimb

import (
	"context"
	"fmt"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/huandu/go-clone"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *hillClimbFinder) optimizeRoute(ctx context.Context, input findroute.Input, data findroute.FinderData, baseRoute *valueobject.Route) (*valueobject.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "hillClimbFinder.optimizeRoute")
	defer span.End()

	var (
		// tmpRoute is a temporary route only meant to call `tmpRoute.AddPath`
		//	to update the balance of pools
		// 	we want to update the balance of pools while optimizing the baseRoute
		//  such that we can more precisely calculate the rate of later paths
		tmpRoute = valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)

		// clone the original baseRoute so that we do not modify it
		currentRoute = clone.Slowly(baseRoute).(*valueobject.Route)
		err          error
	)

	// sequentially adjust the distribution of paths
	for pathId := 0; pathId < len(currentRoute.Paths)-1; pathId++ {
		// adjust the distribution percent of two consecutive paths
		// for example, adjust [path1: 5% , path2: 10%] to [path1: 7% , path2: 8%]
		currentRoute.Paths[pathId], currentRoute.Paths[pathId+1], err =
			f.binarySearch(input, data, currentRoute.Paths[pathId], currentRoute.Paths[pathId+1])

		if err != nil {
			return nil, err
		}

		if err = tmpRoute.AddPath(data.PoolBucket, currentRoute.Paths[pathId], data.SwapLimits); err != nil {
			logger.WithFields(ctx, logger.Fields{"error": err}).
				Warnf("cannot optimize path from token %v to token %v", input.TokenInAddress, input.TokenOutAddress)
			return currentRoute, nil
		}
	}
	return currentRoute, nil
}

func (f *hillClimbFinder) binarySearch(
	input findroute.Input, data findroute.FinderData, baseFirstPath, baseSecondPath *valueobject.Path,
) (*valueobject.Path, *valueobject.Path, error) {
	var (
		bestFirstPath, bestSecondPath, firstPath, secondPath *valueobject.Path
		bestTokenAmountOut                                   *poolpkg.TokenAmount

		low  = -int(f.hillClimbIteration)
		high = int(f.hillClimbIteration) - 1

		tokenAmountOutResult = make(map[int]*poolpkg.TokenAmount)
	)

	for low <= high {
		mid := (low + high) / 2

		// calculate the token amount out if we move `mid * f.distributionPercent`% of amountIn
		if _, ok := tokenAmountOutResult[mid]; !ok {
			tokenAmountOutResult[mid], firstPath, secondPath = f.calcAdjustedTokenAmount(input, data, baseFirstPath, baseSecondPath, mid)
			// if better than the best found way to adjust the distribution
			if cmpTokenAmount(tokenAmountOutResult[mid], bestTokenAmountOut, input.GasInclude) == 1 {
				bestTokenAmountOut = tokenAmountOutResult[mid]
				bestFirstPath = firstPath
				bestSecondPath = secondPath
			}
		}

		// calculate the token amount out if we move `(mid + 1) * f.distributionPercent`% of amountIn
		if _, ok := tokenAmountOutResult[mid+1]; !ok {
			tokenAmountOutResult[mid+1], firstPath, secondPath = f.calcAdjustedTokenAmount(input, data, baseFirstPath, baseSecondPath, mid+1)
			// if better than the best found way to adjust the distribution
			if cmpTokenAmount(tokenAmountOutResult[mid+1], bestTokenAmountOut, input.GasInclude) == 1 {
				bestTokenAmountOut = tokenAmountOutResult[mid+1]
				bestFirstPath = firstPath
				bestSecondPath = secondPath
			}
		}

		// if tokenAmountOutResult[mid] > tokenAmountOutResult[mid+1]
		if cmpTokenAmount(tokenAmountOutResult[mid], tokenAmountOutResult[mid+1], input.GasInclude) == 1 {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	// if no valid way to adjust the distribution (including keep the original distribution)
	if bestFirstPath == nil || bestSecondPath == nil {
		return nil, nil, fmt.Errorf("could not find any valid distribution between two paths")
	}
	return bestFirstPath, bestSecondPath, nil
}

// return the output amount and the adjusted paths
//
//	move `splits * f.distributionPercent`% of amountIn from secondPath to firstPath
//	note that `splits` can be negative
func (f *hillClimbFinder) calcAdjustedTokenAmount(
	input findroute.Input, data findroute.FinderData,
	baseFirstPath, baseSecondPath *valueobject.Path,
	splits int,
) (*poolpkg.TokenAmount, *valueobject.Path, *valueobject.Path) {
	// Step 1: calculate the adjust input for the two paths

	var (
		tokenInPriceUSD = data.PriceUSDByAddress[input.TokenInAddress]
		tokenInDecimal  = data.TokenByAddress[input.TokenInAddress].Decimals

		amountInBigInt = input.AmountIn
		maxNumSplits   = int64(constant.OneHundredPercent / f.distributionPercent)

		amountInPerSplit    = new(big.Int).Div(amountInBigInt, big.NewInt(maxNumSplits))
		amountInPerSplitUsd = utils.CalcTokenAmountUsd(amountInPerSplit, tokenInDecimal, tokenInPriceUSD)

		firstPathInput = poolpkg.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    new(big.Int).Add(baseFirstPath.Input.Amount, new(big.Int).Mul(big.NewInt(int64(splits)), amountInPerSplit)),
			AmountUsd: baseFirstPath.Input.AmountUsd + float64(splits)*amountInPerSplitUsd,
		}
		secondPathInput = poolpkg.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    new(big.Int).Sub(baseSecondPath.Input.Amount, new(big.Int).Mul(big.NewInt(int64(splits)), amountInPerSplit)),
			AmountUsd: baseSecondPath.Input.AmountUsd - float64(splits)*amountInPerSplitUsd,
		}
	)

	// if any path does not satisfy minPartUSD condition, return amount out = 0
	if firstPathInput.AmountUsd < f.minPartUSD || secondPathInput.AmountUsd < f.minPartUSD {
		return nil, nil, nil
	}

	// Step 2: recalculate the rate of two paths with the new input

	var (
		tokenOutPriceUSD = data.PriceUSDByAddress[input.TokenOutAddress]
		tokenOutDecimal  = data.TokenByAddress[input.TokenOutAddress].Decimals
		gasOption        = valueobject.GasOption{
			TokenPrice:    input.GasTokenPriceUSD,
			Price:         input.GasPrice,
			GasFeeInclude: input.GasInclude,
		}
	)

	firstPathAdjusted, err := valueobject.NewPath(data.PoolBucket, baseFirstPath.PoolAddresses, baseFirstPath.Tokens,
		firstPathInput, input.TokenOutAddress, tokenOutPriceUSD, tokenOutDecimal, gasOption, data.SwapLimits)

	if err != nil {
		return nil, nil, nil
	}

	secondPathAdjusted, err := valueobject.NewPath(data.PoolBucket, baseSecondPath.PoolAddresses, baseSecondPath.Tokens, secondPathInput,
		input.TokenOutAddress, tokenOutPriceUSD, tokenOutDecimal, gasOption, data.SwapLimits)

	if err != nil {
		return nil, nil, nil
	}

	return &poolpkg.TokenAmount{
		Token:     input.TokenOutAddress,
		Amount:    new(big.Int).Add(firstPathAdjusted.Output.Amount, secondPathAdjusted.Output.Amount),
		AmountUsd: firstPathAdjusted.Output.AmountUsd + secondPathAdjusted.Output.AmountUsd,
	}, firstPathAdjusted, secondPathAdjusted
}

// return 1 if a greater than b
// return 0 if a == b
// return -1 otherwise
func cmpTokenAmount(a, b *poolpkg.TokenAmount, gasFeeInclude bool) int {
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if gasFeeInclude && !utils.Float64AlmostEqual(a.AmountUsd, b.AmountUsd) {
		if a.AmountUsd > b.AmountUsd {
			return 1
		} else {
			return -1
		}
	}
	// Otherwise, prioritize node with more token Amount
	return a.Amount.Cmp(b.Amount)
}

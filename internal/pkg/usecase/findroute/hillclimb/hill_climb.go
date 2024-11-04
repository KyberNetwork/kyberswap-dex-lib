package hillclimb

import (
	"context"
	"fmt"
	"math/big"

	clone "github.com/huandu/go-clone/generic"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *HillClimbFinder) optimizeRoute(ctx context.Context, input findroute.Input, data findroute.FinderData, baseRoute *valueobject.Route) (*valueobject.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "hillClimbFinder.optimizeRoute")
	defer span.End()

	var (
		// tmpRoute is a temporary route only meant to call `tmpRoute.AddPath`
		//	to update the balance of pools
		// 	we want to update the balance of pools while optimizing the baseRoute
		//  such that we can more precisely calculate the rate of later paths
		tmpRoute = valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)

		// clone the original baseRoute so that we do not modify it
		currentRoute = clone.Slowly(baseRoute)
		err          error
	)

	// sequentially adjust the distribution of paths
	for pathId := 0; pathId < len(currentRoute.Paths)-1; pathId++ {
		// adjust the distribution percent of two consecutive paths
		// for example, adjust [path1: 5% , path2: 10%] to [path1: 7% , path2: 8%]
		currentRoute.Paths[pathId], currentRoute.Paths[pathId+1], err =
			f.binarySearch(ctx, input, data, currentRoute.Paths[pathId], currentRoute.Paths[pathId+1])

		if err != nil {
			return nil, err
		}

		if err = tmpRoute.AddPath(ctx, data.PoolBucket, currentRoute.Paths[pathId], data.SwapLimits); err != nil {
			logger.WithFields(ctx, logger.Fields{"error": err}).
				Debugf("cannot optimize path from token %v to token %v", input.TokenInAddress, input.TokenOutAddress)
			return currentRoute, nil
		}
	}
	return currentRoute, nil
}

func (f *HillClimbFinder) binarySearch(
	ctx context.Context, input findroute.Input, data findroute.FinderData, baseFirstPath, baseSecondPath *valueobject.Path,
) (*valueobject.Path, *valueobject.Path, error) {
	var (
		bestFirstPath, bestSecondPath, firstPath, secondPath *valueobject.Path
		bestTokenAmountOut                                   *valueobject.TokenAmount

		low  = -int(f.hillClimbIteration)
		high = int(f.hillClimbIteration) - 1

		tokenAmountOutResult = make(map[int]*valueobject.TokenAmount)
	)

	for low <= high {
		mid := (low + high) / 2

		// calculate the token amount out if we move `mid * f.distributionPercent`% of amountIn
		if _, ok := tokenAmountOutResult[mid]; !ok {
			tokenAmountOutResult[mid], firstPath, secondPath = f.calcAdjustedTokenAmount(ctx, input, data, baseFirstPath, baseSecondPath, mid)
			// if better than the best found way to adjust the distribution
			if tokenAmountOutResult[mid].Compare(bestTokenAmountOut, input.GasInclude) == 1 {
				bestTokenAmountOut = tokenAmountOutResult[mid]
				bestFirstPath = firstPath
				bestSecondPath = secondPath
			}
		}

		// calculate the token amount out if we move `(mid + 1) * f.distributionPercent`% of amountIn
		if _, ok := tokenAmountOutResult[mid+1]; !ok {
			tokenAmountOutResult[mid+1], firstPath, secondPath = f.calcAdjustedTokenAmount(ctx, input, data, baseFirstPath, baseSecondPath, mid+1)
			// if better than the best found way to adjust the distribution
			if tokenAmountOutResult[mid+1].Compare(bestTokenAmountOut, input.GasInclude) == 1 {
				bestTokenAmountOut = tokenAmountOutResult[mid+1]
				bestFirstPath = firstPath
				bestSecondPath = secondPath
			}
		}

		// if tokenAmountOutResult[mid] > tokenAmountOutResult[mid+1]
		if tokenAmountOutResult[mid].Compare(tokenAmountOutResult[mid+1], input.GasInclude) == 1 {
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
func (f *HillClimbFinder) calcAdjustedTokenAmount(
	ctx context.Context,
	input findroute.Input, data findroute.FinderData,
	baseFirstPath, baseSecondPath *valueobject.Path,
	splits int,
) (*valueobject.TokenAmount, *valueobject.Path, *valueobject.Path) {
	// Step 1: calculate the adjust input for the two paths

	var (
		tokenInPriceUSD = input.TokenInPriceUSD
		tokenInDecimal  = data.TokenByAddress[input.TokenInAddress].Decimals

		amountInBigInt = input.AmountIn
		maxNumSplits   = int64(constant.OneHundredPercent / f.distributionPercent)

		amountInPerSplit    = new(big.Int).Div(amountInBigInt, big.NewInt(maxNumSplits))
		amountInPerSplitUsd = utils.CalcTokenAmountUsd(amountInPerSplit, tokenInDecimal, tokenInPriceUSD)

		firstPathInput = valueobject.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    new(big.Int).Add(baseFirstPath.Input.Amount, new(big.Int).Mul(big.NewInt(int64(splits)), amountInPerSplit)),
			AmountUsd: baseFirstPath.Input.AmountUsd + float64(splits)*amountInPerSplitUsd,
		}
		secondPathInput = valueobject.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    new(big.Int).Sub(baseSecondPath.Input.Amount, new(big.Int).Mul(big.NewInt(int64(splits)), amountInPerSplit)),
			AmountUsd: baseSecondPath.Input.AmountUsd - float64(splits)*amountInPerSplitUsd,
		}
	)

	// if any path does not satisfy minPartUSD condition, return amount out = 0
	if firstPathInput.AmountUsd < f.minPartUSD || secondPathInput.AmountUsd < f.minPartUSD {
		return nil, nil, nil
	}

	// if splits==0 then don't bother to do the duplicated calculation (same input)
	if splits == 0 {
		var amountAfterGas *big.Int
		if baseFirstPath.Output.AmountAfterGas != nil && baseSecondPath.Output.AmountAfterGas != nil {
			amountAfterGas = new(big.Int).Add(baseFirstPath.Output.AmountAfterGas, baseSecondPath.Output.AmountAfterGas)
		}
		return &valueobject.TokenAmount{
			Token:          input.TokenOutAddress,
			Amount:         new(big.Int).Add(baseFirstPath.Output.Amount, baseSecondPath.Output.Amount),
			AmountUsd:      baseFirstPath.Output.AmountUsd + baseSecondPath.Output.AmountUsd,
			AmountAfterGas: amountAfterGas,
		}, baseFirstPath, baseSecondPath
	}

	// Step 2: recalculate the rate of two paths with the new input

	var (
		tokenOutPriceUSD    = data.PriceUSDByAddress[input.TokenOutAddress]
		tokenOutPriceNative = data.TokenNativeBuyPrice(input.TokenOutAddress)
		tokenOutDecimal     = data.TokenByAddress[input.TokenOutAddress].Decimals
		gasOption           = valueobject.GasOption{
			TokenPrice:    input.GasTokenPriceUSD,
			Price:         input.GasPrice,
			GasFeeInclude: input.GasInclude,
		}
	)

	firstPathAdjusted, err := valueobject.NewPath(ctx, data.PoolBucket, baseFirstPath.PoolAddresses, baseFirstPath.Tokens,
		firstPathInput, input.TokenOutAddress, tokenOutPriceUSD, tokenOutPriceNative, tokenOutDecimal, gasOption, data.SwapLimits)

	if err != nil {
		return nil, nil, nil
	}

	secondPathAdjusted, err := valueobject.NewPath(ctx, data.PoolBucket, baseSecondPath.PoolAddresses, baseSecondPath.Tokens, secondPathInput,
		input.TokenOutAddress, tokenOutPriceUSD, tokenOutPriceNative, tokenOutDecimal, gasOption, data.SwapLimits)

	if err != nil {
		return nil, nil, nil
	}

	var amountAfterGas *big.Int
	if firstPathAdjusted.Output.AmountAfterGas != nil && secondPathAdjusted.Output.AmountAfterGas != nil {
		amountAfterGas = new(big.Int).Add(firstPathAdjusted.Output.AmountAfterGas, secondPathAdjusted.Output.AmountAfterGas)
	}
	return &valueobject.TokenAmount{
		Token:          input.TokenOutAddress,
		Amount:         new(big.Int).Add(firstPathAdjusted.Output.Amount, secondPathAdjusted.Output.Amount),
		AmountUsd:      firstPathAdjusted.Output.AmountUsd + secondPathAdjusted.Output.AmountUsd,
		AmountAfterGas: amountAfterGas,
	}, firstPathAdjusted, secondPathAdjusted
}

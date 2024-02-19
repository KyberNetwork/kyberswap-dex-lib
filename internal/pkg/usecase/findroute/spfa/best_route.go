package spfa

import (
	"context"
	"math"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfaFinder) bestRouteExactIn(ctx context.Context, input findroute.Input, data findroute.FinderData) (*valueobject.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "spfaFinder.bestRouteExactIn")
	defer span.End()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// it is fine if prices[token] is not set because it would default to zero
	tokenAmountIn := poolpkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: utils.CalcTokenAmountUsd(input.AmountIn, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress]),
	}

	hopsToTokenOut, err := common.MinHopsToTokenOut(data.PoolBucket.PerRequestPoolsByAddress, data.TokenByAddress, data.TokenToPoolAddress, input.TokenOutAddress)
	if err != nil {
		return nil, err
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > f.maxHops {
		return nil, nil
	}

	bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRoute(ctx, input, data, tokenAmountIn, data.TokenToPoolAddress, hopsToTokenOut)

	// if SaveGas option enabled, consider only single-path route
	if input.SaveGas && errFindSinglePathRoute != nil {
		return nil, errFindSinglePathRoute
	}
	if input.SaveGas && errFindSinglePathRoute == nil {
		return bestSinglePathRoute, nil
	}

	bestMultiPathRoute, errFindMultiPathRoute := f.bestMultiPathRoute(ctx, input, data, tokenAmountIn, data.TokenToPoolAddress, hopsToTokenOut)

	// cannot find any route
	if errFindSinglePathRoute != nil && errFindMultiPathRoute != nil {
		return nil, nil
	}

	// return the better route between bestSinglePathRoute and bestMultiPathRoute
	if errFindSinglePathRoute != nil || bestSinglePathRoute == nil || len(bestSinglePathRoute.Paths) == 0 {
		return bestMultiPathRoute, nil
	}

	if errFindMultiPathRoute != nil || bestMultiPathRoute == nil || len(bestMultiPathRoute.Paths) == 0 {
		return bestSinglePathRoute, nil
	}

	if bestSinglePathRoute.CompareTo(bestMultiPathRoute, input.GasInclude) > 0 {
		return bestSinglePathRoute, nil
	}

	return bestMultiPathRoute, nil
}

func (f *spfaFinder) bestSinglePathRoute(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string]*types.AddressList,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	bestPath, err := f.bestPathExactIn(ctx, input, data, tokenAmountIn, hopsToTokenOut)
	if err != nil {
		return nil, err
	}

	if bestPath == nil {
		return nil, nil
	}

	bestSinglePathRoute := valueobject.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, []*valueobject.Path{bestPath})
	return bestSinglePathRoute, nil
}

func (f *spfaFinder) bestMultiPathRoute(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string]*types.AddressList,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	var (
		splits             = f.splitAmountIn(input, data, tokenAmountIn)
		bestMultiPathRoute = valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)
	)

	for _, amountInPerSplit := range splits {
		bestPath, err := f.bestPathExactIn(ctx, input, data, amountInPerSplit, hopsToTokenOut)
		if err != nil {
			logger.WithFields(ctx, logger.Fields{"error": err}).
				Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
					input.TokenInAddress, input.TokenOutAddress, amountInPerSplit.Amount, amountInPerSplit.AmountUsd)
		}
		bestAddedPath, err := common.BestPathAmongAddedPaths(input, data, amountInPerSplit, bestMultiPathRoute.Paths)
		if err == nil && bestAddedPath.CompareTo(bestPath, input.GasInclude && data.PriceUSDByAddress[bestAddedPath.Output.Token] != 0) < 0 {
			bestPath = bestAddedPath
		}
		if bestPath == nil {
			return nil, nil
		}
		if err = bestMultiPathRoute.AddPath(data.PoolBucket, bestPath, data.SwapLimits); err != nil {
			return nil, err
		}
	}
	return bestMultiPathRoute, nil
}

// split amount in into portions of f.distributionPercent% such that each split has value >= minUsdPerSplit
// if there are remaining amount after splitting, we add to the first split (because it is always the best possible path)
// e.g. distributionPercent = 10, but we need 30% amountIn to be > minUsdPerSplit -> split 40, 30, 30
func (f *spfaFinder) splitAmountIn(input findroute.Input, data findroute.FinderData, totalAmountIn poolpkg.TokenAmount) []poolpkg.TokenAmount {
	tokenInPrice := data.PriceUSDByAddress[input.TokenInAddress]
	tokenInDecimal := data.TokenByAddress[input.TokenInAddress].Decimals

	if f.distributionPercent == constant.OneHundredPercent || tokenInPrice == 0 || totalAmountIn.AmountUsd <= f.minPartUSD {
		return []poolpkg.TokenAmount{totalAmountIn}
	}
	var (
		amountInBigInt = totalAmountIn.Amount
		amountInUsd    = totalAmountIn.AmountUsd

		// f.distributionPercent should be a divisor of 100
		// maxNumSplits is the max number of splits with each split contains a portion of f.distributionPercent% of amountIn
		// But we need to account for the f.MinPartUsd requirement by merging these splits
		maxNumSplits = int64(constant.OneHundredPercent / f.distributionPercent)

		amountInPerSplit    = new(big.Int).Div(amountInBigInt, big.NewInt(maxNumSplits))
		amountInPerSplitUsd = utils.CalcTokenAmountUsd(amountInPerSplit, tokenInDecimal, tokenInPrice)

		minSplitsToMeetMinUsdRequirement = int64(math.Max(math.Ceil(f.minPartUSD/amountInPerSplitUsd), 1))

		// the actual number of splits that we would make, considering the f.MinPartUSD requirement
		trueNumSplits           = maxNumSplits / minSplitsToMeetMinUsdRequirement
		trueAmountInPerSplit    = new(big.Int).Mul(amountInPerSplit, big.NewInt(minSplitsToMeetMinUsdRequirement))
		trueAmountInPerSplitUsd = amountInPerSplitUsd * float64(minSplitsToMeetMinUsdRequirement)

		// remaining amount after split, will be added to the first split
		remainingAmountIn    = new(big.Int).Sub(amountInBigInt, new(big.Int).Mul(trueAmountInPerSplit, big.NewInt(trueNumSplits)))
		remainingAmountInUsd = amountInUsd - trueAmountInPerSplitUsd*float64(trueNumSplits)

		splits = make([]poolpkg.TokenAmount, trueNumSplits)
	)

	splits[0] = poolpkg.TokenAmount{
		Token:     totalAmountIn.Token,
		Amount:    new(big.Int).Add(trueAmountInPerSplit, remainingAmountIn),
		AmountUsd: trueAmountInPerSplitUsd + remainingAmountInUsd,
	}
	for i := 1; i < int(trueNumSplits); i++ {
		splits[i] = poolpkg.TokenAmount{
			Token:     totalAmountIn.Token,
			Amount:    new(big.Int).Set(trueAmountInPerSplit),
			AmountUsd: trueAmountInPerSplitUsd,
		}
	}
	return splits
}

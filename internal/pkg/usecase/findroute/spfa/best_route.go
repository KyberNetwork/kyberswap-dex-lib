package spfa

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func (f *spfaFinder) bestRouteExactIn(ctx context.Context, input findroute.Input, data findroute.FinderData) (*core.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "[spfa] bestRouteExactIn")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// Optimize graph traversal by using adjacent list
	tokenToPoolAddress := make(map[string][]string)
	for poolAddress, pool := range data.PoolByAddress {
		for _, fromToken := range pool.GetTokens() {
			tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
		}
	}

	// it is fine if prices[token] is not set because it would default to zero
	tokenAmountIn := poolPkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: utils.CalcTokenAmountUsd(input.AmountIn, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress]),
	}

	hopsToTokenOut, err := common.MinHopsToTokenOut(data.PoolByAddress, data.TokenByAddress, tokenToPoolAddress, input.TokenOutAddress)
	if err != nil {
		return nil, err
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > f.maxHops {
		return nil, nil
	}

	bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRoute(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	// if SaveGas option enabled, consider only single-path route
	if input.SaveGas && errFindSinglePathRoute != nil {
		return nil, errFindSinglePathRoute
	}
	if input.SaveGas && errFindSinglePathRoute == nil {
		return bestSinglePathRoute, nil
	}

	var (
		bestMultiPathRoute    *core.Route
		errFindMultiPathRoute error
	)

	bestMultiPathRoute, errFindMultiPathRoute = f.bestMultiPathRoute(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	//fmt.Println(bestMultiPathRoute.Output.AmountUsd, f.bruteforce)
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
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	bestPath, err := f.bestPathExactIn(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
	if err != nil {
		return nil, err
	}

	if bestPath == nil {
		return nil, nil
	}

	bestSinglePathRoute := core.NewRouteFromPaths(poolPkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: 0,
	}, input.TokenOutAddress, data.PoolByAddress, []*core.Path{bestPath})
	return bestSinglePathRoute, nil
}

func (f *spfaFinder) bestMultiPathRoute(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	var (
		splits             = f.splitAmountIn(input, data, tokenAmountIn)
		bestMultiPathRoute = core.NewEmptyRouteFromPoolData(poolPkg.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    input.AmountIn,
			AmountUsd: 0,
		}, input.TokenOutAddress, data.PoolByAddress)
	)
	for _, amountInPerSplit := range splits {
		bestPath, err := f.bestPathExactIn(ctx, input, data, amountInPerSplit, tokenToPoolAddress, hopsToTokenOut)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).
				Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
					input.TokenInAddress, input.TokenOutAddress, amountInPerSplit.Amount, amountInPerSplit.AmountUsd)
		}
		bestAddedPath, err := common.BestPathAmongAddedPaths(input, data, amountInPerSplit, bestMultiPathRoute.Paths)
		if err == nil && bestAddedPath.CompareTo(bestPath, input.GasInclude) < 0 {
			bestPath = bestAddedPath
		}
		if bestPath == nil {
			return nil, nil
		}
		if ok := bestMultiPathRoute.AddPath(bestPath); !ok {
			return nil, fmt.Errorf("cannot add path to bestMultiPathRoute")
		}
	}
	return bestMultiPathRoute, nil
}

// split amount in into portions of f.distributionPercent% such that each split has value >= minUsdPerSplit
// if there are remaining amount after splitting, we add to the first split (because it is always the best possible path)
// e.g. distributionPercent = 10, but we need 30% amountIn to be > minUsdPerSplit -> split 40, 30, 30
func (f *spfaFinder) splitAmountIn(input findroute.Input, data findroute.FinderData, totalAmountIn poolPkg.TokenAmount) []poolPkg.TokenAmount {
	tokenInPrice := data.PriceUSDByAddress[input.TokenInAddress]
	tokenInDecimal := data.TokenByAddress[input.TokenInAddress].Decimals

	if f.distributionPercent == constant.OneHundredPercent || tokenInPrice == 0 || totalAmountIn.AmountUsd <= f.minPartUSD {
		return []poolPkg.TokenAmount{totalAmountIn}
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

		splits = make([]poolPkg.TokenAmount, trueNumSplits)
	)

	splits[0] = poolPkg.TokenAmount{
		Token:     totalAmountIn.Token,
		Amount:    new(big.Int).Add(trueAmountInPerSplit, remainingAmountIn),
		AmountUsd: trueAmountInPerSplitUsd + remainingAmountInUsd,
	}
	for i := 1; i < int(trueNumSplits); i++ {
		splits[i] = poolPkg.TokenAmount{
			Token:     totalAmountIn.Token,
			Amount:    new(big.Int).Set(trueAmountInPerSplit),
			AmountUsd: trueAmountInPerSplitUsd,
		}
	}
	return splits
}

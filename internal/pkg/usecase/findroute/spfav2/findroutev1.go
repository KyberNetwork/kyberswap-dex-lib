package spfav2

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfav2Finder) findrouteV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.findrouteV1")
	defer span.End()

	bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRouteV1(ctx, input, data, tokenAmountIn, hopsToTokenOut)

	// if SaveGas option enabled, consider only single-path route
	if input.SaveGas && errFindSinglePathRoute != nil {
		return nil, errFindSinglePathRoute
	}
	if input.SaveGas && errFindSinglePathRoute == nil {
		return bestSinglePathRoute, nil
	}

	bestMultiPathRoute, errFindMultiPathRoute := f.bestMultiPathRouteV1(ctx, input, data, tokenAmountIn, hopsToTokenOut)

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

func (f *spfav2Finder) bestSinglePathRouteV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestSinglePathRouteV1")
	defer span.End()

	bestPath, err := f.bestPathExactInV1(ctx, input, data, tokenAmountIn, hopsToTokenOut)
	if err != nil {
		return nil, err
	}

	if bestPath == nil {
		return nil, nil
	}

	bestSinglePathRoute := valueobject.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, []*valueobject.Path{bestPath})
	return bestSinglePathRoute, nil
}

func (f *spfav2Finder) bestMultiPathRouteV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestMultiPathRouteV1")
	defer span.End()

	var (
		splits             = f.splitAmountIn(input, data, tokenAmountIn)
		bestMultiPathRoute = valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)
	)
	if len(splits) == 1 {
		// just use bestSinglePathRoute
		return nil, nil
	}
	for _, amountInPerSplit := range splits {
		bestPath, err := f.bestPathExactInV1(ctx, input, data, amountInPerSplit, hopsToTokenOut)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).
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

func (f *spfav2Finder) bestPathExactInV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Path, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestPathExactInV1")
	defer span.End()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// only pick one best path, so set maxPathsToGenerate = 1.
	paths, err := common.GenKthBestPaths(ctx, input, data, tokenAmountIn, hopsToTokenOut, f.maxHops, 1, 1)
	defer valueobject.ReturnPaths(paths)

	if err != nil {
		return nil, err
	}
	var bestPath *valueobject.Path
	for _, path := range paths {
		if path != nil && path.CompareTo(bestPath, input.GasInclude && data.PriceUSDByAddress[path.Output.Token] != 0) < 0 {
			bestPath = path
		}
	}
	return bestPath.Clone(), nil
}

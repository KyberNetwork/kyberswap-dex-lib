package spfav2

import (
	"context"
	"fmt"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfav2Finder) findrouteV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.findrouteV1")
	defer span.Finish()

	bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRouteV1(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	// if SaveGas option enabled, consider only single-path route
	if input.SaveGas && errFindSinglePathRoute != nil {
		return nil, errFindSinglePathRoute
	}
	if input.SaveGas && errFindSinglePathRoute == nil {
		return bestSinglePathRoute, nil
	}

	bestMultiPathRoute, errFindMultiPathRoute := f.bestMultiPathRouteV1(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

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
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestSinglePathRouteV1")
	defer span.Finish()

	bestPath, err := f.bestPathExactInV1(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
	if err != nil {
		return nil, err
	}

	if bestPath == nil {
		return nil, nil
	}

	bestSinglePathRoute := core.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, data.PoolByAddress, []*core.Path{bestPath})
	return bestSinglePathRoute, nil
}

func (f *spfav2Finder) bestMultiPathRouteV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	var (
		splits             = f.splitAmountIn(input, data, tokenAmountIn)
		bestMultiPathRoute = core.NewEmptyRouteFromPoolData(input.TokenInAddress, input.TokenOutAddress, data.PoolByAddress)
	)
	for _, amountInPerSplit := range splits {
		bestPath, err := f.bestPathExactInV1(ctx, input, data, amountInPerSplit, tokenToPoolAddress, hopsToTokenOut)
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

func (f *spfav2Finder) bestPathExactInV1(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestPathExactInV1")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// only pick one best path, so set maxPathsToGenerate = 1.
	paths, err := common.GenKthBestPaths(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut, f.maxHops, 1, 1)
	if err != nil {
		return nil, err
	}
	var bestPath *core.Path
	for _, path := range paths {
		if path != nil && path.CompareTo(bestPath, input.GasInclude) < 0 {
			bestPath = path
		}
	}
	return bestPath, nil
}

package spfav2

import (
	"context"
	"fmt"
	"sort"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfav2Finder) findrouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.findrouteV2")
	defer span.Finish()

	if input.SaveGas {
		bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRouteV1(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
		if errFindSinglePathRoute != nil {
			return nil, errFindSinglePathRoute
		}
		return bestSinglePathRoute, nil
	}

	bestMultiPathRoute, errFindMultiPathRoute := f.bestRouteV2(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	if errFindMultiPathRoute != nil {
		return nil, errFindMultiPathRoute
	}

	return bestMultiPathRoute, nil
}

func (f *spfav2Finder) bestRouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestRouteV2")
	defer span.Finish()

	var splits = f.splitAmountIn(input, data, tokenAmountIn)

	// step 1: generate k paths with best rate.
	// k <= maxHop * maxPathsToReturn
	// maxPathsToGenerate is a parameter we can control, higher maxPathsToGenerate <=> more diverse (better rate) paths found + higher runtime
	if len(splits) == 0 {
		return nil, fmt.Errorf("cannot split amount in")
	}
	amountInToGeneratePath := splits[len(splits)-1]

	numberOfPathToGenerate := uint32(len(splits))
	if numberOfPathToGenerate > f.maxPathsToGenerate {
		numberOfPathToGenerate = f.maxPathsToGenerate
	}

	paths, errGenPath := common.GenKthBestPaths(ctx, input, data, amountInToGeneratePath, tokenToPoolAddress, hopsToTokenOut, f.maxHops, numberOfPathToGenerate, f.maxPathsToReturn)
	if errGenPath != nil {
		logger.WithFields(logger.Fields{"error": errGenPath}).
			Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
				input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd)
		return nil, nil
	}
	cmpFunc := func(a, b int) bool { return paths[a].CompareTo(paths[b], input.GasInclude) < 0 }
	sort.Slice(paths, cmpFunc)

	// step 2: Find single-path route
	bestSinglePathRoute := f.bestSinglePathRouteV2(input, data, tokenAmountIn, paths, len(splits))

	// step 3: Find multi-path route
	// For each chunk (split), iterate through generated k paths to recalculate amountOut and get the best path among them

	bestMultiPathRoute := valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)
	h := NewFindPathV2Helper(len(paths), int(f.maxPathsInRoute), amountInToGeneratePath, cmpFunc)

	for _, amountInPerSplit := range splits {
		bestPath := h.bestPathExactInV2(ctx, input, data, paths, amountInPerSplit)
		if bestPath == nil {
			return nil, nil
		}

		if err := bestMultiPathRoute.AddPath(data.PoolBucket, bestPath); err != nil {
			return nil, err
		}
	}

	// step 4: compare and return the best route
	if bestSinglePathRoute == nil {
		return bestMultiPathRoute, nil
	}

	if bestMultiPathRoute == nil {
		return bestSinglePathRoute, nil
	}

	if bestSinglePathRoute.CompareTo(bestMultiPathRoute, input.GasInclude) > 0 {
		return bestSinglePathRoute, nil
	}
	return bestMultiPathRoute, nil
}

func (f *spfav2Finder) bestSinglePathRouteV2(
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	paths []*valueobject.Path,
	numberOfPathToTry int,
) *valueobject.Route {
	var bestPath *valueobject.Path
	for i := 0; i < numberOfPathToTry && i < len(paths); i++ {
		if paths[i] == nil {
			continue
		}
		path := newPath(input, data, paths[i].PoolAddresses, paths[i].Tokens, tokenAmountIn, false)
		if path != nil && (bestPath == nil || path.CompareTo(bestPath, input.GasInclude) < 0) {
			bestPath = path
		}
	}

	if bestPath == nil {
		return nil
	}

	return valueobject.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, []*valueobject.Path{bestPath})
}

package spfav2

import (
	"context"
	"fmt"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfav2Finder) findrouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.findrouteV2")
	defer span.End()

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
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestRouteV2")
	defer span.End()

	var (
		generatedBestPaths []*entity.MinimalPath
		paths              []*valueobject.Path
	)

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

	if input.IsPathGeneratorEnabled && f.getGeneratedBestPaths != nil {
		generatedBestPaths = f.getGeneratedBestPaths(input.SourceHash, input.TokenInAddress, input.TokenOutAddress)
	}

	// if api params set IsPathGeneratorEnabled = true, or we have isPathGeneratorEnabled on the yaml config
	if input.IsPathGeneratorEnabled && len(generatedBestPaths) > 0 {
		metrics.IncrFindRoutePregenCount(true, nil)
		logger.WithFields(logger.Fields{"pre_computed_bestPaths_len": len(generatedBestPaths)}).Infof(
			"used precomputated_paths. sourceHash %v tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
			input.SourceHash, input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd,
		)

		paths = bestPathToPath(ctx, input, data, amountInToGeneratePath, generatedBestPaths)
	} else {
		metrics.IncrFindRoutePregenCount(false, map[string]string{
			"reason": "doesn't hit",
		})
		logger.Infof("manually gen Path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
			input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd,
		)

		var errGenPath error
		paths, errGenPath = common.GenKthBestPaths(ctx, input, data, amountInToGeneratePath, tokenToPoolAddress, hopsToTokenOut, f.maxHops, numberOfPathToGenerate, f.maxPathsToReturn)
		if errGenPath != nil {
			logger.WithFields(logger.Fields{"error": errGenPath}).
				Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
					input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd)
			return nil, nil
		}
	}

	cmpFunc := func(a, b int) bool { return paths[a].CompareTo(paths[b], input.GasInclude) < 0 }
	sort.Slice(paths, cmpFunc)

	// step 2: Find single-path route
	bestSinglePathRoute := f.bestSinglePathRouteV2(input, data, tokenAmountIn, paths, len(splits))

	// step 3: Find multi-path route
	bestMultiPathRoute, errFindMultiPathRoute := f.bestMultiPathRouteV2(ctx, input, data, paths, amountInToGeneratePath, splits, cmpFunc)

	// step 4: compare and return the best route
	if bestSinglePathRoute == nil {
		return bestMultiPathRoute, nil
	}

	if errFindMultiPathRoute != nil || bestMultiPathRoute == nil {
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
	tokenAmountIn poolpkg.TokenAmount,
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

func (f *spfav2Finder) bestMultiPathRouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	paths []*valueobject.Path,
	amountInToGeneratePath poolpkg.TokenAmount,
	splits []poolpkg.TokenAmount,
	cmpFunc func(a, b int) bool,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestMultiPathRouteV2")
	defer span.End()

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
	return bestMultiPathRoute, nil
}

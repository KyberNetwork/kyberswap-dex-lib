package spfav2

import (
	"context"
	"fmt"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (f *spfav2Finder) findrouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.findrouteV2")
	defer span.End()

	if input.SaveGas {
		bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRouteV1(ctx, input, data, tokenAmountIn, hopsToTokenOut)
		if errFindSinglePathRoute != nil {
			return nil, errFindSinglePathRoute
		}
		return bestSinglePathRoute, nil
	}

	bestMultiPathRoute, errFindMultiPathRoute := f.bestRouteV2(ctx, input, data, tokenAmountIn, hopsToTokenOut)

	if errFindMultiPathRoute != nil {
		return nil, errFindMultiPathRoute
	}

	return bestMultiPathRoute, nil
}

func (f *spfav2Finder) isPregenPathValid(
	ctx context.Context,
	data findroute.FinderData,
	paths []*entity.MinimalPath,
) bool {
	span, _ := tracer.StartSpanFromContext(ctx, "spfav2Finder.isPregenPathValid")
	defer span.End()
	for _, path := range paths {
		for _, pool := range path.Pools {
			iPool, exist := data.PoolBucket.GetPool(pool)
			if !exist {
				logger.WithFields(ctx, logger.Fields{
					"poolAddress": pool,
				}).Warnf("Pregen check fail: pool in the pregenPaths is not exist")
				return false
			}

			// we check if the pool is kyber pmm and all reserves are zero
			reserves := iPool.GetReserves()
			zeroReserves := true
			for _, reserve := range reserves {
				if reserve.Cmp(constant.Zero) > 0 {
					zeroReserves = false
					break
				}
			}

			// a pmm pool is delete if all reserves are zero
			if zeroReserves {
				logger.WithFields(ctx, logger.Fields{
					"poolAddress": pool,
					"poolType":    iPool.GetType(),
					"reserves":    iPool.GetReserves(),
				}).Warnf("Pregen check fail: pool in the pregenPaths is deleted or has zero reserves %s", iPool.GetReserves())
				return false
			}
		}
	}

	return true
}

func (f *spfav2Finder) bestRouteV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
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

	isPregenPathValid := f.isPregenPathValid(ctx, data, generatedBestPaths)
	metrics.IncrIsPregenPathValidCount(ctx, isPregenPathValid, nil)

	// if api params set IsPathGeneratorEnabled = true, or we have isPathGeneratorEnabled on the yaml config
	if input.IsPathGeneratorEnabled && len(generatedBestPaths) > 0 && isPregenPathValid {
		metrics.IncrFindRoutePregenCount(ctx, true, nil)
		logger.WithFields(ctx, logger.Fields{"pre_computed_bestPaths_len": len(generatedBestPaths)}).Infof(
			"used precomputated_paths. sourceHash %v tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
			input.SourceHash, input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd,
		)

		paths = bestPathToPath(ctx, input, data, amountInToGeneratePath, generatedBestPaths)
	} else {
		metrics.IncrFindRoutePregenCount(ctx, false, map[string]string{
			"reason": "doesn't hit",
		})
		logger.Debugf(ctx, "manually gen Path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
			input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd,
		)

		var errGenPath error
		paths, errGenPath = common.GenKthBestPaths(ctx, input, data, amountInToGeneratePath, hopsToTokenOut, f.maxHops, numberOfPathToGenerate, f.maxPathsToReturn)
		if errGenPath != nil {
			logger.WithFields(ctx, logger.Fields{"error": errGenPath}).
				Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
					input.TokenInAddress, input.TokenOutAddress, amountInToGeneratePath, amountInToGeneratePath.AmountUsd)
			return nil, nil
		}
	}
	defer valueobject.ReturnPaths(paths)

	if len(paths) == 0 {
		return nil, findroute.ErrNoPath
	}

	cmpFunc := func(a, b int) bool {
		priceAvailable := data.BuyPriceAvailable(paths[a].Output.Token)
		return paths[a].CompareTo(paths[b], input.GasInclude && priceAvailable) < 0
	}
	sort.Slice(paths, cmpFunc)

	// step 2: Find single-path route
	bestSinglePathRoute := f.bestSinglePathRouteV2(ctx, input, data, tokenAmountIn, paths, len(splits))

	// step 3: Find multi-path route
	bestMultiPathRoute, errFindMultiPathRoute := f.bestMultiPathRouteV2(ctx, input, data, paths, amountInToGeneratePath, splits, cmpFunc)

	logger.Debugf(ctx, "bestSinglePathRoute %v, bestMultiPathRoute %v, errFindMultiPathRoute %v", bestSinglePathRoute, bestMultiPathRoute, errFindMultiPathRoute)

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
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
	paths []*valueobject.Path,
	numberOfPathToTry int,
) *valueobject.Route {
	var bestPath *valueobject.Path
	for i := 0; i < numberOfPathToTry && i < len(paths); i++ {
		if paths[i] == nil {
			continue
		}
		path := newPath(ctx, input, data, paths[i].PoolAddresses, paths[i].Tokens, tokenAmountIn, false)
		if path != nil && (bestPath == nil || path.CompareTo(bestPath, input.GasInclude && data.BuyPriceAvailable(path.Output.Token)) < 0) {
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
	amountInToGeneratePath valueobject.TokenAmount,
	splits []valueobject.TokenAmount,
	cmpFunc func(a, b int) bool,
) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestMultiPathRouteV2")
	defer span.End()

	// For each chunk (split), iterate through generated k paths to recalculate amountOut and get the best path among them
	bestMultiPathRoute := valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)
	h := NewFindPathV2Helper(len(paths), int(f.maxPathsInRoute), amountInToGeneratePath, cmpFunc)

	for _, amountInPerSplit := range splits {
		//continuously pop the bestPath and add it until we either has no path left or we got a valid path for route
		count := 0
		for {
			bestPath := h.bestPathExactInV2(ctx, input, data, paths, amountInPerSplit)
			if bestPath == nil {
				logger.Warn(ctx, "no more paths to try.")
				return nil, nil
			}

			if err := bestMultiPathRoute.AddPath(ctx, data.PoolBucket, bestPath.Clone(), data.SwapLimits); err == nil {
				break
			} else {
				// the below logic fixes specifically a PMM swapLimit issue, if bestPath doesn't have PMM pool, just return
				if !bestPath.HasPoolType(data.PoolBucket.PerRequestPoolsByAddress, pooltypes.PoolTypes.KyberPMM) {
					return nil, err
				}

				count++
				if count >= 3 {
					logger.Error(ctx, "AddPath failed 3 times, no more try.")
					return nil, err
				}

				logger.Warnf(ctx, "AddPath crash into error, pop next path. Error :%s", err)
			}
		}

	}
	return bestMultiPathRoute, nil
}

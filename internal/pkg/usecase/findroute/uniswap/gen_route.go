package uniswap

import (
	"context"
	"math/big"
	"sort"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type routeNode struct {
	pathsOnRoute            []*core.Path
	lastDistributionPercent int
	remainingPercent        int
	nSwapUsed               int
}

func (f *uniswapFinder) genSinglePathRoutes(
	ctx context.Context, input findroute.Input, data findroute.FinderData, paths []*core.Path,
) []*core.Route {
	span, _ := tracer.StartSpanFromContext(ctx, "[uniswap] genSinglePathRoutes")
	defer span.Finish()

	singlePathRoutes := make([]*core.Route, 0, len(paths))
	for _, path := range paths {
		singlePathRoutes = append(singlePathRoutes, core.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, data.PoolByAddress, []*core.Path{path}))
	}
	sort.Slice(singlePathRoutes, func(i, j int) bool {
		return singlePathRoutes[i].CompareTo(singlePathRoutes[j], input.GasInclude) > 0
	})
	if uint32(len(singlePathRoutes)) < f.maxRoutes {
		return singlePathRoutes
	}
	return singlePathRoutes[:f.maxRoutes]
}

func (f *uniswapFinder) genBestRoutes(
	ctx context.Context, input findroute.Input, data findroute.FinderData, paths []*core.Path,
) ([]*core.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "[uniswap] genBestRoutes")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// Step 1: for each multiple distributionPercent (<=100), we try swap using generated "paths" with amountIn=input.AmountIn*.../100
	// percentToPath = {5:{path1,path2,...}, 10:{path3,path4,...},...,100:{path_x,path_y,...}}
	// percents = {5,10,15,20,...,100}
	percentToPath, percents := f.genPathsWithSplitAmountIn(input, data, paths)

	var (
		// currentLayer now contains routes that consist of exactly one path
		currentLayer = f.initFirstLayer(percentToPath, percents)
		routes       []*core.Route
	)

	// Step 2: Perform layered BFS, each edge would be a path => adding a path = travel an edge
	for currentNumberOfPaths := uint32(1); currentNumberOfPaths <= f.maxPaths; currentNumberOfPaths++ {
		routes = append(routes, getPossibleRoutesFromCurrentLayer(input, data, currentLayer)...)

		if currentNumberOfPaths < f.maxPaths {
			currentLayer = f.getNextLayerOfRoutes(percentToPath, percents, currentLayer)
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].CompareTo(routes[j], input.GasInclude) > 0
	})
	// TODO If there are additional requirements on routes (e.g total volume through a dex cannot exceed ...), add filters here
	if uint32(len(routes)) < f.maxRoutes {
		return routes, nil
	}
	return routes[:f.maxRoutes], nil
}

func (f *uniswapFinder) genPathsWithSplitAmountIn(input findroute.Input, data findroute.FinderData, paths []*core.Path) (percentToPath map[int][]*core.Path, percents []int) {
	percentToPath = make(map[int][]*core.Path)

	for percentIndex := 1; int(f.distributionPercent)*percentIndex <= constant.OneHundredPercent; percentIndex++ {
		var (
			percent = int(f.distributionPercent) * percentIndex
			amount  = new(big.Int).Div(
				new(big.Int).Mul(big.NewInt(int64(percent)), input.AmountIn),
				big.NewInt(constant.OneHundredPercent),
			)
			amountUSD          = utils.CalcTokenAmountUsd(amount, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress])
			splitTokenAmountIn = poolPkg.TokenAmount{
				Token:     input.TokenInAddress,
				Amount:    amount,
				AmountUsd: amountUSD,
			}
		)
		if percent < constant.OneHundredPercent && amountUSD < f.minPartUSD {
			continue
		}

		percents = append(percents, percent)

		var splitPaths = make([]*core.Path, 0, len(paths))
		for _, path := range paths {
			splitPath, err := core.NewPath(path.Pools, path.Tokens, splitTokenAmountIn, input.TokenOutAddress,
				data.PriceUSDByAddress[input.TokenOutAddress], data.TokenByAddress[input.TokenOutAddress].Decimals,
				core.GasOption{GasFeeInclude: input.GasInclude, Price: input.GasPrice, TokenPrice: input.GasTokenPriceUSD},
			)
			if err != nil {
				logger.WithFields(logger.Fields{"error": err}).
					Debug("cannot init new path with split amount")
			} else {
				splitPaths = append(splitPaths, splitPath)
			}
		}
		// better path first
		sort.Slice(splitPaths, func(i, j int) bool {
			return splitPaths[i].CompareTo(splitPaths[j], input.GasInclude) < 0
		})
		percentToPath[percent] = splitPaths
	}
	return percentToPath, percents
}

func (f *uniswapFinder) initFirstLayer(percentToPath map[int][]*core.Path, percents []int) []*routeNode {
	var layer []*routeNode
	for percentIndex, percent := range percents {
		for _, path := range getNonOverlappingPathsOfEachLength(f.maxHops, percentToPath[percent], nil) {
			if uint32(len(path.Pools)) > f.maxPools {
				continue
			}
			layer = append(layer, &routeNode{
				pathsOnRoute:            []*core.Path{path},
				remainingPercent:        constant.OneHundredPercent - percent,
				lastDistributionPercent: percentIndex,
				nSwapUsed:               len(path.Pools),
			})
		}
	}
	return layer
}

func (f *uniswapFinder) getNextLayerOfRoutes(percentToPath map[int][]*core.Path, percents []int, currentLayer []*routeNode) []*routeNode {
	var nextLayer []*routeNode
	for _, currentRoute := range currentLayer {
		nextLayer = append(nextLayer, f.getNextRoutesFromCurrentRoute(percentToPath, percents, currentRoute)...)
	}
	return nextLayer
}

func (f *uniswapFinder) getNextRoutesFromCurrentRoute(percentToPath map[int][]*core.Path, percents []int, currentRoute *routeNode) []*routeNode {
	var nextRoutes []*routeNode
	for index := 0; index <= currentRoute.lastDistributionPercent && percents[index] <= currentRoute.remainingPercent; index++ {
		// iterate at most maxHop path here
		for _, path := range getNonOverlappingPathsOfEachLength(f.maxHops, percentToPath[percents[index]], currentRoute.pathsOnRoute) {
			if uint32(currentRoute.nSwapUsed+len(path.Pools)) > f.maxPools {
				continue
			}
			nextRoutes = append(nextRoutes, &routeNode{
				lastDistributionPercent: index,
				remainingPercent:        currentRoute.remainingPercent - percents[index],
				nSwapUsed:               currentRoute.nSwapUsed + len(path.Pools),
				pathsOnRoute:            append(append([]*core.Path{}, currentRoute.pathsOnRoute...), path),
			})
		}
	}
	return nextRoutes
}

func getPossibleRoutesFromCurrentLayer(input findroute.Input, data findroute.FinderData, currentLayer []*routeNode) []*core.Route {
	var possibleRoutes []*core.Route
	for _, node := range currentLayer {
		if node.remainingPercent == 0 {
			possibleRoutes = append(possibleRoutes, core.NewRouteFromPaths(input.TokenInAddress, input.TokenOutAddress, data.PoolByAddress, node.pathsOnRoute))
		}
	}
	return possibleRoutes
}

// return best path for each hop
func getNonOverlappingPathsOfEachLength(maxHops uint32, paths []*core.Path, usedPaths []*core.Path) []*core.Path {
	var (
		usedPoolAddresses     = sets.NewString()
		foundPathOfLen        = make([]bool, maxHops)
		bestPathForEachLength []*core.Path
	)

	for _, path := range usedPaths {
		for _, pool := range path.Pools {
			usedPoolAddresses.Insert(pool.GetAddress())
		}
	}
	// since paths is sorted in decreasing order of amountOut (or amountOutUsd),
	// the first path found of each len is the best path of that len
	for _, path := range paths {
		pathLen := len(path.Pools)
		poolAddressesOnPath := make([]string, 0, pathLen)
		for _, pool := range path.Pools {
			poolAddressesOnPath = append(poolAddressesOnPath, pool.GetAddress())
		}
		if !foundPathOfLen[pathLen-1] && !usedPoolAddresses.HasAny(poolAddressesOnPath...) {
			foundPathOfLen[pathLen-1] = true
			bestPathForEachLength = append(bestPathForEachLength, path)
			// we have found maxHops paths (one for each len) -> we can break
			if uint32(len(bestPathForEachLength)) == maxHops {
				break
			}
		}
	}
	return bestPathForEachLength
}

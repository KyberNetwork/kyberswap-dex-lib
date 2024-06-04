package hillclimb

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	defaultHillClimbDistributionPercent uint32  = 1
	defaultHillClimbIteration           uint32  = 2
	defaultHillClimbMinPartUSD          float64 = 500
)

type hillClimbFinder struct {
	distributionPercent uint32

	hillClimbIteration uint32

	minPartUSD float64

	baseIFinder findroute.IFinder
}

func NewHillClimbingFinder(distributionPercent, hillClimbIteration uint32, minPartUSD float64, baseIFinder findroute.IFinder) *hillClimbFinder {
	return &hillClimbFinder{distributionPercent, hillClimbIteration, minPartUSD, baseIFinder}
}

func NewDefaultHillClimbingFinder() *hillClimbFinder {
	return NewHillClimbingFinder(defaultHillClimbDistributionPercent, defaultHillClimbIteration, defaultHillClimbMinPartUSD, spfav2.NewDefaultSPFAv2Finder())
}

func (f *hillClimbFinder) Find(ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "hillClimbFinder.Find")
	defer span.End()

	// NOTE: We need to deeply clone swapLimit before doing Route.AddPath because this action will change swapLimit value directly.
	// Step 1: Find the route using spfav2 (with split = 5%)
	data.Refresh()
	baseBestRoutes, err := f.baseIFinder.Find(ctx, input, data)
	if err != nil {
		// this error will be propagated up and logged at the response, no need to log here
		return nil, err
	}

	baseBestRoute := extractBestRoute(baseBestRoutes)
	if baseBestRoute == nil {
		logger.Infof(ctx, "hill climb: extract best base route failed %s", err)
		return nil, nil
	}

	if len(baseBestRoute.Paths) == 1 {
		logger.Debugf(ctx, "hill climb: return baseBestRoute due to lenPaths == 1")
		return []*valueobject.Route{baseBestRoute}, nil
	}

	// recalculate rate of route again to ensure consistency with summarize
	data.Refresh()
	baseBestRoute = recalculateRoute(ctx, input, data, baseBestRoute)
	if baseBestRoute == nil {
		logger.Infof(ctx, "hill climb: return nil due to cannot recalculateRoute base")
		return nil, nil
	}

	// Step 2: Use hill-climb to adjust the rate between each split
	// Replace the original route if the rate from hill climb is better
	data.Refresh()
	hillClimbBestRoute, err := f.optimizeRoute(ctx, input, data, baseBestRoute)
	if err != nil {
		logger.Infof(ctx, "hill climb: optimizeRoute failed %s", err)
		return []*valueobject.Route{baseBestRoute}, nil
	}

	// recalculate rate of route again to ensure consistency with summarize
	data.Refresh()
	hillClimbBestRoute = recalculateRoute(ctx, input, data, hillClimbBestRoute)

	logger.Debugf(ctx,
		"successfully using hill climb to optimize route from token %v to token %v", input.TokenInAddress, input.TokenOutAddress,
	)

	// if the route cannot be optimized or the input is different from the input of base best route
	if hillClimbBestRoute == nil || hillClimbBestRoute.Input.CompareRaw(&baseBestRoute.Input) != 0 {
		logger.Infof(ctx,
			"hill climb: used baseRoute which better",
		)
		return []*valueobject.Route{baseBestRoute}, nil
	}
	if hillClimbBestRoute.CompareTo(baseBestRoute, input.GasInclude) > 0 {
		logger.Infof(ctx,
			"hill climb: used hillClimb Route which better",
		)
		return []*valueobject.Route{hillClimbBestRoute}, nil
	}

	return []*valueobject.Route{baseBestRoute}, nil
}

func recalculateRoute(ctx context.Context, input findroute.Input, data findroute.FinderData, route *valueobject.Route) *valueobject.Route {
	var (
		tokenOutPriceUSD    = data.PriceUSDByAddress[input.TokenOutAddress]
		tokenOutPriceNative = data.TokenNativeBuyPrice(input.TokenOutAddress)
		tokenOutDecimal     = data.TokenByAddress[input.TokenOutAddress].Decimals
		gasOption           = valueobject.GasOption{
			TokenPrice:    input.GasTokenPriceUSD,
			Price:         input.GasPrice,
			GasFeeInclude: input.GasInclude,
		}
		newRoute = valueobject.NewRoute(input.TokenInAddress, input.TokenOutAddress)
	)

	for i := 0; i < len(route.Paths); i++ {
		pathRecalculated, err := valueobject.NewPath(ctx, data.PoolBucket, route.Paths[i].PoolAddresses, route.Paths[i].Tokens,
			route.Paths[i].Input, input.TokenOutAddress, tokenOutPriceUSD, tokenOutPriceNative, tokenOutDecimal, gasOption, data.SwapLimits)
		if err != nil {
			return nil
		}

		if err = newRoute.AddPath(ctx, data.PoolBucket, pathRecalculated, data.SwapLimits); err != nil {
			return nil
		}
	}
	return newRoute
}

func extractBestRoute(routes []*valueobject.Route) *valueobject.Route {
	if len(routes) == 0 {
		return nil
	}
	return routes[0]
}

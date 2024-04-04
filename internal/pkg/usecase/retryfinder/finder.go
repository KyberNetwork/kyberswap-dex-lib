package retryfinder

import (
	"context"
	"math/big"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type RetryFinder struct {
	baseIFinder findroute.IFinder
}

func NewRetryFinder(baseFinder findroute.IFinder) *RetryFinder {
	return &RetryFinder{baseIFinder: baseFinder}
}

// extractBestRoute returns the best routes among routes
func extractBestRoute(routes []*valueobject.Route) *valueobject.Route {
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

func (r *RetryFinder) Find(ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*valueobject.Route, error) {
	data.Refresh()
	routes, err := r.baseIFinder.Find(ctx, input, data)
	if err != nil {
		logger.Errorf(ctx, "retry finder: baseIFinder failed %s", err)
		return nil, err
	}
	if len(routes) == 0 {
		logger.Debugf(ctx, "retry finder: extract best base route failed %s", err)
		return nil, nil
	}
	//clear/ restart.
	data.Refresh()

	baseBestRoute := extractBestRoute(routes)
	if baseBestRoute == nil {
		return routes, nil
	}
	newRoute := r.retryDynamicPools(ctx, baseBestRoute, []string{kyberpmm.DexTypeKyberPMM, limitorder.DexTypeLimitOrder},
		data,
		valueobject.GasOption{
			GasFeeInclude: input.GasInclude,
			Price:         input.GasPrice,
			TokenPrice:    input.GasTokenPriceUSD,
		})
	if newRoute != nil && newRoute.CompareTo(baseBestRoute, input.GasInclude) > 0 {
		logger.Infof(ctx,
			"retry finder: success retry with better rate baseAmountOut: %s baseAmountOutUsd %s poolAddr %s, and newAmountOut %s newAmountOutUsd %s poolAddr %s",
			baseBestRoute.Output.Amount.String(),
			baseBestRoute.Output.AmountUsd,
			baseBestRoute.ExtractPoolAddresses(),
			newRoute.Output.Amount.String(),
			newRoute.Output.AmountUsd,
			newRoute.ExtractPoolAddresses())
		routes = append([]*valueobject.Route{newRoute}, routes...)
	}

	return routes, nil
}

// retryDynamicPools will re try the found route with dyanmic pools type (i.e, pools given higher amount out ratio at higher amountin).
// WARNING: finderData must be a fresh copy.
func (r *RetryFinder) retryDynamicPools(ctx context.Context, route *valueobject.Route, dynamicTypes []string, data findroute.FinderData, gasOption valueobject.GasOption) *valueobject.Route {
	var newRoute = valueobject.NewRoute(route.Input.Token, route.Output.Token)
	typeSet := sets.NewString(dynamicTypes...)
	poolsInRoute := route.ExtractPoolAddresses()
	routeModified := false
	for i := 0; i < len(route.Paths); i++ {
		var (
			newPath  *valueobject.Path
			currPath = route.Paths[i] //shallow copy
			inp      = &poolpkg.TokenAmount{
				Token:     currPath.Input.Token,
				Amount:    big.NewInt(0).Set(currPath.Input.Amount),
				AmountUsd: currPath.Input.AmountUsd,
			}
			currOutput = &poolpkg.TokenAmount{
				Token:     "",
				Amount:    big.NewInt(0),
				AmountUsd: 0,
			}
			modified       = false
			poolsOnNewPath = make([]string, len(currPath.PoolAddresses))
		)
		for pIndex := 0; pIndex < len(currPath.PoolAddresses); pIndex++ {
			currPool, avail := data.PoolBucket.GetPool(currPath.PoolAddresses[pIndex])
			if !avail {
				logger.Errorf(ctx, "pool is removed from pool bucket, poolAddress: %s", currPath.PoolAddresses[pIndex])
				return route
			}
			result, err := poolpkg.CalcAmountOut(currPool, *inp, currPath.Tokens[pIndex+1].Address, data.SwapLimits[currPool.GetType()])
			if err != nil {
				logger.Errorf(ctx, "cannot calculate amount out for pool %s, error: %s", currPool.GetAddress(), err)
				return route
			}
			currOutput = result.TokenAmountOut
			bestNewPool, newAmount := findNewBestPoolWithAmount(data, inp, result.TokenAmountOut, typeSet, poolsInRoute)

			if bestNewPool != "" {
				modified = true
				poolsOnNewPath[pIndex] = bestNewPool
				currOutput = newAmount
				poolsInRoute.Insert(bestNewPool)
			} else {
				poolsOnNewPath[pIndex] = currPath.PoolAddresses[pIndex]
			}
			inp = currOutput
		}
		//found better path
		if modified {
			var err error
			newPath, err = valueobject.NewPath(data.PoolBucket, poolsOnNewPath, currPath.Tokens, currPath.Input, currPath.Output.Token, data.PriceUSDByAddress[currPath.Output.Token], data.TokenByAddress[currPath.Output.Token].Decimals, gasOption, data.SwapLimits)
			if err != nil {
				logger.Errorf(ctx, "cannot create new path, error: %s", err.Error())
				newPath = currPath
			}
			routeModified = true
		} else {
			newPath = currPath
		}
		newRoute.AddPath(data.PoolBucket, newPath, data.SwapLimits)
	}
	if routeModified {
		logger.Infof(ctx, "found better route %v. Old pool %v", newRoute, route)
	}
	return newRoute
}

func findNewBestPoolWithAmount(data findroute.FinderData, inp, out *poolpkg.TokenAmount, typeSet sets.String, poolsUsed sets.String) (string, *poolpkg.TokenAmount) {
	var bestNewPool = ""
	for _, poolVal := range data.PoolBucket.PerRequestPoolsByAddress {
		if typeSet.Has(poolVal.GetType()) && !poolsUsed.Has(poolVal.GetAddress()) {
			counter := 0
			for _, token := range poolVal.GetTokens() {
				if token == inp.Token || token == out.Token {
					counter++
				}
			}
			if counter == 2 {
				pool, avail := data.PoolBucket.GetPool(poolVal.GetAddress())
				if !avail {
					continue
				}
				swapLimit := data.SwapLimits[pool.GetType()]
				result, err := poolpkg.CalcAmountOut(pool, *inp, out.Token, swapLimit)
				if err != nil {
					continue
				}
				if result.TokenAmountOut.Amount.Cmp(out.Amount) <= 0 {
					continue
				}
				//got a better new pool, store it.
				bestNewPool = pool.GetAddress()
				out = result.TokenAmountOut
			}
		}
	}
	return bestNewPool, out
}

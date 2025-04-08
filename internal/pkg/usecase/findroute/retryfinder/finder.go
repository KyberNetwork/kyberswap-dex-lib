package retryfinder

import (
	"context"
	"math/big"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
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
		// this error will be propagated up and logged at the response, no need to log here
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
	newRoute := r.retryDynamicPools(ctx, input, baseBestRoute, []string{kyberpmm.DexTypeKyberPMM, limitorder.DexTypeLimitOrder},
		data,
		valueobject.GasOption{
			GasFeeInclude: input.GasInclude,
			Price:         input.GasPrice,
			TokenPrice:    input.GasTokenPriceUSD,
		})
	if newRoute != nil && newRoute.CompareTo(baseBestRoute, input.GasInclude) > 0 {
		logger.Debugf(ctx,
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

// retryDynamicPools will re try the found route with dynanmic pools type (i.e, pools given higher amount out ratio at higher amountin).
// it will return nil if it cannot find a better route than base route
// WARNING: finderData must be a fresh copy.
func (r *RetryFinder) retryDynamicPools(ctx context.Context, input findroute.Input, route *valueobject.Route, dynamicTypes []string, data findroute.FinderData, gasOption valueobject.GasOption) *valueobject.Route {
	var newRoute = valueobject.NewRoute(route.Input.Token, route.Output.Token)
	typeSet := sets.New(dynamicTypes...)
	poolsInRoute := route.ExtractPoolAddresses()
	routeModified := false
	for i := 0; i < len(route.Paths); i++ {
		var (
			newPath  *valueobject.Path
			currPath = route.Paths[i] //shallow copy
			inp      = &valueobject.TokenAmount{
				Token:     currPath.Input.Token,
				Amount:    big.NewInt(0).Set(currPath.Input.Amount),
				AmountUsd: currPath.Input.AmountUsd,
			}

			modified                    = false
			poolsOnNewPath              = make([]string, len(currPath.PoolAddresses))
			onGoingCalculatingGas int64 = 0
		)
		if currPath.Input.AmountAfterGas != nil {
			inp.AmountAfterGas = new(big.Int).Set(currPath.Input.AmountAfterGas)
		}
		for pIndex := 0; pIndex < len(currPath.PoolAddresses); pIndex++ {
			currPool, avail := data.PoolBucket.GetPool(currPath.PoolAddresses[pIndex])
			if !avail {
				logger.Errorf(ctx, "pool is removed from pool bucket, poolAddress: %s", currPath.PoolAddresses[pIndex])
				return route
			}
			currentOutPut, newGas, err := common.CalcNewTokenAmountAndGas(ctx, currPool, *inp, onGoingCalculatingGas, currPath.Tokens[pIndex+1], data, input, map[string]bool{})
			if err != nil {
				logger.Errorf(ctx, "cannot calculate amount out for pool %s, error: %s", currPool.GetAddress(), err)
				return route
			}

			bestNewPool, newAmount, bestNewGas := findNewBestPoolWithAmount(ctx, input, data, inp, currentOutPut, typeSet, poolsInRoute, onGoingCalculatingGas, currPath.Tokens[pIndex+1])
			onGoingCalculatingGas = newGas

			inp = currentOutPut

			if bestNewPool != "" {
				modified = true
				poolsOnNewPath[pIndex] = bestNewPool
				inp = newAmount
				poolsInRoute.Insert(bestNewPool)
				onGoingCalculatingGas = bestNewGas
			} else {
				poolsOnNewPath[pIndex] = currPath.PoolAddresses[pIndex]
			}

		}
		//found better path
		if modified {
			var err error
			newPath, err = valueobject.NewPath(ctx, data.PoolBucket, poolsOnNewPath, currPath.Tokens, currPath.Input, currPath.Output.Token, data.PriceUSDByAddress[currPath.Output.Token], data.TokenNativeBuyPrice(currPath.Output.Token), data.TokenByAddress[currPath.Output.Token].Decimals, gasOption, data.SwapLimits)
			if err != nil {
				logger.Errorf(ctx, "cannot create new path, error: %s", err.Error())
				newPath = currPath
			}
			routeModified = true
		} else {
			newPath = currPath
		}
		if err := newRoute.AddPath(ctx, data.PoolBucket, newPath, data.SwapLimits); err != nil {
			logger.Debugf(ctx, "could not add Path. Error :%s", err)
			return nil
		}

	}
	if routeModified {
		logger.Debugf(ctx, "found better route %v. Old pool %v", newRoute, route)
		return newRoute
	}
	return nil
}

func findNewBestPoolWithAmount(ctx context.Context, input findroute.Input, data findroute.FinderData, inp, currentOutPut *valueobject.TokenAmount, typeSet sets.Set[string], poolsUsed sets.Set[string], currentGas int64, tokenOut *entity.Token) (string, *valueobject.TokenAmount, int64) {
	var (
		bestNewPool       = ""
		newGas      int64 = 0
	)
	for _, poolVal := range data.PoolBucket.PerRequestPoolsByAddress {
		if typeSet.Has(poolVal.GetType()) && !poolsUsed.Has(poolVal.GetAddress()) {
			counter := 0
			for _, token := range poolVal.GetTokens() {
				if token == inp.Token || token == currentOutPut.Token {
					counter++
				}
			}
			if counter == 2 {
				pool, avail := data.PoolBucket.GetPool(poolVal.GetAddress())
				if !avail {
					continue
				}
				result, newGasAmount, err := common.CalcNewTokenAmountAndGas(ctx, pool, *inp, currentGas, tokenOut, data, input, map[string]bool{})
				if err != nil {
					continue
				}

				if result.Compare(currentOutPut, input.GasInclude) <= 0 {
					continue
				}

				//got a better new pool, store it.
				bestNewPool = pool.GetAddress()
				currentOutPut = result
				newGas = newGasAmount
			}
		}
	}
	return bestNewPool, currentOutPut, newGas
}

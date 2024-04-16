package retryfinder

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
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
	newRoute := r.retryDynamicPools(ctx, input, baseBestRoute, []string{kyberpmm.DexTypeKyberPMM, limitorder.DexTypeLimitOrder},
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

// retryDynamicPools will re try the found route with dynanmic pools type (i.e, pools given higher amount out ratio at higher amountin).
// it will return nil if it cannot find a better route than base route
// WARNING: finderData must be a fresh copy.
func (r *RetryFinder) retryDynamicPools(ctx context.Context, input findroute.Input, route *valueobject.Route, dynamicTypes []string, data findroute.FinderData, gasOption valueobject.GasOption) *valueobject.Route {
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

			modified                    = false
			poolsOnNewPath              = make([]string, len(currPath.PoolAddresses))
			onGoingCalculatingGas int64 = 0
		)
		for pIndex := 0; pIndex < len(currPath.PoolAddresses); pIndex++ {
			currPool, avail := data.PoolBucket.GetPool(currPath.PoolAddresses[pIndex])
			if !avail {
				logger.Errorf(ctx, "pool is removed from pool bucket, poolAddress: %s", currPath.PoolAddresses[pIndex])
				return route
			}
			currentOutPut, newGas, err := utils.CalcNewTokenAmountAndGas(currPool, *inp, onGoingCalculatingGas, currPath.Tokens[pIndex+1].Address, data.PriceUSDByAddress[currPath.Tokens[pIndex+1].Address], currPath.Tokens[pIndex+1].Decimals, input.GasPrice, input.GasTokenPriceUSD, data.SwapLimits[currPool.GetType()])
			//currentOutPut, err := poolpkg.CalcAmountOut(currPool, *inp, currPath.Tokens[pIndex+1].Address, data.SwapLimits[currPool.GetType()])
			if err != nil {
				logger.Errorf(ctx, "cannot calculate amount out for pool %s, error: %s", currPool.GetAddress(), err)
				return route
			}

			bestNewPool, newAmount, bestNewGas := findNewBestPoolWithAmount(input, data, inp, currentOutPut, typeSet, poolsInRoute, onGoingCalculatingGas, data.PriceUSDByAddress[currentOutPut.Token], currPath.Tokens[pIndex+1])
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
			newPath, err = valueobject.NewPath(data.PoolBucket, poolsOnNewPath, currPath.Tokens, currPath.Input, currPath.Output.Token, data.PriceUSDByAddress[currPath.Output.Token], data.TokenNativeBuyPrice(currPath.Output.Token), data.TokenByAddress[currPath.Output.Token].Decimals, gasOption, data.SwapLimits)
			if err != nil {
				logger.Errorf(ctx, "cannot create new path, error: %s", err.Error())
				newPath = currPath
			}
			routeModified = true
		} else {
			newPath = currPath
		}
		if err := newRoute.AddPath(data.PoolBucket, newPath, data.SwapLimits); err != nil {
			logger.Debugf(ctx, "could not add Path. Error :%s", err)
			return nil
		}

	}
	if routeModified {
		logger.Infof(ctx, "found better route %v. Old pool %v", newRoute, route)
		return newRoute
	}
	return nil
}

func betterAmountOut(newAmount, oldAmount *poolpkg.TokenAmount, gasInclude bool) bool {
	if gasInclude && !utils.Float64AlmostEqual(newAmount.AmountUsd, oldAmount.AmountUsd) {
		return newAmount.AmountUsd > oldAmount.AmountUsd
	}
	if newAmount.Amount.Cmp(oldAmount.Amount) > 0 {
		return true
	}
	return false
}

func findNewBestPoolWithAmount(input findroute.Input, data findroute.FinderData, inp, currentOutPut *poolpkg.TokenAmount, typeSet sets.String, poolsUsed sets.String, currentGas int64, tokenOutPrice float64, tokenOut *entity.Token) (string, *poolpkg.TokenAmount, int64) {
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
				swapLimit := data.SwapLimits[pool.GetType()]
				result, newGasAmount, err := utils.CalcNewTokenAmountAndGas(pool, *inp, currentGas, currentOutPut.Token, tokenOutPrice, tokenOut.Decimals, input.GasPrice, input.GasTokenPriceUSD, swapLimit)
				if err != nil {
					continue
				}

				if !betterAmountOut(result, currentOutPut, input.GasInclude) {
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

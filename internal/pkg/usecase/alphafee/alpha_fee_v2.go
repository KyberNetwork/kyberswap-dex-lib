package alphafee

import (
	"context"
	"math"
	"math/big"
	"runtime/debug"

	"slices"

	"github.com/KyberNetwork/kutils/klog"
	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	routerValueObject "github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/pkg/errors"
)

type AlphaFeeV2Calculation struct {
	reductionFactorInBps *big.Int
	config               routerValueObject.AlphaFeeConfig
	entity.ICustomFuncsHolder
}

func NewAlphaFeeV2Calculation(
	config routerValueObject.AlphaFeeConfig,
	customFuncs entity.ICustomFuncs,
) *AlphaFeeV2Calculation {
	// For phase 2, take KyberPMM as default value for reductionFactorInBps.
	// If the config is not set for KyberPMM, we will take the first value in the map.
	reductionFactorInBps := DefaultReductionFactorInBps
	if number, ok := config.ReductionConfig.ReductionFactorInBps[string(valueobject.ExchangeKyberPMM)]; ok {
		reductionFactorInBps = big.NewInt(int64(number))
	} else {
		for _, number := range config.ReductionConfig.ReductionFactorInBps {
			reductionFactorInBps = big.NewInt(int64(number))
			break
		}
	}

	return &AlphaFeeV2Calculation{
		reductionFactorInBps: reductionFactorInBps,
		config:               config,
		ICustomFuncsHolder:   &entity.CustomFuncsHolder{ICustomFuncs: customFuncs},
	}
}

func (c *AlphaFeeV2Calculation) Calculate(ctx context.Context, param AlphaFeeParams) (*routerEntity.AlphaFeeV2, error) {
	routeInfo := convertConstructRouteToRouteInfoV2(ctx, param.BestRoute, param.PoolSimulatorBucket)

	if !c.isRouteContainsAlphaFeeSource(ctx, routeInfo) {
		return nil, ErrRouteNotHaveAlphaFeeDex
	}

	if param.BestAmmRoute != nil && c.notMuchBetter(param.BestRoute, param.BestAmmRoute, true) {
		return nil, errors.WithMessage(ErrAlphaFeeNotExists, "amm route is almost equal with best route")
	}

	ammBestRouteAmountOut := c.getAMMBestRouteAmountOut(ctx, param)

	var reductionDelta big.Int
	reductionDelta.Sub(param.BestRoute.AmountOut, ammBestRouteAmountOut)
	if reductionDelta.Sign() <= 0 {
		return nil, errors.WithMessagef(
			ErrAlphaFeeNotExists,
			"reductionDelta is negative %v, bestAmount %s, ammRoute %v",
			reductionDelta, param.BestRoute.AmountOut, ammBestRouteAmountOut,
		)
	}

	// With phase 2.5, we use full diff between best route and AMM route,
	// instead of just reductionDelta * c.reductionFactorInBps.
	// We will apply the custom factor for each alpha fee source later.
	pathReductions := c.getReductionPerPath(ctx, routeInfo, &reductionDelta)

	swapReductions, err := c.getReductionPerSwap(ctx, param, pathReductions, routeInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to apply reduction on paths")
	}

	return &routerEntity.AlphaFeeV2{
		AMMAmount:      ammBestRouteAmountOut,
		SwapReductions: swapReductions,
	}, nil
}

func (c *AlphaFeeV2Calculation) CalculateDefaultAlphaFee(ctx context.Context, param DefaultAlphaFeeParams) (*routerEntity.AlphaFeeV2, error) {
	routeInfo := convertRouteSummaryToRouteInfoV2(ctx, param.RouteSummary)

	if !c.isRouteContainsAlphaFeeSource(ctx, routeInfo) {
		return nil, ErrRouteNotHaveAlphaFeeDex
	}

	var alphaFee big.Int

	alphaFee.Div(
		alphaFee.Mul(
			param.RouteSummary.AmountOut,
			alphaFee.SetInt64(int64(c.config.ReductionConfig.DefaultAlphaFeePercentageBps)),
		),
		valueobject.BasisPoint,
	)

	if !isMergeSwapRoute(param.RouteSummary) {
		return c.calculateDefaultAlphaFeeNonMergeRoute(ctx, param, &alphaFee)
	}
	return c.calculateDefaultAlphaFeeMergeRoute(ctx, param, &alphaFee)
}

func (c *AlphaFeeV2Calculation) notMuchBetter(bestRoute, bestAMMRoute *finderCommon.ConstructRoute, gasIncluded bool) bool {
	if bestPrice, ammPrice := bestRoute.AmountOutPrice, bestAMMRoute.AmountOutPrice; bestPrice != 0 || ammPrice != 0 {
		if gasIncluded {
			bestPrice -= bestRoute.GasFeePrice + bestRoute.L1GasFeePrice
			ammPrice -= bestAMMRoute.GasFeePrice + bestAMMRoute.L1GasFeePrice
		}
		return bestPrice-ammPrice <= c.config.ReductionConfig.MinDifferentThresholdUSD
	}

	var diff big.Int
	diff.Sub(bestRoute.AmountOut, bestAMMRoute.AmountOut)
	diff.Mul(&diff, valueobject.BasisPoint).Div(&diff, bestAMMRoute.AmountOut)
	return diff.Int64() <= c.config.ReductionConfig.MinDifferentThresholdBps
}

func (c *AlphaFeeV2Calculation) isRouteContainsAlphaFeeSource(_ context.Context, routeInfo [][]swapInfoV2) bool {
	return slices.ContainsFunc(routeInfo, isPathContainsAlphaFeeSources)
}

func (c *AlphaFeeV2Calculation) getAMMBestRouteAmountOut(_ context.Context, param AlphaFeeParams) *big.Int {
	ammBestRouteAmountOut := new(big.Int)
	if param.BestAmmRoute != nil {
		ammBestRouteAmountOut.Set(param.BestAmmRoute.AmountOut)
	}

	// If the AMM best route is unavailable, or if it returns an exceedingly small amount,
	// we set the AMM best route to `maxReducedAmountOut`, which is a portion of the bestRoute.AmountOut.
	var maxReducedAmountOut big.Int
	maxReducedAmountOut.Div(
		maxReducedAmountOut.Mul(
			param.BestRoute.AmountOut,
			maxReducedAmountOut.SetInt64(c.config.ReductionConfig.MaxThresholdPercentageInBps),
		),
		valueobject.BasisPoint,
	)

	if ammBestRouteAmountOut.Cmp(&maxReducedAmountOut) < 0 {
		ammBestRouteAmountOut.Set(&maxReducedAmountOut)
	}

	return ammBestRouteAmountOut
}

func (c *AlphaFeeV2Calculation) getReductionPerPath(
	_ context.Context, routeInfo [][]swapInfoV2, alphaFee *big.Int,
) []pathReduction {
	numberOfAlphaFeePaths := 0
	var totalAmountOutOfAlphaFeeSources big.Int

	for _, path := range routeInfo {
		if isPathContainsAlphaFeeSources(path) {
			numberOfAlphaFeePaths++
			pathAmountOut := path[len(path)-1].AmountOut
			totalAmountOutOfAlphaFeeSources.Add(
				&totalAmountOutOfAlphaFeeSources,
				pathAmountOut,
			)
		}
	}

	var tmp big.Int
	pathReductions := make([]pathReduction, 0, numberOfAlphaFeePaths)
	for idx, path := range routeInfo {
		if isPathContainsAlphaFeeSources(path) {
			pathAmountOut := path[len(path)-1].AmountOut
			reduceAmount := new(big.Int).Div(
				tmp.Mul(alphaFee, pathAmountOut),
				&totalAmountOutOfAlphaFeeSources,
			)

			// Cap reduceAmount at `c.reductionFactorInBps`
			// if the reduceAmount greater than the current path's amount out.
			if reduceAmount.Cmp(pathAmountOut) > 0 {
				reduceAmount.Set(
					tmp.Div(
						tmp.Mul(pathAmountOut, c.reductionFactorInBps),
						valueobject.BasisPoint,
					),
				)
			}

			pathReduction := pathReduction{
				PathIdx:      idx,
				ReduceAmount: reduceAmount,
			}

			pathReductions = append(pathReductions, pathReduction)
		}
	}

	return pathReductions
}

// getReductionPerSwap calculates & returns the reduction for each swap in a path.
// It does not modify the param.BestRoute.
func (c *AlphaFeeV2Calculation) getReductionPerSwap(
	ctx context.Context,
	param AlphaFeeParams,
	pathReductions []pathReduction,
	routeInfo [][]swapInfoV2,
) (swapReductions []routerEntity.AlphaFeeV2SwapReduction, err error) {
	swapReductions = make([]routerEntity.AlphaFeeV2SwapReduction, 0)

	// Handle recovery from panic in UpdateBalance.
	var currentPool string
	defer func() {
		if r := recover(); r != nil {
			swapReductions = nil
			err = errors.WithMessagef(finderCommon.ErrPanicRefreshPath, "%v", r)
			klog.Warnf(ctx, "alphaFeeV2|refreshPath|%s panicked: %s", currentPool, string(debug.Stack()))
		}
	}()

	// NOTE: param.PoolSimulatorBucket should be a fresh instance and not shared,
	// as it will be modified.
	simulatorBucket := param.PoolSimulatorBucket

	pointer := 0 // to track the pathReductions
	executedId := 0
	for idx, path := range param.BestRoute.Paths {
		pathInfo := routeInfo[idx]
		pathContainsAlphaFeeSources := isPathContainsAlphaFeeSources(pathInfo)
		var reductionPercentWithAllFee float64
		var numOfAlphaFeeSources int
		if pathContainsAlphaFeeSources {
			// Calculate the percentage of amount out reduction of
			// each alpha fee source in the path
			if numOfAlphaFeeSources == 0 {
				numOfAlphaFeeSources = countAlphaFeeSourcesInPath(pathInfo)
			}
			pathReduction := pathReductions[pointer].ReduceAmount
			pointer++

			pathReductionF, _ := pathReduction.Float64()
			pathAmountOutF, _ := path.AmountOut.Float64()
			alphaFeePercent := pathReductionF / pathAmountOutF

			// We have `amountIn`, rate through `p1, p2... pn`, k alpha fee sources in the path (k <= n).
			// We want to apply a `reductionPercent` for each alpha fee source swap,
			// that the final amount out will be reduce by `pathReduction`.
			// We have:
			// - amountIn * p1 * ... * pn = pathAmountOut (1)
			// - amountIn * p1 * ... * pn * reductionPercent ** k = pathAmountOut - pathReduction (2)
			// => (amountIn * p1 * ... * pn) * (1 - reductionPercent ** k) = pathReduction ((1) - (2))
			// => pathAmountOut * (1 - reductionRate ** k) = pathReduction
			// => reductionPercent ** k = 1 - pathReduction / pathAmountOut = 1 - alphaFeePercent
			// => reductionPercent = (1 - alphaFeePercent) ** (1 / k)
			reductionPercentWithAllFee = math.Pow(1-alphaFeePercent, 1/float64(numOfAlphaFeeSources))
		}

		// Update the path amount out and the path reduction.
		// Even if the path does not contain alpha fee sources,
		// we still need to refresh the path amount out,
		// to update the pool state in simulatorBucket for the next path.
		var currentAmountIn big.Int
		currentAmountIn.Set(path.AmountIn)
		for i, poolId := range path.PoolsOrder {
			fromToken := path.TokensOrder[i]
			toToken := path.TokensOrder[i+1]

			pool := simulatorBucket.GetPool(poolId)
			swapLimit := simulatorBucket.GetPoolSwapLimit(poolId)
			currentPool = pool.GetExchange() + "/" + poolId // For tracking panic refresh path

			tokenAmountIn := dexlibPool.TokenAmount{Token: fromToken, Amount: &currentAmountIn}
			res, err := c.CalcAmountOut(ctx, pool, tokenAmountIn, toToken, swapLimit)
			if err != nil {
				return nil, err
			} else if !res.IsValid() {
				return nil, finderCommon.ErrCalcAmountOutEmpty
			}

			var currentAmountOut big.Int
			currentAmountOut.Set(res.TokenAmountOut.Amount)

			if privo.IsAlphaFeeSource(pool.GetExchange()) {
				currentAmountOutF, _ := currentAmountOut.Float64()

				// reductionPercentWithSourceFactor = (1 - (1 - reductionPercentWithAllFee ** numOfSources) * reductionFactorInBps) ** (1/numOfSources)
				sourceReductionFactorF, ok := c.config.ReductionConfig.ReductionFactorInBps[pool.GetExchange()]
				if !ok {
					sourceReductionFactorF, _ = c.reductionFactorInBps.Float64()
				}

				reductionPercentWithSourceFactor := 1 - math.Pow(reductionPercentWithAllFee, float64(numOfAlphaFeeSources))
				reductionPercentWithSourceFactor = reductionPercentWithSourceFactor * sourceReductionFactorF / basisPointFloat
				reductionPercentWithSourceFactor = math.Pow(1-reductionPercentWithSourceFactor, 1/float64(numOfAlphaFeeSources))

				currentAmountOutF = currentAmountOutF * reductionPercentWithSourceFactor

				new(big.Float).SetFloat64(currentAmountOutF).Int(&currentAmountOut)

				reducedAmount := new(big.Int).Sub(res.TokenAmountOut.Amount, &currentAmountOut)
				reducedAmountUsd := c.GetFairPrice(
					ctx, fromToken, toToken,
					param.Prices[fromToken], param.Prices[toToken],
					param.Tokens[fromToken].Decimals, param.Tokens[toToken].Decimals,
					&currentAmountIn, &currentAmountOut, reducedAmount,
				)
				swapReductions = append(swapReductions, routerEntity.AlphaFeeV2SwapReduction{
					ExecutedId:      executedId,
					PoolAddress:     poolId,
					TokenIn:         fromToken,
					TokenOut:        toToken,
					ReduceAmount:    reducedAmount,
					ReduceAmountUsd: reducedAmountUsd,
				})
			}

			// No need to BackupPool here, since if we fails,
			// we stop the process and return the error.
			pool = simulatorBucket.ClonePoolById(ctx, poolId)
			swapLimit = simulatorBucket.CloneSwapLimitById(ctx, poolId)

			updateBalanceParams := dexlibPool.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *res.TokenAmountOut,
				Fee:            *res.Fee,
				SwapInfo:       res.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool.UpdateBalance(updateBalanceParams)

			currentAmountIn.Set(&currentAmountOut)
			executedId++
		}
	}

	return swapReductions, nil
}

func (c *AlphaFeeV2Calculation) GetFairPrice(
	_ context.Context,
	tokenIn, tokenOut string,
	tokenInPrice, tokenOutPrice float64,
	tokenInDecimals, tokenOutDecimals uint8,
	amountIn, amountOut, alphaFeeAmount *big.Int,
) float64 {
	isTokenInWhitelistPrices := c.config.WhitelistPrices[tokenIn]
	isTokenOutWhitelistPrice := c.config.WhitelistPrices[tokenOut]

	if isTokenOutWhitelistPrice || !isTokenInWhitelistPrices {
		return finderUtil.CalcAmountPrice(
			alphaFeeAmount,
			tokenOutDecimals,
			tokenOutPrice,
		)
	}

	amountInF, _ := amountIn.Float64()
	amountInF = amountInF / math.Pow(10, float64(tokenInDecimals))
	amountOutF, _ := amountOut.Float64()
	amountOutF = amountOutF / math.Pow(10, float64(tokenOutDecimals))
	tokenOutFairPrice := tokenInPrice * amountInF / amountOutF

	return finderUtil.CalcAmountPrice(
		alphaFeeAmount,
		tokenOutDecimals,
		tokenOutFairPrice,
	)
}

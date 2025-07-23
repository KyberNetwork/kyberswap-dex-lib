package alphafee

import (
	"context"
	"math"
	"math/big"
	"runtime/debug"
	"slices"

	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	routerValueObject "github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	DefaultReductionFactor  = 5000
	DefaultWeightDistribute = 1000
)

type AlphaFeeV3Calculation struct {
	v2Calculation *AlphaFeeV2Calculation
	config        routerValueObject.AlphaFeeConfig
	tokenGroups   *routerValueObject.TokenGroupConfig
	entity.ICustomFuncsHolder
}

func NewAlphaFeeV3Calculation(
	v2Calculation *AlphaFeeV2Calculation,
	config routerValueObject.AlphaFeeConfig,
	tokenGroupsConfig *routerValueObject.TokenGroupConfig,
	customFuncs entity.ICustomFuncs,
) *AlphaFeeV3Calculation {
	return &AlphaFeeV3Calculation{
		v2Calculation:      v2Calculation,
		config:             config,
		tokenGroups:        tokenGroupsConfig,
		ICustomFuncsHolder: &entity.CustomFuncsHolder{ICustomFuncs: customFuncs},
	}
}

func (c *AlphaFeeV3Calculation) Calculate(ctx context.Context, param AlphaFeeParams) (*routerEntity.AlphaFeeV2, error) {
	routeInfo := convertConstructRouteToRouteInfoV2(ctx, param.BestRoute, param.PoolSimulatorBucket)

	if !c.isRouteContainsAlphaFeeSource(ctx, routeInfo) {
		return nil, ErrRouteNotHaveAlphaFeeDex
	}

	if param.BestAmmRoute != nil && c.notMuchBetter(param.BestRoute, param.BestAmmRoute, true) {
		return nil, errors.WithMessage(ErrAlphaFeeNotExists, "amm route is almost equal with best route")
	}

	ammBestRouteAmountOut := c.getAMMBestRouteAmountOut(ctx, param)

	pathReductions := c.getReductionPerPath(ctx, param, routeInfo, ammBestRouteAmountOut)
	if len(pathReductions) == 0 {
		return nil, errors.WithMessage(ErrAlphaFeeNotExists, "empty path reductions")
	}

	swapReductions, err := c.getReductionPerSwap(ctx, param, pathReductions, routeInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to apply reduction on paths")
	}

	return &routerEntity.AlphaFeeV2{
		AMMAmount:      ammBestRouteAmountOut,
		SwapReductions: swapReductions,
	}, nil
}

func (c *AlphaFeeV3Calculation) CalculateDefaultAlphaFee(ctx context.Context,
	param DefaultAlphaFeeParams) (*routerEntity.AlphaFeeV2, error) {
	return c.v2Calculation.CalculateDefaultAlphaFee(ctx, param)
}

func (c *AlphaFeeV3Calculation) getReductionPerPath(
	ctx context.Context,
	param AlphaFeeParams,
	routeInfo [][]swapInfoV2,
	ammBestRouteAmountOut *big.Int,
) []pathReduction {
	var amountOutDiff big.Int
	amountOutDiff.Sub(param.BestRoute.AmountOut, ammBestRouteAmountOut)
	amountOutDiffF, _ := amountOutDiff.Float64()

	pathExchangeRates := c.getPathExchangeRate(ctx, routeInfo)

	// Path with best rate will be at the start of the slice
	slices.SortFunc(pathExchangeRates, func(a, b pathExchangeRate) int {
		if a.PathAmountOutF*b.PathAmountInF >= a.PathAmountInF*b.PathAmountOutF {
			return -1
		}
		return 1
	})

	surpluses := make([]pathReduction, 0, len(pathExchangeRates))

	var curAmountInF, totalSurplus float64

	for idx, path := range pathExchangeRates {
		var nextRate float64
		if idx < len(pathExchangeRates)-1 {
			nextPath := pathExchangeRates[idx+1]
			nextRate = nextPath.PathAmountOutF / nextPath.PathAmountInF
		}

		curRate := path.PathAmountOutF / path.PathAmountInF

		curAmountInF += path.PathAmountInF
		surplus := max(curAmountInF*(curRate-nextRate), 0)

		if totalSurplus+surplus > amountOutDiffF {
			surplus = amountOutDiffF - totalSurplus
		}

		totalSurplus += surplus

		surplusF := surplus * path.PathAmountInF / curAmountInF
		surplusBI, _ := new(big.Float).SetFloat64(surplusF).Int(nil)

		// We will also add surplus = 0 to the `surpluses` slice,
		// since some paths can have the same rate. We will filter them out later.
		surpluses = append(surpluses, pathReduction{
			PathIdx:      path.PathIdx,
			ReduceAmount: surplusBI,
		})
		for i := idx - 1; i >= 0; i-- {
			surplusF := surplus * pathExchangeRates[i].PathAmountInF / curAmountInF
			surplusBI, _ := new(big.Float).SetFloat64(surplusF).Int(nil)
			surpluses[i].ReduceAmount.Add(surpluses[i].ReduceAmount, surplusBI)
		}
	}

	surpluses = lo.Filter(surpluses, func(surplus pathReduction, _ int) bool {
		return surplus.ReduceAmount.Sign() > 0
	})

	// Sort surpluses by path index, since later functions expect them to be sorted.
	slices.SortFunc(surpluses, func(a, b pathReduction) int {
		return a.PathIdx - b.PathIdx
	})

	return surpluses
}

func (c *AlphaFeeV3Calculation) getReductionPerSwap(
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
			log.Ctx(ctx).Warn().
				Str("pool", currentPool).Bytes("stack", debug.Stack()).Msg("alphaFeeV3|refreshPath panicked")
		}
	}()

	// NOTE: param.PoolSimulatorBucket should be a fresh instance and not shared,
	// as it will be modified.
	simulatorBucket := param.PoolSimulatorBucket

	pointer := 0 // to track the pathReductions
	executedId := 0
	for idx, path := range param.BestRoute.Paths {
		pathInfo := routeInfo[idx]
		shouldTakeAlphaFeeInPath := isPathContainsAlphaFeeSources(pathInfo) &&
			pointer < len(pathReductions) &&
			pathReductions[pointer].PathIdx == idx

		var pathSurplusRate, totalWeight float64
		if shouldTakeAlphaFeeInPath {
			surplusAmountF, _ := pathReductions[pointer].ReduceAmount.Float64()
			pathAmountOutF, _ := path.AmountOut.Float64()
			pathSurplusRate = surplusAmountF / pathAmountOutF

			for _, swapInfo := range pathInfo {
				if privo.IsAlphaFeeSource(swapInfo.Exchange) {
					weight := c.getWeightDistribute(simulatorBucket.GetPool(swapInfo.Pool), swapInfo.TokenIn,
						swapInfo.TokenOut)
					totalWeight += float64(weight)
				}
			}
			pointer++
		}

		// Update the path amount out and the path reduction.
		// Even if the path does not contain alpha fee sources,
		// we still need to refresh the path amount out,
		// to update the pool state in simulatorBucket for the next path.
		currentAmountIn := path.AmountIn
		for i, poolId := range path.PoolsOrder {
			fromToken := path.TokensOrder[i]
			toToken := path.TokensOrder[i+1]

			pool := simulatorBucket.GetPool(poolId)
			swapLimit := simulatorBucket.GetPoolSwapLimit(poolId)
			currentPool = pool.GetExchange() + "/" + poolId // For tracking panic refresh path

			tokenAmountIn := dexlibPool.TokenAmount{Token: fromToken, Amount: currentAmountIn}
			res, err := c.CalcAmountOut(ctx, pool, tokenAmountIn, toToken, swapLimit)
			if err != nil {
				return nil, err
			} else if !res.IsValid() {
				return nil, finderCommon.ErrCalcAmountOutEmpty
			}

			currentAmountOut := res.TokenAmountOut.Amount

			if shouldTakeAlphaFeeInPath && privo.IsAlphaFeeSource(pool.GetExchange()) {
				currentAmountOutF, _ := currentAmountOut.Float64()
				currentAmountOutUsd := c.GetFairPrice(
					ctx, fromToken, toToken,
					param.Prices[fromToken], param.Prices[toToken],
					param.Tokens[fromToken].Decimals, param.Tokens[toToken].Decimals,
					currentAmountIn, currentAmountOut, currentAmountOut,
				)

				// https://www.notion.so/kybernetwork/Alpha-Fee-Phase-3-20026751887e809b9d53f6937378078c
				weight := c.getWeightDistribute(pool, fromToken, toToken)
				surplusRate := 1 - math.Exp(math.Log(1-pathSurplusRate)*float64(weight)/totalWeight)
				poolSurplus := currentAmountOutF * surplusRate

				reductionFactor := c.getReductionFactor(pool)
				reduceAmountF := poolSurplus * reductionFactor / basisPointFloat
				reduceAmountUsd := currentAmountOutUsd * reduceAmountF / currentAmountOutF

				if surplusAllowanceUsd := c.getSurplusAllowanceUsd(pool); surplusAllowanceUsd > 0 {
					minReduceAmountUsd := reduceAmountUsd*basisPointFloat/reductionFactor - surplusAllowanceUsd
					if reduceAmountUsd < minReduceAmountUsd {
						reduceAmountUsd = minReduceAmountUsd
						reduceAmountF = currentAmountOutF * reduceAmountUsd / currentAmountOutUsd
					}
				}

				var reduceAmountBF big.Float
				reduceAmount, _ := reduceAmountBF.SetFloat64(reduceAmountF).Int(nil)
				swapReductions = append(swapReductions, routerEntity.AlphaFeeV2SwapReduction{
					ExecutedId:      executedId,
					PoolAddress:     poolId,
					TokenIn:         fromToken,
					TokenOut:        toToken,
					ReduceAmount:    reduceAmount,
					ReduceAmountUsd: reduceAmountUsd,
				})

				currentAmountOut.Sub(currentAmountOut, reduceAmount)
			}

			// No need to BackupPool here, since if we fail,
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

			currentAmountIn = currentAmountOut
			executedId++
		}
	}

	return swapReductions, nil
}

func (c *AlphaFeeV3Calculation) notMuchBetter(bestRoute, bestAMMRoute *finderCommon.ConstructRoute,
	gasIncluded bool) bool {
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

func (c *AlphaFeeV3Calculation) isRouteContainsAlphaFeeSource(_ context.Context, routeInfo [][]swapInfoV2) bool {
	return slices.ContainsFunc(routeInfo, isPathContainsAlphaFeeSources)
}

func (c *AlphaFeeV3Calculation) getAMMBestRouteAmountOut(_ context.Context, param AlphaFeeParams) *big.Int {
	ammBestRouteAmountOut := new(big.Int)
	if param.BestAmmRoute != nil {
		ammBestRouteAmountOut.Set(param.BestAmmRoute.AmountOut)
	}

	// If the AMM best route is unavailable, or if it returns an exceedingly tiny amount,
	// we set the AMM best route to `maxReduceAmountOut`, which is a portion of the bestRoute.AmountOut.
	var maxReduceAmountOut big.Int
	maxReduceAmountOut.Div(
		maxReduceAmountOut.Mul(
			param.BestRoute.AmountOut,
			maxReduceAmountOut.SetInt64(c.config.ReductionConfig.MaxThresholdPercentageInBps),
		),
		valueobject.BasisPoint,
	)

	if ammBestRouteAmountOut.Cmp(&maxReduceAmountOut) < 0 {
		ammBestRouteAmountOut.Set(&maxReduceAmountOut)
	}

	return ammBestRouteAmountOut
}

func (c *AlphaFeeV3Calculation) GetFairPrice(
	_ context.Context,
	tokenIn, tokenOut string,
	tokenInPrice, tokenOutPrice float64,
	tokenInDecimals, tokenOutDecimals uint8,
	amountIn, amountOut, specifiedAmount *big.Int,
) float64 {
	isTokenInWhitelistPrices := c.config.WhitelistPrices[tokenIn]
	isTokenOutWhitelistPrice := c.config.WhitelistPrices[tokenOut]

	if isTokenOutWhitelistPrice || !isTokenInWhitelistPrices {
		return finderUtil.CalcAmountPrice(
			specifiedAmount,
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
		specifiedAmount,
		tokenOutDecimals,
		tokenOutFairPrice,
	)
}

// getReductionFactor returns the reduction factor of a surplus for a given pool.
func (c *AlphaFeeV3Calculation) getReductionFactor(pool dexlibPool.IPoolSimulator) float64 {
	if reductionFactor, ok := c.config.ReductionConfig.ReductionFactorByPool[pool.GetAddress()]; ok {
		return reductionFactor
	} else if reductionFactor, ok = c.config.ReductionConfig.ReductionFactorInBps[pool.GetExchange()]; ok {
		return reductionFactor
	}
	return DefaultReductionFactor
}

// getSurplusAllowanceUsd returns the maximum surplus allowance in USD of a pool. Negative value means no limit.
func (c *AlphaFeeV3Calculation) getSurplusAllowanceUsd(pool dexlibPool.IPoolSimulator) float64 {
	if surplusAllowanceUsd, ok := c.config.ReductionConfig.SurplusAllowanceUsdByPool[pool.GetAddress()]; ok {
		return surplusAllowanceUsd
	} else if surplusAllowanceUsd, ok = c.config.ReductionConfig.SurplusAllowanceUsd[pool.GetExchange()]; ok {
		return surplusAllowanceUsd
	}
	return -1
}

func (c *AlphaFeeV3Calculation) getWeightDistribute(pool dexlibPool.IPoolSimulator, tokenIn, tokenOut string) int {
	if c.config.ReductionConfig.WeightDistributeByPool != nil {
		if weightDistribute, ok := c.config.ReductionConfig.WeightDistributeByPool[pool.GetAddress()]; ok {
			return weightDistribute
		}
	}

	if c.tokenGroups != nil {
		tokenGroupParams := routerValueObject.TokenGroupParams{
			TokenIn:  tokenIn,
			TokenOut: tokenOut,
			Exchange: pool.GetExchange(),
		}

		if groupType, ok := c.tokenGroups.GetTokenGroupType(tokenGroupParams); ok {
			if c.config.ReductionConfig.WeightDistributeByTokenGroup != nil {
				if weightDistribute, ok := c.config.ReductionConfig.WeightDistributeByTokenGroup[groupType]; ok {
					return weightDistribute
				}
			}
		}
	}

	if c.config.ReductionConfig.WeightDistributeBySource != nil {
		if weightDistribute, ok := c.config.ReductionConfig.WeightDistributeBySource[pool.GetExchange()]; ok {
			return weightDistribute
		}
	}

	return DefaultWeightDistribute
}

func (c *AlphaFeeV3Calculation) getPathExchangeRate(
	ctx context.Context,
	routeInfo [][]swapInfoV2,
) []pathExchangeRate {
	// Some alpha fee paths share a same pool.
	// Due to slippage, the rate of the first path can be better than the second path,
	// causing the second path to have smaller/empty surplus.
	// To avoid this, among alpha fee paths, we will merge the swaps with the same pool,
	// and calculate the rate of each pool by its average rate among those swaps.
	var rateSharePools = make(map[string]amountThroughPool)

	if c.config.ReductionConfig.CalculateSurplusMergeSharePools {
		for _, path := range routeInfo {
			if isPathContainsAlphaFeeSources(path) {
				for _, swapInfo := range path {
					key := swapInfo.Pool + "-" + swapInfo.TokenIn + "-" + swapInfo.TokenOut
					if _, exist := rateSharePools[key]; !exist {
						rateSharePools[key] = amountThroughPool{
							TotalAmountIn:  new(big.Int),
							TotalAmountOut: new(big.Int),
							Count:          0,
						}
					}

					val := rateSharePools[key]
					val.TotalAmountIn.Add(val.TotalAmountIn, swapInfo.AmountIn)
					val.TotalAmountOut.Add(val.TotalAmountOut, swapInfo.AmountOut)
					val.Count++
					rateSharePools[key] = val
				}
			}
		}
	}

	var pathExchangeRates []pathExchangeRate
	for idx, path := range routeInfo {
		if isPathContainsAlphaFeeSources(path) {
			pathAmountInF, _ := path[0].AmountIn.Float64()
			defaultPathAmountOutF, _ := path[len(path)-1].AmountOut.Float64()
			pathAmountOutF := pathAmountInF

			shouldRecalculatePathAmountOutWithMergePool := c.config.ReductionConfig.CalculateSurplusMergeSharePools &&
				lo.SomeBy(path, func(swapInfo swapInfoV2) bool {
					key := swapInfo.Pool + "-" + swapInfo.TokenIn + "-" + swapInfo.TokenOut
					val, exist := rateSharePools[key]
					if !exist || val.Count <= 1 {
						return false
					}
					return true
				})

			if shouldRecalculatePathAmountOutWithMergePool {
				for _, swapInfo := range path {
					key := swapInfo.Pool + "-" + swapInfo.TokenIn + "-" + swapInfo.TokenOut
					ratePool, exist := rateSharePools[key]
					if !exist {
						log.Ctx(ctx).Warn().
							Str("key", key).Msg("alphaFeeV3|getPathExchangeRate|key not found in rateSharePools")
						pathAmountOutF = defaultPathAmountOutF
						break
					}

					ratePoolAmountInF, _ := ratePool.TotalAmountIn.Float64()
					ratePoolAmountOutF, _ := ratePool.TotalAmountOut.Float64()
					if utils.Float64AlmostEqual(ratePoolAmountOutF, 0) {
						log.Ctx(ctx).Warn().
							Str("key", key).Msg("alphaFeeV3|getPathExchangeRate|key has zero amount out")
						pathAmountOutF = defaultPathAmountOutF
						break
					}

					pathAmountOutF *= ratePoolAmountOutF / ratePoolAmountInF
				}
			} else {
				pathAmountOutF = defaultPathAmountOutF
			}

			pathExchangeRates = append(pathExchangeRates, pathExchangeRate{
				PathIdx:        idx,
				PathAmountInF:  pathAmountInF,
				PathAmountOutF: pathAmountOutF,
			})
		}
	}

	return pathExchangeRates
}

package alphafee

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	dexlibEntity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	"github.com/samber/lo"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	routerValueObject "github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrInvalidSwap                         = errors.New("invalid swap")
	ErrCalcAmountOutEmpty                  = errors.New("calc amount out empty")
	ErrAlphaFeeNotExists                   = errors.New("alpha fee doesn't exists")
	ErrRouteNotHaveAlphaFeeDex             = errors.New("route doesn't have alpha-able swaps")
	ErrAlphaSwapNotEnoughToCoverAlphaFee   = errors.New("alpha swap doesn't have enough amount out to cover alpha fee")
	ErrApplyAlphaFeeYieldLessAmountThanAMM = errors.New("applying alpha fee yields less amount out than amm route")

	DefaultReductionFactorInBps = big.NewInt(5000)
)

type SwapIndex struct {
	PathId     int
	SwapId     int
	ExecutedId int32
}

type AlphaFeeParams struct {
	BestRoute    *finderCommon.ConstructRoute
	BestAmmRoute *finderCommon.ConstructRoute

	Prices              map[string]float64
	Tokens              map[string]dexlibEntity.SimplifiedToken
	PoolSimulatorBucket *finderCommon.SimulatorBucket
}

type DefaultAlphaFeeParams struct {
	RouteSummary routerValueObject.RouteSummary
}

type SwapInfo struct {
	pool     string
	exchange valueobject.Exchange
}

type PathInfo struct {
	SwapInfo []SwapInfo
}

type AlphaFeeCalculation struct {
	// Config alpha Fee rate using percentage in BPS, the same as safety quoting, 1 bps = 0.01%
	// Convert deductionFactor from float to integer by multiply it by 10, then we will div (BasisPoint * 10)
	ReductionFactorInBps map[string]*big.Int
	config               routerValueObject.AlphaFeeConfig
	entity.ICustomFuncsHolder
}

func NewAlphaFeeCalculation(
	config routerValueObject.AlphaFeeConfig,
	customFuncs entity.ICustomFuncs) *AlphaFeeCalculation {
	factors := map[string]*big.Int{}
	for dex, number := range config.ReductionConfig.ReductionFactorInBps {
		factors[dex] = big.NewInt(int64(number * 10))
	}
	return &AlphaFeeCalculation{
		ReductionFactorInBps: factors,
		ICustomFuncsHolder:   &entity.CustomFuncsHolder{ICustomFuncs: customFuncs},
		config:               config,
	}
}

func (c *AlphaFeeCalculation) Calculate(ctx context.Context, param AlphaFeeParams) (*routerEntity.AlphaFee, error) {
	swapIndex := c.findValidAlphaFeeSwap(c.convertToPathInfo(param.BestRoute, param.PoolSimulatorBucket))
	// swap doesn't contains valid alpha fee swap
	if swapIndex.ExecutedId == -1 {
		return nil, ErrRouteNotHaveAlphaFeeDex
	}

	var alphaFee *dexlibPool.TokenAmount
	ammBestRouteAmountOut := bignumber.ZeroBI
	if param.BestAmmRoute != nil {
		ammBestRouteAmountOut = param.BestAmmRoute.AmountOut

		// If best route is not that much better than amm route, we don't need to apply alpha fee
		if c.NotMuchBetter(param.BestRoute, param.BestAmmRoute, true) {
			return nil, fmt.Errorf("amm route is almost equal with best route %w", ErrAlphaFeeNotExists)
		}
	}

	// To avoid amm best path returns weird route due to lack of swap source, we must check difference between
	// amm best path and multi best path do not exceed AlphaFeeSlippageTolerance config
	var tmp big.Int
	maxReducedAmountOut := tmp.Div(
		tmp.Mul(
			param.BestRoute.AmountOut,
			tmp.SetInt64(c.config.ReductionConfig.MaxThresholdPercentageInBps),
		),
		valueobject.BasisPoint,
	)
	// if amm best path returns weird route due to lack of swap source
	// we must cap amm best path amount out to a specific amount base on configuration rate
	if ammBestRouteAmountOut.Cmp(maxReducedAmountOut) < 0 {
		ammBestRouteAmountOut = maxReducedAmountOut
	}

	reductionDelta := new(big.Int).Sub(param.BestRoute.AmountOut, ammBestRouteAmountOut)
	if reductionDelta.Sign() <= 0 {
		return nil, fmt.Errorf("reductionDelta is negative reduction delta %v, best Amount %s, ammRoute %v, %w",
			reductionDelta, param.BestRoute.AmountOut, ammBestRouteAmountOut, ErrAlphaFeeNotExists)
	}

	currentPath := param.BestRoute.Paths[swapIndex.PathId]
	currentAmountIn := currentPath.AmountIn
	var pmmTokenAmount *dexlibPool.TokenAmount
	for i, poolId := range currentPath.PoolsOrder {
		fromToken := currentPath.TokensOrder[i]
		toToken := currentPath.TokensOrder[i+1]

		pool := param.PoolSimulatorBucket.GetPool(poolId)
		swapLimit := param.PoolSimulatorBucket.GetPoolSwapLimit(poolId)

		tokenAmountIn := dexlibPool.TokenAmount{Token: fromToken, Amount: currentAmountIn}
		res, err := c.CalcAmountOut(ctx, pool, tokenAmountIn, toToken, swapLimit)

		if err != nil {
			logger.Errorf(ctx, "Alpha fee calculation finalize|CalcAmountOut err: %v|%v %s->%s thru %s",
				err, currentAmountIn, fromToken, toToken, poolId)
			return nil, ErrInvalidSwap
		}

		if !res.IsValid() {
			return nil, ErrCalcAmountOutEmpty
		}

		currentAmountIn = res.TokenAmountOut.Amount
		if i == swapIndex.SwapId {
			pmmTokenAmount = res.TokenAmountOut
			alphaFee = c.calculateAlphaFee(param, reductionDelta, pmmTokenAmount, currentPath, pool.GetExchange())
			currentAmountIn = tmp.Sub(res.TokenAmountOut.Amount, alphaFee.Amount)
			if currentAmountIn.Sign() < 0 {
				// return error if amount out of pmm swap isn't enough to cover alpha fee
				// (this may not happen in reality, but we must have a check here to avoid weird error in calculation)
				logger.Errorf(ctx, "pmm swap amount %s are not enough to cover alpha fee %s",
					pmmTokenAmount.Amount, alphaFee.Amount)
				return nil, ErrAlphaSwapNotEnoughToCoverAlphaFee
			}
		}
	}
	if alphaFee == nil {
		return nil, ErrAlphaFeeNotExists
	}

	// recalculate total amount for the whole route
	totalAmount := tmp.Sub(currentPath.AmountOut, currentAmountIn)
	totalAmount = totalAmount.Sub(param.BestRoute.AmountOut, totalAmount)

	// final check alpha fee is valid if it still provides better amount than amm amount out
	if totalAmount.Cmp(ammBestRouteAmountOut) < 0 {
		logger.Errorf(ctx, "apply alpha fee %s provides less amount than amm amount %s",
			alphaFee.Amount, currentAmountIn)
		return nil, ErrApplyAlphaFeeYieldLessAmountThanAMM
	}

	return &routerEntity.AlphaFee{
		AlphaFeeToken: alphaFee.Token,
		Amount:        alphaFee.Amount,
		Pool:          currentPath.PoolsOrder[swapIndex.SwapId],
		AMMAmount:     ammBestRouteAmountOut,
		ExecutedId:    swapIndex.ExecutedId,
		TokenIn:       currentPath.TokensOrder[swapIndex.SwapId],
	}, nil
}

func (c *AlphaFeeCalculation) convertToPathInfo(
	route *finderCommon.ConstructRoute, simulatorBucket *finderCommon.SimulatorBucket) []PathInfo {
	result := make([]PathInfo, 0, len(route.Paths))
	for _, path := range route.Paths {
		swaps := make([]SwapInfo, 0, len(path.PoolsOrder))
		for _, pool := range path.PoolsOrder {
			poolSim := simulatorBucket.GetPool(pool)
			swaps = append(swaps, SwapInfo{
				pool:     pool,
				exchange: valueobject.Exchange(poolSim.GetExchange()),
			})
		}
		result = append(result, PathInfo{swaps})
	}

	return result
}

// This method returns executedId which is the order when sc execute the swap
func (c *AlphaFeeCalculation) findValidAlphaFeeSwap(paths []PathInfo) SwapIndex {
	minDistance := math.MaxInt
	minLen := math.MaxInt
	pathId := -1
	executedId := -1
	totalCount := 0

	for i, path := range paths {
		pathLen := len(path.SwapInfo)
		j := pathLen - 1 // last pmm pool
		for ; j >= 0; j-- {
			swap := path.SwapInfo[j]
			if privo.IsAlphaFeeSource(swap.exchange) {
				break
			}
		}
		// pmm swap not found
		if j == -1 {
			totalCount += pathLen
			continue
		}

		distance := pathLen - 1 - j
		if distance < minDistance || (distance == minDistance && pathLen < minLen) {
			minDistance = distance
			minLen = pathLen
			pathId = i
			executedId = totalCount + j
		}
		totalCount += pathLen
	}

	if executedId == -1 {
		return SwapIndex{
			PathId:     -1,
			SwapId:     -1,
			ExecutedId: -1,
		}
	}

	return SwapIndex{
		PathId:     pathId,
		SwapId:     len(paths[pathId].SwapInfo) - 1 - minDistance,
		ExecutedId: int32(executedId),
	}
}

func (c *AlphaFeeCalculation) calculateAlphaFee(param AlphaFeeParams, reductionDelta *big.Int,
	pmmTokenAmount *dexlibPool.TokenAmount, currentPath *finderCommon.ConstructPath,
	exchange string) *dexlibPool.TokenAmount {
	// deductionFactors are converted from float to integer by multiply it by 10, so we will div (BasisPoint * 10)
	alphaFee := new(big.Int)
	alphaFee.Div(
		alphaFee.Mul(reductionDelta, lo.CoalesceOrEmpty(c.ReductionFactorInBps[exchange], DefaultReductionFactorInBps)),
		types.BasisPointMulByTen,
	)

	// In case token out has price
	var alphaFeeTokenAmount *dexlibPool.TokenAmount
	if param.Prices[param.BestRoute.TokenOut] > 0 && param.Prices[pmmTokenAmount.Token] > 0 {
		alphaFeeTokenAmount = c.calculatePmmAlphaFeeExactly(
			pmmTokenAmount,
			&dexlibPool.TokenAmount{
				Token:  param.BestRoute.TokenOut,
				Amount: alphaFee,
			},
			param.Prices,
			param.Tokens,
		)
	} else {
		alphaFeeTokenAmount = c.calculateAlphaFeeApproximately(
			param.BestRoute,
			pmmTokenAmount,
			currentPath.AmountOut,
			alphaFee,
		)
	}

	return &dexlibPool.TokenAmount{
		Token:  alphaFeeTokenAmount.Token,
		Amount: alphaFeeTokenAmount.Amount,
	}
}

// this function will calculate alpha fee base on currency conversion rate through their prices
func (c *AlphaFeeCalculation) calculatePmmAlphaFeeExactly(
	pmmSwapTokenOut *dexlibPool.TokenAmount,
	alphaFee *dexlibPool.TokenAmount,
	prices map[string]float64, // usd prices
	tokens map[string]dexlibEntity.SimplifiedToken,
) *dexlibPool.TokenAmount {
	alphaFeeUsd := finderUtil.CalcAmountPrice(alphaFee.Amount, tokens[alphaFee.Token].Decimals, prices[alphaFee.Token])
	pmmSwapTokenOutAlphaFee := finderUtil.CalcAmountFromPrice(alphaFeeUsd, tokens[pmmSwapTokenOut.Token].Decimals,
		prices[pmmSwapTokenOut.Token])

	return &dexlibPool.TokenAmount{
		Token:     pmmSwapTokenOut.Token,
		Amount:    pmmSwapTokenOutAlphaFee,
		AmountUsd: alphaFeeUsd,
	}
}

func (c *AlphaFeeCalculation) calculateAlphaFeeApproximately(
	bestRoute *finderCommon.ConstructRoute,
	pmmSwapTokenOut *dexlibPool.TokenAmount,
	pmmPathAmountOut *big.Int,
	alphaFee *big.Int,
) *dexlibPool.TokenAmount {
	// Calculate split amount between the path contains pmmSwap need to be reduced and total amount
	routeAmountOutF, _ := bestRoute.AmountOut.Float64()
	pmmPathAmountOutF, _ := pmmPathAmountOut.Float64()
	splitPercentage := routeAmountOutF / pmmPathAmountOutF

	// Calculate the rate between alpha fee and total amount out
	alphaFeeAmountF, _ := alphaFee.Float64()
	amountOutF, _ := bestRoute.AmountOut.Float64()
	alphaFeeRate := alphaFeeAmountF / amountOutF

	// Calculate alpha fee in pmm swap using proportion formula
	pmmSwapAmountF, _ := pmmSwapTokenOut.Amount.Float64()
	pmmSwapTokenOutAlphaFee := alphaFeeRate * pmmSwapAmountF
	finalResult := pmmSwapTokenOutAlphaFee * splitPercentage

	// Convert float to int
	pmmAlphaFeeInt, _ := big.NewFloat(finalResult).Int(nil)

	return &dexlibPool.TokenAmount{
		Token:  pmmSwapTokenOut.Token,
		Amount: pmmAlphaFeeInt,
	}
}

func (c *AlphaFeeCalculation) NotMuchBetter(best, amm *finderCommon.ConstructRoute, gasIncluded bool) bool {
	if bestPrice, ammPrice := best.AmountOutPrice, amm.AmountOutPrice; bestPrice != 0 || ammPrice != 0 {
		if gasIncluded {
			bestPrice -= best.GasFeePrice + best.L1GasFeePrice
			ammPrice -= amm.GasFeePrice + amm.L1GasFeePrice
		}
		return bestPrice-ammPrice <= c.config.ReductionConfig.MinDifferentThresholdUSD
	}

	diff := new(big.Int).Sub(best.AmountOut, amm.AmountOut)
	diff.Mul(diff, valueobject.BasisPoint).Div(diff, amm.AmountOut)
	return diff.Int64() <= c.config.ReductionConfig.MinDifferentThresholdBps
}

func (c *AlphaFeeCalculation) CalculateDefaultAlphaFee(_ context.Context,
	param DefaultAlphaFeeParams) (*routerEntity.AlphaFee, error) {
	swapIndex := c.findValidAlphaFeeSwap(c.convertRouteSummaryToPathInfo(param.RouteSummary))
	// swap doesn't contains valid pmm swap
	if swapIndex.PathId == -1 || swapIndex.SwapId == -1 {
		return nil, ErrRouteNotHaveAlphaFeeDex
	}
	currentSwap := param.RouteSummary.Route[swapIndex.PathId][swapIndex.SwapId]

	amountF, _ := currentSwap.AmountOut.Float64()
	feeAmountF := amountF * c.config.ReductionConfig.DefaultAlphaFeePercentageBps / 1e4
	feeAmount, _ := big.NewFloat(feeAmountF).Int(nil)

	return &routerEntity.AlphaFee{
		Pool:          currentSwap.Pool,
		AlphaFeeToken: currentSwap.TokenOut,
		Amount:        feeAmount,
		ExecutedId:    swapIndex.ExecutedId,
	}, nil
}

func (c *AlphaFeeCalculation) convertRouteSummaryToPathInfo(route routerValueObject.RouteSummary) []PathInfo {
	result := make([]PathInfo, 0, len(route.Route))
	for _, path := range route.Route {
		swaps := make([]SwapInfo, 0, len(path))
		for _, swap := range path {
			swaps = append(swaps, SwapInfo{
				pool:     swap.Pool,
				exchange: swap.Exchange,
			})
		}
		result = append(result, PathInfo{swaps})
	}

	return result
}

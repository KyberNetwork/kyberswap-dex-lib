package alphafee

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/KyberNetwork/kutils/klog"
	dexlibEntity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kyber-pmm"
	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	routerValueObject "github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrInvalidSwap                         = errors.New("invalid swap")
	ErrCalcAmountOutEmpty                  = errors.New("calc amount out empty")
	ErrAlphaFeeNotExists                   = errors.New("alpha fee doesn't exit")
	ErrRouteNotHavePMM                     = errors.New("route doesn't have pmm swaps")
	ErrPMMSwapNotEnoughToCoverAlphaFee     = errors.New("pmm swap doesn't have enough amount out to cover alpha fee")
	ErrApplyAlphaFeeYeildLessAmountThanAMM = errors.New("applying alpha fee yeilds less amount out than amm route")
)

type SwapIndex struct {
	PathId int
	SwapId int
}

type AlphaFeeParams struct {
	BestRoute    *finderCommon.ConstructRoute
	BestAmmRoute *finderCommon.ConstructRoute

	Prices              map[string]float64
	Tokens              map[string]dexlibEntity.Token
	PoolSimulatorBucket *finderCommon.SimulatorBucket
}

type DefaultAlphaFeeParams struct {
	RouteSummary routerValueObject.RouteSummary
}

type SwapInfo struct {
	pool     string
	poolType string
}

type PathInfo struct {
	SwapInfo []SwapInfo
}

type AlphaFeeCalculation struct {
	// Config alpha Fee rate using percentage in BPS, the same as safety quoting, 1 bps = 0.01%
	// Convert deductionFactor from float to integer by multiply it by 10, then we will div (BasisPoint * 10)
	ReductionFactorInBps map[valueobject.Exchange]*big.Int
	config               routerValueObject.AlphaFeeConfig
	entity.ICustomFuncsHolder
}

func NewAlphaFeeCalculation(
	config routerValueObject.AlphaFeeConfig,
	customFuncs entity.ICustomFuncs) *AlphaFeeCalculation {
	factors := map[valueobject.Exchange]*big.Int{}
	for dex, number := range config.ReductionConfig.ReductionFactorInBps {
		factors[valueobject.Exchange(dex)] = big.NewInt(int64(number * 10))
	}
	return &AlphaFeeCalculation{
		ReductionFactorInBps: factors,
		ICustomFuncsHolder:   &entity.CustomFuncsHolder{ICustomFuncs: customFuncs},
		config:               config,
	}
}

func (c *AlphaFeeCalculation) Calculate(ctx context.Context, param AlphaFeeParams) (*routerEntity.AlphaFee, error) {
	if param.BestAmmRoute == nil {
		return nil, fmt.Errorf("amm route is nil %w", ErrAlphaFeeNotExists)
	}

	reductionDelta := new(big.Int).Sub(param.BestRoute.AmountOut, param.BestAmmRoute.AmountOut)
	if reductionDelta.Sign() <= 0 {
		return nil, fmt.Errorf("reductionDelta is negative reduction delta %v, best Amount %s, ammAmount %s, %w",
			reductionDelta, param.BestRoute.AmountOut, param.BestAmmRoute.AmountOut, ErrAlphaFeeNotExists)
	}

	// If AMM best path and pmm best path almost equal, return error
	if c.AlmostEqual(param.BestRoute, param.BestAmmRoute, true) {
		return nil, fmt.Errorf("amm route is almost equal with best route %w", ErrAlphaFeeNotExists)
	}

	var alphaFee *dexlibPool.TokenAmount
	ammBestRouteAmountOut := param.BestAmmRoute.AmountOut

	// To avoid amm best path returns weird route due to lack of swap source, we must check differency between
	// amm best path and multi best path do not exeed AlphaFeeSlippageTolerance config
	reducedAmountOutWithSlippageTolerance := new(big.Int).Div(
		new(big.Int).Mul(
			param.BestRoute.AmountOut,
			big.NewInt(c.config.ReductionConfig.MaxThresholdPercentageInBps),
		),
		valueobject.BasisPoint,
	)
	// if amm best path returns weird route due to lack of swap source
	// we must cap amm best path amount out to a specific amount base on configuration rate
	if param.BestAmmRoute.AmountOut.Cmp(reducedAmountOutWithSlippageTolerance) < 0 {
		ammBestRouteAmountOut = reducedAmountOutWithSlippageTolerance
		reductionDelta = new(big.Int).Sub(param.BestRoute.AmountOut, ammBestRouteAmountOut)
	}

	swapIndex := c.findValidPmmSwap(c.convertToPathInfo(param.BestRoute, param.PoolSimulatorBucket))
	// swap doesn't contains valid pmm swap
	if swapIndex.PathId == -1 || swapIndex.SwapId == -1 {
		return nil, ErrRouteNotHavePMM
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
			klog.Warnf(ctx, "Finalize|CalcAmountOut err: %v|%v %s->%s thru %s",
				err, currentAmountIn, fromToken, toToken, poolId)
			return nil, ErrInvalidSwap
		}

		if !res.IsValid() {
			return nil, ErrCalcAmountOutEmpty
		}

		currentAmountIn = res.TokenAmountOut.Amount
		if i == swapIndex.SwapId {
			pmmTokenAmount = res.TokenAmountOut
			alphaFee = c.calculateAlphaFee(param, reductionDelta, pmmTokenAmount, currentPath)
			currentAmountIn = new(big.Int).Sub(res.TokenAmountOut.Amount, alphaFee.Amount)
			if currentAmountIn.Sign() < 0 {
				// return error if amount out of pmm swap isn't enough to cover alpha fee
				// (this may not happen in reality but we must have a check here to avoid weird error in calculation)
				logger.Errorf(ctx, "pmm swap amount %s are not enough to cover alpha fee %s", pmmTokenAmount.Amount.Text(10), alphaFee.Amount.Text(10))
				return nil, ErrPMMSwapNotEnoughToCoverAlphaFee
			}
		}
	}

	// recalculate total amount for the whole route
	totalAmount := new(big.Int).Sub(currentPath.AmountOut, currentAmountIn)
	totalAmount = totalAmount.Sub(param.BestRoute.AmountOut, totalAmount)

	// final check alpha fee is valid if it still provide better amount than amm amount out
	if totalAmount.Cmp(ammBestRouteAmountOut) < 0 {
		logger.Errorf(ctx, "apply alpha fee %s provides less amount than amm amount %s", alphaFee.Amount.Text(10), currentAmountIn.Text(10))
		return nil, ErrApplyAlphaFeeYeildLessAmountThanAMM
	}

	return &routerEntity.AlphaFee{
		Token:     alphaFee.Token,
		Amount:    alphaFee.Amount,
		Pool:      currentPath.PoolsOrder[swapIndex.SwapId],
		AMMAmount: ammBestRouteAmountOut,
		PathId:    swapIndex.PathId,
		SwapId:    swapIndex.SwapId,
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
				poolType: poolSim.GetType(),
			})
		}
		result = append(result, PathInfo{swaps})
	}

	return result
}

func (c *AlphaFeeCalculation) findValidPmmSwap(paths []PathInfo) SwapIndex {
	minDistance := math.MaxInt
	minLen := math.MaxInt
	pathId := -1

	for i := 0; i < len(paths); i++ {
		pathLen := len(paths[i].SwapInfo)
		j := pathLen - 1 // last pmm pool
		for ; j >= 0; j-- {
			swap := paths[i].SwapInfo[j]
			if kyberpmm.DexTypeKyberPMM == valueobject.Exchange(swap.poolType) {
				break
			}
		}
		// pmm swap not found
		if j == -1 {
			continue
		}
		distance := pathLen - 1 - j
		if distance < minDistance || (distance == minDistance && pathLen < minLen) {
			minDistance = distance
			minLen = pathLen
			pathId = i
		}
	}
	if pathId == -1 {
		return SwapIndex{
			PathId: -1,
			SwapId: -1,
		}
	}

	return SwapIndex{
		PathId: pathId,
		SwapId: len(paths[pathId].SwapInfo) - 1 - minDistance,
	}
}

func (c *AlphaFeeCalculation) calculateAlphaFee(
	param AlphaFeeParams,
	reductionDelta *big.Int,
	pmmTokenAmount *dexlibPool.TokenAmount,
	currentPath *finderCommon.ConstructPath) *dexlibPool.TokenAmount {
	// deductionFactors are converted from float to integer by multiply it by 10, so we will div (BasisPoint * 10)
	alphaFee := new(big.Int).Div(
		new(big.Int).Mul(reductionDelta, c.ReductionFactorInBps[valueobject.ExchangeKyberPMM]),
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
	prices map[string]float64, //usd prices
	tokens map[string]dexlibEntity.Token,
) *dexlibPool.TokenAmount {
	alphaFeeUsd := finderUtil.CalcAmountPrice(alphaFee.Amount, tokens[alphaFee.Token].Decimals, prices[alphaFee.Token])
	pmmSwapTokenOutAlphaFee := finderUtil.CalcAmountFromPrice(alphaFeeUsd, tokens[pmmSwapTokenOut.Token].Decimals, prices[pmmSwapTokenOut.Token])

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
	routeAmountOutFloat := new(big.Float).SetInt(bestRoute.AmountOut)
	pmmPathAmountOutFloat := new(big.Float).SetInt(pmmPathAmountOut)
	splitPercentage := new(big.Float).Quo(routeAmountOutFloat, pmmPathAmountOutFloat)

	// Calculate the rate between alpha fee and total amount out
	alphaFeeAmountFloat := new(big.Float).SetInt(alphaFee)
	amountOutFloat := new(big.Float).SetInt(bestRoute.AmountOut)
	alphaFeeRate := new(big.Float).Quo(alphaFeeAmountFloat, amountOutFloat)

	// Calculate alpha fee in pmm swap using propotion formula
	pmmSwapAmountFloat := new(big.Float).SetInt(pmmSwapTokenOut.Amount)
	pmmSwapTokenOutAlphaFee := new(big.Float).Mul(alphaFeeRate, pmmSwapAmountFloat)
	finalResult := new(big.Float).Mul(pmmSwapTokenOutAlphaFee, splitPercentage)

	// Convert float to int
	pmmAlphaFeeInt := new(big.Int)
	pmmAlphaFeeInt, _ = finalResult.Int(pmmAlphaFeeInt)

	return &dexlibPool.TokenAmount{
		Token:  pmmSwapTokenOut.Token,
		Amount: pmmAlphaFeeInt,
	}

}

func (c *AlphaFeeCalculation) AlmostEqual(
	r *finderCommon.ConstructRoute, y *finderCommon.ConstructRoute, gasIncluded bool) bool {
	priceAvailable := r.AmountOutPrice != 0 || y.AmountOutPrice != 0

	if gasIncluded && priceAvailable {
		xValue := r.AmountOutPrice - r.L1GasFeePrice
		yValue := y.AmountOutPrice - y.L1GasFeePrice

		return math.Abs(xValue-yValue) <= c.config.ReductionConfig.MinDifferentThresholdUSD
	}

	diff := r.AmountOut.Sub(r.AmountOut, y.AmountOut)
	diff.Abs(diff)

	return diff.Cmp(big.NewInt(c.config.ReductionConfig.MinDifferentThresholdBps)) < 0
}

func (uc *AlphaFeeCalculation) CalculateDefaultAlphaFee(ctx context.Context, param DefaultAlphaFeeParams) (*routerEntity.AlphaFee, error) {
	swapIndex := uc.findValidPmmSwap(uc.convertRouteSummaryToPathInfo(param.RouteSummary))

	// swap doesn't contains valid pmm swap
	if swapIndex.PathId == -1 || swapIndex.SwapId == -1 {
		return nil, ErrRouteNotHavePMM
	}
	currentSwap := param.RouteSummary.Route[swapIndex.PathId][swapIndex.SwapId]

	percentageBps := big.NewFloat(uc.config.ReductionConfig.DefaultAlphaFeePercentageBps)
	basisPointF := new(big.Float).SetInt(valueobject.BasisPoint)
	amountF := new(big.Float).SetInt(currentSwap.AmountOut)

	feeAmountF := new(big.Float).Mul(amountF, percentageBps)
	feeAmountF.Quo(feeAmountF, basisPointF)

	feeAmount := new(big.Int)
	feeAmountF.Int(feeAmount)

	return &routerEntity.AlphaFee{
		Pool:   currentSwap.Pool,
		Token:  currentSwap.TokenOut,
		Amount: feeAmount,
		PathId: swapIndex.PathId,
		SwapId: swapIndex.SwapId,
	}, nil
}

func (c *AlphaFeeCalculation) convertRouteSummaryToPathInfo(route routerValueObject.RouteSummary) []PathInfo {
	result := make([]PathInfo, 0, len(route.Route))
	for _, path := range route.Route {
		swaps := make([]SwapInfo, 0, len(path))
		for _, swap := range path {
			swaps = append(swaps, SwapInfo{
				pool:     swap.Pool,
				poolType: swap.PoolType,
			})
		}
		result = append(result, PathInfo{swaps})
	}

	return result
}

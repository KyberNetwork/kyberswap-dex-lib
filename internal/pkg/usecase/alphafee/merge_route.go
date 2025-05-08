package alphafee

import (
	"context"
	"math"
	"math/big"

	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/pkg/errors"
)

func (c *AlphaFeeV2Calculation) calculateDefaultAlphaFeeNonMergeRoute(ctx context.Context, param DefaultAlphaFeeParams, alphaFee *big.Int) (*routerEntity.AlphaFeeV2, error) {
	routeInfo := convertRouteSummaryToRouteInfoV2(ctx, param.RouteSummary)

	pathReductions := c.getReductionPerPath(ctx, routeInfo, alphaFee)

	swapReductions, err := c.getDefaultReductionPerSwap(ctx, param, pathReductions, routeInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to apply reduction on paths")
	}

	return &routerEntity.AlphaFeeV2{
		AMMAmount:      param.RouteSummary.AmountOut,
		SwapReductions: swapReductions,
	}, nil
}

// getDefaultReductionPerSwap calculates and returns the reduction for each swap in a path.
// Unlike the original getReductionPerSwap function, this version does not use the pool simulator
// to compute the amount out. Instead, it relies on the ratio of amount out to amount in
// to quickly estimate the next amount out.
// Some of the logic is similar to the getReductionPerSwap function.
// This function is used during the BuildRoute step, where we do not yet refresh the pool state.
// It is also only used to calculate the default alpha fee—specifically for routes where alpha fee data
// is unavailable in our data store.
func (c *AlphaFeeV2Calculation) getDefaultReductionPerSwap(
	_ context.Context,
	param DefaultAlphaFeeParams,
	pathReductions []pathReduction,
	routeInfo [][]swapInfoV2,
) (swapReductions []routerEntity.AlphaFeeV2SwapReduction, err error) {
	swapReductions = make([]routerEntity.AlphaFeeV2SwapReduction, 0)

	pointer := 0 // to track the pathReductions
	executedId := 0
	for idx, path := range param.RouteSummary.Route {
		pathInfo := routeInfo[idx]
		pathContainsAlphaFeeSources := isPathContainsAlphaFeeSources(pathInfo)
		var reductionPercent float64
		if pathContainsAlphaFeeSources {
			// Calculate the percentage of amount out reduction of
			// each alpha fee source in the path
			numOfAlphaFeeSources := countAlphaFeeSourcesInPath(pathInfo)
			pathReduction := pathReductions[pointer].ReduceAmount
			pointer++

			pathReductionF, _ := pathReduction.Float64()
			pathAmountOutF, _ := path[len(path)-1].AmountOut.Float64()
			alphaFeePercent := pathReductionF / pathAmountOutF

			reductionPercent = math.Pow(1-alphaFeePercent, 1/float64(numOfAlphaFeeSources))
		}

		var currentAmountIn big.Int
		currentAmountIn.Set(path[0].SwapAmount)
		for _, swap := range path {
			tokenIn := swap.TokenIn
			tokenOut := swap.TokenOut

			amountOut := calculateAmountByRatio(swap.SwapAmount, swap.AmountOut, &currentAmountIn)

			var currentAmountOut big.Int
			currentAmountOut.Set(amountOut)

			if privo.IsAlphaFeeSource(swap.Exchange) {
				currentAmountOutF, _ := currentAmountOut.Float64()
				currentAmountOutF = currentAmountOutF * reductionPercent

				new(big.Float).SetFloat64(currentAmountOutF).Int(&currentAmountOut)

				reducedAmount := new(big.Int).Sub(amountOut, &currentAmountOut)
				swapReductions = append(swapReductions, routerEntity.AlphaFeeV2SwapReduction{
					ExecutedId:      executedId,
					PoolAddress:     swap.Pool,
					TokenIn:         tokenIn,
					TokenOut:        tokenOut,
					ReduceAmount:    reducedAmount,
					ReduceAmountUsd: 0, // Will be re-calculated using fair price when building route.
				})
			}

			currentAmountIn.Set(&currentAmountOut)
			executedId++
		}
	}

	return swapReductions, nil
}

func (c *AlphaFeeV2Calculation) calculateDefaultAlphaFeeMergeRoute(_ context.Context, param DefaultAlphaFeeParams, alphaFee *big.Int) (*routerEntity.AlphaFeeV2, error) {
	// Find the highest reduceBps, as long as
	// amountOutAfterReduction >= amountOut - alphaFee.
	var resultSwapReductions []routerEntity.AlphaFeeV2SwapReduction
	lo := 0
	hi := 10000

	for {
		if lo > hi {
			break
		}

		mid := (lo + hi) / 2
		isValid, swapReductions := isValidReduction(param.RouteSummary, mid, alphaFee)
		if isValid {
			resultSwapReductions = swapReductions
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	return &routerEntity.AlphaFeeV2{
		AMMAmount:      param.RouteSummary.AmountOut,
		SwapReductions: resultSwapReductions,
	}, nil
}

func isValidReduction(route valueobject.RouteSummary, reduceBps int, alphaFee *big.Int) (bool, []routerEntity.AlphaFeeV2SwapReduction) {
	reduceBpsF := float64(reduceBps) / 10000.0

	amountOutBeforeReduction := make(map[string]*big.Int)
	for _, path := range route.Route {
		for _, swap := range path {
			tokenOut := swap.TokenOut
			if _, ok := amountOutBeforeReduction[tokenOut]; !ok {
				amountOutBeforeReduction[tokenOut] = new(big.Int)
			}
			amountOutBeforeReduction[tokenOut].Add(amountOutBeforeReduction[tokenOut], swap.AmountOut)
		}
	}

	swapReductions := make([]routerEntity.AlphaFeeV2SwapReduction, 0)
	amountOutAfterReduction := make(map[string]*big.Int)
	var totalAmountOut big.Int
	routeTokenOut := route.Route[len(route.Route)-1][0].TokenOut
	executedId := 0

	for _, path := range route.Route {
		for _, swap := range path {
			// Scale down amountIn, since it can be reduced by previous swaps.
			tokenIn := swap.TokenIn
			amountIn := swap.SwapAmount
			if _, ok := amountOutAfterReduction[tokenIn]; ok {
				amountIn = calculateAmountByRatio(
					amountOutBeforeReduction[tokenIn],
					amountOutAfterReduction[tokenIn],
					amountIn,
				)
			}

			// We use calculateAmountOutByRatio to quickly calculate the amountOut,
			// instead of using the pool simulator to calculate the amountOut.
			amountOut := calculateAmountByRatio(swap.SwapAmount, swap.AmountOut, amountIn)

			// Charge alpha fee.
			if privo.IsAlphaFeeSource(swap.Exchange) {
				amountOutF, _ := amountOut.Float64()
				amountOutF = amountOutF * (1 - reduceBpsF)
				amountOutAfterReduce, _ := new(big.Float).SetFloat64(amountOutF).Int(nil)
				reduceAmount := new(big.Int).Sub(amountOut, amountOutAfterReduce)

				amountOut = amountOutAfterReduce

				swapReductions = append(swapReductions, routerEntity.AlphaFeeV2SwapReduction{
					ExecutedId:      executedId,
					PoolAddress:     swap.Pool,
					TokenIn:         swap.TokenIn,
					TokenOut:        swap.TokenOut,
					ReduceAmount:    reduceAmount,
					ReduceAmountUsd: 0, // Will be re-calculated using fair price when building route.
				})
			}

			if _, ok := amountOutAfterReduction[swap.TokenOut]; !ok {
				amountOutAfterReduction[swap.TokenOut] = new(big.Int)
			}
			amountOutAfterReduction[swap.TokenOut].Add(amountOutAfterReduction[swap.TokenOut], amountOut)

			if swap.TokenOut == routeTokenOut {
				totalAmountOut.Add(&totalAmountOut, amountOut)
			}

			executedId++
		}
	}

	if new(big.Int).Sub(route.AmountOut, alphaFee).Cmp(&totalAmountOut) <= 0 {
		return true, swapReductions
	}

	return false, swapReductions
}

func calculateAmountByRatio(numerator, denominator, amount *big.Int) *big.Int {
	if denominator.Sign() == 0 {
		return new(big.Int)
	}

	// If numerator == denominator, immediately return amount,
	// since we don't want to cast into float64 for incorrect precision.
	if numerator.Cmp(denominator) == 0 {
		return new(big.Int).Set(amount)
	}

	numeratorF, _ := numerator.Float64()
	denominatorF, _ := denominator.Float64()
	amountF, _ := amount.Float64()

	newAmountF := denominatorF * amountF / numeratorF

	newAmount, _ := new(big.Float).SetFloat64(newAmountF).Int(nil)
	return newAmount
}

func isMergeSwapRoute(route valueobject.RouteSummary) bool {
	if len(route.Route) == 0 {
		return false
	}

	for _, path := range route.Route {
		if len(path) > 1 {
			return false
		}
	}

	// Should take the tokenIn/tokenOut from the first and last path,
	// instead of route.TokenIn/TokenOut, because route.TokenIn/TokenOut
	// might be the native token.
	tokenIn := route.Route[0][0].TokenIn
	tokenOut := route.Route[len(route.Route)-1][0].TokenOut

	for _, path := range route.Route {
		swap := path[0]
		if swap.TokenIn != tokenIn || swap.TokenOut != tokenOut {
			return true
		}
	}

	return false
}

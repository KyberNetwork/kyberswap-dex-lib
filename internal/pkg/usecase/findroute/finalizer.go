package findroute

import (
	"context"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/safetyquote"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/pkg/errors"
)

type SafetyQuotingRouteFinalizer struct {
	safetyQuoteReduction *safetyquote.SafetyQuoteReduction
}

func NewSafetyQuotingRouteFinalizer(safetyQuoteReduction *safetyquote.SafetyQuoteReduction) *SafetyQuotingRouteFinalizer {
	return &SafetyQuotingRouteFinalizer{
		safetyQuoteReduction: safetyQuoteReduction,
	}
}

func (f *SafetyQuotingRouteFinalizer) FinalizeRoute(
	ctx context.Context,
	route *valueobject.Route,
	poolByAddress map[string]poolpkg.IPoolSimulator,
	swapLimits map[string]poolpkg.SwapLimit,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 2: finalize route
	finalizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {

		// Step 2.1: finalize path
		finalizedPath := make([]valueobject.Swap, 0, len(path.PoolAddresses))

		// Step 2.1.0: prepare input of the first swap
		tokenAmountIn := *path.Input.ToDexLibAmount()

		for swapIdx, swapPoolAddress := range path.PoolAddresses {
			// Step 2.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(swapPoolAddress)
			if !ok {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeRoute > pool not found [%s]",
					swapPoolAddress,
				)
			}

			swapLimit := swapLimits[pool.GetType()]
			// Step 2.1.2: simulate c swap through the pool
			result, err := routerpoolpkg.CalcAmountOut(ctx, pool, tokenAmountIn, path.Tokens[swapIdx+1].Address, swapLimit, map[string]bool{})
			if err != nil {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeRoute > swap failed > pool: [%s] > error : [%v]",
					swapPoolAddress,
					err,
				)
			}

			// Step 2.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeRoute > invalid swap > pool : [%s]",
					swapPoolAddress,
				)
			}

			//Step 2.1.4: clone the pool before updating it (do not modify IPool returned by `poolManager`)
			pool = poolBucket.ClonePool(swapPoolAddress)

			// Step 2.1.5: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool.UpdateBalance(updateBalanceParams)

			sqParams := types.SafetyQuotingParams{
				PoolType:             pool.GetType(),
				TokenIn:              tokenAmountIn.Token,
				TokenOut:             result.TokenAmountOut.Token,
				ApplyDeductionFactor: route.HasOnlyOneSwap(),
				ClientId:             params.ClientId,
			}

			// Step 2.1.6: We need to calculate safety quoting amount and reasign new amount out to next path's amount in
			reducedNextAmountIn := f.safetyQuoteReduction.Reduce(
				result.TokenAmountOut,
				f.safetyQuoteReduction.GetSafetyQuotingRate(sqParams))

			// Step 2.1.7: finalize the swap
			// important: must re-update amount out to reducedNextAmountIn
			swap := valueobject.Swap{
				Pool:              pool.GetAddress(),
				TokenIn:           tokenAmountIn.Token,
				TokenOut:          result.TokenAmountOut.Token,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         reducedNextAmountIn.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(tokenAmountIn.Token, result.TokenAmountOut.Token),
				Extra:             result.SwapInfo,
			}

			finalizedPath = append(finalizedPath, swap)

			// Step 2.1.7: add up gas fee
			gas += result.Gas

			// Step 2.1.8: update input of the next swap is output of current swap
			tokenAmountIn = reducedNextAmountIn

			metrics.IncrDexHitRate(ctx, string(swap.Exchange))
			metrics.IncrPoolTypeHitRate(ctx, swap.PoolType)
		}

		// Step 2.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		finalizedRoute = append(finalizedRoute, finalizedPath)
	}

	return &valueobject.RouteSummary{
		TokenIn:      params.TokenIn.Address,
		AmountIn:     params.AmountIn,
		AmountInUSD:  utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD),
		TokenOut:     params.TokenOut.Address,
		AmountOut:    amountOut,
		AmountOutUSD: utils.CalcTokenAmountUsd(amountOut, params.TokenOut.Decimals, params.TokenOutPriceUSD),
		Gas:          gas,
		GasPrice:     params.GasPrice,
		GasUSD:       utils.CalcGasUsd(params.GasPrice, gas, params.GasTokenPriceUSD),
		ExtraFee:     params.ExtraFee,
		Route:        finalizedRoute,
	}, nil
}

func (f *SafetyQuotingRouteFinalizer) FinalizeSimpleRoute(
	ctx context.Context,
	simpleRoute *valueobject.SimpleRoute,
	poolByAddress map[string]poolpkg.IPoolSimulator,
	swapLimits map[string]poolpkg.SwapLimit,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 2: distribute amountIn into paths following distributions
	distributedAmounts := business.DistributeAmount(params.AmountIn, simpleRoute.Distributions)

	// Step 3: finalize route
	finalizedRoute := make([][]valueobject.Swap, 0, len(simpleRoute.Paths))
	for pathIdx, simplePath := range simpleRoute.Paths {

		// Step 3.1: finalize path
		finalizedPath := make([]valueobject.Swap, 0, len(simplePath))

		// Step 3.1.0: prepare input of the first swap
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  simplePath[0].TokenInAddress,
			Amount: distributedAmounts[pathIdx],
		}

		for _, simpleSwap := range simplePath {
			// Step 3.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(simpleSwap.PoolAddress)
			if !ok {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeSimpleRoute > pool not found [%s]",
					simpleSwap.PoolAddress,
				)
			}

			swapLimit := swapLimits[pool.GetType()]
			// Step 3.1.2: simulate c swap through the pool
			result, err := routerpoolpkg.CalcAmountOut(ctx, pool, tokenAmountIn, simpleSwap.TokenOutAddress, swapLimit, map[string]bool{})
			if err != nil {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeSimpleRoute > swap failed > pool: [%s] > error : [%v]",
					simpleSwap.PoolAddress,
					err,
				)
			}

			// Step 3.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.WithMessagef(
					ErrInvalidSwap,
					"finalizer.FinalizeSimpleRoute > invalid swap > pool : [%s]",
					simpleSwap.PoolAddress,
				)
			}

			// Step 3.1.4: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool = poolBucket.ClonePool(simpleSwap.PoolAddress)
			pool.UpdateBalance(updateBalanceParams)

			sqParams := types.SafetyQuotingParams{
				PoolType:             pool.GetType(),
				TokenIn:              simpleSwap.TokenInAddress,
				TokenOut:             simpleSwap.TokenOutAddress,
				ApplyDeductionFactor: simpleRoute.HasOnlyOneSwap(),
				ClientId:             params.ClientId,
			}

			// Step 3.1.5
			// We need to calculate safety quoting amount and reasign new amount out to next path's amount in
			reducedNextAmountIn := f.safetyQuoteReduction.Reduce(
				result.TokenAmountOut,
				f.safetyQuoteReduction.GetSafetyQuotingRate(sqParams))

			// Step 3.1.6: finalize the swap
			// important: must re-update amount out to reducedNextAmountIn
			swap := valueobject.Swap{
				Pool:              simpleSwap.PoolAddress,
				TokenIn:           simpleSwap.TokenInAddress,
				TokenOut:          simpleSwap.TokenOutAddress,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         reducedNextAmountIn.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(simpleSwap.TokenInAddress, simpleSwap.TokenOutAddress),
				Extra:             result.SwapInfo,
			}

			finalizedPath = append(finalizedPath, swap)

			// Step 3.1.7: add up gas fee
			gas += result.Gas

			// Step 3.1.8: update input of the next swap is output of current swap
			tokenAmountIn = reducedNextAmountIn
		}

		// Step 3.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		finalizedRoute = append(finalizedRoute, finalizedPath)
	}

	return &valueobject.RouteSummary{
		TokenIn:      params.TokenIn.Address,
		AmountIn:     params.AmountIn,
		AmountInUSD:  utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD),
		TokenOut:     params.TokenOut.Address,
		AmountOut:    amountOut,
		AmountOutUSD: utils.CalcTokenAmountUsd(amountOut, params.TokenOut.Decimals, params.TokenOutPriceUSD),
		Gas:          gas,
		GasPrice:     params.GasPrice,
		GasUSD:       utils.CalcGasUsd(params.GasPrice, gas, params.GasTokenPriceUSD),
		ExtraFee:     params.ExtraFee,
		Route:        finalizedRoute,
	}, nil
}

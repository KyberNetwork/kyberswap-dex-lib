package findroute

import (
	"context"
	"fmt"
	"math/big"
	"runtime/debug"

	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderFinalizer "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finalizer"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/alphafee"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/mergeswap"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/safetyquote"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type FeeReductionRouteFinalizer struct {
	safetyQuoteReduction *safetyquote.SafetyQuoteReduction
	alphaFeeCalculation  *alphafee.AlphaFeeCalculation

	finderEntity.ICustomFuncsHolder
}

type FeeReductionFinalizerExtraData struct {
	BestAmmRoute *finderCommon.ConstructRoute
}

func NewFeeReductionRouteFinalizer(
	safetyQuoteReduction *safetyquote.SafetyQuoteReduction,
	alphaFeeCalculation *alphafee.AlphaFeeCalculation,
	customFuncs finderEntity.ICustomFuncs,
) *FeeReductionRouteFinalizer {
	return &FeeReductionRouteFinalizer{
		safetyQuoteReduction: safetyQuoteReduction,
		alphaFeeCalculation:  alphaFeeCalculation,
		ICustomFuncsHolder:   &finderEntity.CustomFuncsHolder{ICustomFuncs: customFuncs},
	}
}

func (f *FeeReductionRouteFinalizer) Finalize(
	ctx context.Context,
	params finderEntity.FinderParams,
	constructRoute *finderCommon.ConstructRoute,
	extraData any,
) (route *finderEntity.Route, err error) {
	routeId := requestid.GetRequestIDFromCtx(ctx)
	defer func() {
		if r := recover(); r != nil {
			route = nil
			err = errors.WithMessage(ErrPanicFinalizeRoute, fmt.Sprintf("err: %v", r))

			logger.WithFields(logger.Fields{
				"recover":     r,
				"route.Paths": constructRoute.Paths,
				"stackTrace":  string(debug.Stack()),
			}).Error("panic in Finalize route")
		}
	}()

	if constructRoute == nil || len(constructRoute.Paths) == 0 {
		return nil, finderFinalizer.ErrEmptyRoute
	}

	// Step 1: prepare pool data
	simulatorBucket := finderCommon.NewSimulatorBucket(params.Pools, params.SwapLimits, f.CustomFuncs())

	var (
		amountOut     = big.NewInt(0)
		gasUsed       = business.BaseGas
		l1GasFeePrice = params.L1GasFeePriceOverhead
	)

	// Step 1.1: Prepare alpha fee if needed
	feeReductionFinalizerExtraData, _ := extraData.(FeeReductionFinalizerExtraData)
	alphaFee, err := f.alphaFeeCalculation.Calculate(
		ctx, alphafee.AlphaFeeParams{
			BestRoute:           constructRoute,
			BestAmmRoute:        feeReductionFinalizerExtraData.BestAmmRoute,
			Prices:              params.Prices,
			Tokens:              params.Tokens,
			PoolSimulatorBucket: simulatorBucket,
		},
	)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err, "routeId": routeId}).Debug("error when calculate alpha fee")
	}

	// Step 2: finalize route
	finalizedRoute := make([][]finderEntity.Swap, 0, len(constructRoute.Paths))
	amountReductionEachSwap := make([][]*big.Int, 0, len(constructRoute.Paths))
	for pathId, path := range constructRoute.Paths {
		// Step 2.1: finalize path
		finalizedPath := make([]finderEntity.Swap, 0, len(path.PoolsOrder))
		amountReductionInPath := make([]*big.Int, 0, len(path.PoolsOrder))

		// Step 2.1.0: prepare input of the first swap
		currentAmountIn := path.AmountIn

		for i := 0; i < len(path.PoolsOrder); i++ {
			fromToken := path.TokensOrder[i]
			toToken := path.TokensOrder[i+1]

			// Step 2.1.1: take the pool with fresh data
			pool := simulatorBucket.GetPool(path.PoolsOrder[i])
			swapLimit := simulatorBucket.GetPoolSwapLimit(path.PoolsOrder[i])

			// Step 2.1.2: simulate swap through the pool
			tokenAmountIn := dexlibPool.TokenAmount{Token: fromToken, Amount: currentAmountIn}
			res, err := f.CalcAmountOut(ctx, pool, tokenAmountIn, toToken, swapLimit)

			if err != nil {
				return nil, errors.WithMessagef(
					finderFinalizer.ErrInvalidSwap,
					"[finalizer.safetyQuote] invalid swap. pool: [%s] err: [%v]",
					pool.GetAddress(), err,
				)
			}

			// Step 2.1.3: check if result is valid
			if res == nil ||
				res.TokenAmountOut == nil ||
				res.TokenAmountOut.Amount == nil ||
				res.TokenAmountOut.Amount.Sign() == 0 {
				return nil, errors.WithMessagef(
					finderFinalizer.ErrCalcAmountOutEmpty,
					"[finalizer.safetyQuote] calc amount out empty. pool: [%s]",
					pool.GetAddress(),
				)
			}

			// Step 2.1.4: clone the pool before updating it (do not modify IPool returned by `poolManager`)
			pool = simulatorBucket.ClonePoolById(ctx, path.PoolsOrder[i])
			swapLimit = simulatorBucket.CloneSwapLimitById(ctx, path.PoolsOrder[i])

			// Step 2.1.5: update balance of the pool
			updateBalanceParams := dexlibPool.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *res.TokenAmountOut,
				Fee:            *res.Fee,
				SwapInfo:       res.SwapInfo,
				SwapLimit:      swapLimit,
			}
			pool.UpdateBalance(updateBalanceParams)

			sqParams := types.SafetyQuotingParams{
				Exchange:             valueobject.Exchange(pool.GetExchange()),
				PoolType:             pool.GetType(),
				TokenIn:              tokenAmountIn.Token,
				TokenOut:             res.TokenAmountOut.Token,
				ApplyDeductionFactor: hasOnlyOneSwap(constructRoute),
				ClientId:             params.ClientId,
			}

			// Step 2.1.6: apply alpha fee reduction
			reducedNextAmountIn := res.TokenAmountOut.Amount
			if alphaFee != nil && alphaFee.PathId == pathId && alphaFee.SwapId == i {
				reducedNextAmountIn = new(big.Int).Sub(res.TokenAmountOut.Amount, alphaFee.Amount)
			}

			// Step 2.1.7: We need to calculate safety quoting amount and reasign new amount out to next path's amount in
			reducedNextAmountIn = f.safetyQuoteReduction.Reduce(
				&dexlibPool.TokenAmount{
					Token:  res.TokenAmountOut.Token,
					Amount: reducedNextAmountIn,
				},
				f.safetyQuoteReduction.GetSafetyQuotingRate(sqParams))

			// Step 2.1.8: finalize the swap
			// important: must re-update amount out to reducedNextAmountIn
			swap := finderEntity.Swap{
				Pool:       pool.GetAddress(),
				TokenIn:    tokenAmountIn.Token,
				TokenOut:   res.TokenAmountOut.Token,
				SwapAmount: tokenAmountIn.Amount,
				AmountOut:  reducedNextAmountIn,
				Exchange:   valueobject.Exchange(pool.GetExchange()),
				PoolType:   pool.GetType(),

				PoolExtra: pool.GetMetaInfo(tokenAmountIn.Token, res.TokenAmountOut.Token),
				Extra:     res.SwapInfo,
			}

			finalizedPath = append(finalizedPath, swap)
			amountReductionInPath = append(amountReductionInPath,
				new(big.Int).Sub(res.TokenAmountOut.Amount, reducedNextAmountIn))

			// Step 2.1.9: add up gas fee
			gasUsed += res.Gas

			// Step 2.1.10: update input of the next swap is output of current swap
			currentAmountIn = reducedNextAmountIn

			metrics.CountDexHit(ctx, string(swap.Exchange))
			metrics.CountPoolTypeHit(ctx, swap.PoolType)
		}

		l1GasFeePrice += params.L1GasFeePricePerPool * float64(len(path.PoolsOrder))

		// Step 2.2: add up amountOut
		if path.TokensOrder[len(path.TokensOrder)-1] == params.TokenOut {
			amountOut.Add(amountOut, currentAmountIn)
		}
		finalizedRoute = append(finalizedRoute, finalizedPath)
		amountReductionEachSwap = append(amountReductionEachSwap, amountReductionInPath)
	}

	gasFee := new(big.Int).Mul(big.NewInt(gasUsed), params.GasPrice)

	// Extra data used for bundled route and alpha fee calculation
	extra := types.FinalizeExtraData{}
	extra.UpdatedBalancePools, extra.UpdatedSwapLimits = simulatorBucket.GetUpdatedState()
	if alphaFee != nil && params.Prices[alphaFee.Token] > 0 {
		alphaFee.AmountUsd = finderUtil.CalcAmountPrice(alphaFee.Amount, params.Tokens[alphaFee.Token].Decimals,
			params.Prices[alphaFee.Token])
	}
	extra.AlphaFee = alphaFee

	if alphaFee != nil {
		logger.WithFields(logger.Fields{
			"routeId":           routeId,
			"alphaFeeToken":     alphaFee.Token,
			"alphaFeeAmount":    alphaFee.Amount.Text(10),
			"alphaFeeAmountUsd": alphaFee.AmountUsd,
		}).Info("route has alpha fee")
	}

	route = &finderEntity.Route{
		TokenIn:  params.TokenIn,
		AmountIn: params.AmountIn,
		AmountInPrice: finderUtil.CalcAmountPrice(params.AmountIn, params.Tokens[params.TokenIn].Decimals,
			params.Prices[params.TokenIn]),
		TokenOut:  params.TokenOut,
		AmountOut: amountOut,
		AmountOutPrice: finderUtil.CalcAmountPrice(amountOut, params.Tokens[params.TokenOut].Decimals,
			params.Prices[params.TokenOut]),
		GasUsed:  gasUsed,
		GasPrice: params.GasPrice,
		GasFee:   gasFee,
		GasFeePrice: finderUtil.CalcAmountPrice(gasFee, params.Tokens[params.GasToken].Decimals,
			params.Prices[params.GasToken]),
		L1GasFeePrice: l1GasFeePrice,
		Route:         finalizedRoute,

		ExtraFinalizerData: extra,
	}

	if params.SkipMergeSwap {
		return route, nil
	}

	mergeSwapRoute, err := mergeswap.MergeSwap(ctx, params, constructRoute, route, amountReductionEachSwap,
		f.CustomFuncs())
	if err != nil {
		return route, nil
	}

	return mergeSwapRoute, nil
}

func hasOnlyOneSwap(r *finderCommon.ConstructRoute) bool {
	if r.Paths == nil || len(r.Paths) != 1 {
		return false
	}

	if r.Paths[0] == nil || len(r.Paths[0].PoolsOrder) != 1 {
		return false
	}

	return true
}

func (f *FeeReductionRouteFinalizer) GetExtraData(_ context.Context,
	bestRouteResult *finderCommon.BestRouteResult) any {
	return FeeReductionFinalizerExtraData{
		BestAmmRoute: bestRouteResult.AMMBestRoute,
	}
}

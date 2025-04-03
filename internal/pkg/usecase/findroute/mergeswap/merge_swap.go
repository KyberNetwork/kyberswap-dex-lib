package mergeswap

import (
	"context"
	"fmt"
	"math/big"

	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	finderFinalizer "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finalizer"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/pkg/errors"
)

var ErrMergeSwapNotBetter = errors.New("merge swap route is not better than the original route")
var ErrCastFinalizeExtraData = errors.New("failed to cast extraFinalizerData to FinalizeExtraData")

// mergeSwap identifies duplicate swaps in the route and merges them.
// This function takes an `entityRoute`, which is being finalized by the previous step,
// to merge any duplicate swaps. Return a new merge swap route, without modifying the original route.
func MergeSwap(
	ctx context.Context,
	params finderEntity.FinderParams,
	constructRoute *finderCommon.ConstructRoute,
	entityRoute *finderEntity.Route,
	amountReductionEachSwap [][]*big.Int,
	customFuncs finderEntity.ICustomFuncs,
) (*finderEntity.Route, error) {
	if !canMergeSwap(ctx, constructRoute) {
		return entityRoute, nil
	}

	tokenTopoOrder, err := generateTokenTopoOrder(ctx, params, constructRoute)
	if err != nil {
		return nil, err
	}

	return mergeSwap(ctx, params, constructRoute, entityRoute, tokenTopoOrder, amountReductionEachSwap, customFuncs)
}

func mergeSwap(
	ctx context.Context,
	params finderEntity.FinderParams,
	constructRoute *finderCommon.ConstructRoute,
	entityRoute *finderEntity.Route,
	tokenTopoOrder []string,
	amountReductionEachSwap [][]*big.Int,
	customFuncs finderEntity.ICustomFuncs,
) (*finderEntity.Route, error) {
	r := finderCommon.NewConstructRoute(params.TokenIn, params.TokenOut, customFuncs)

	// Group the amount reduction of each swap by the key: `fromToken-toToken-poolAddress`.
	amountReductionEachMergeSwap := make(map[string]*big.Int)
	for i, amountReductionInPath := range amountReductionEachSwap {
		path := constructRoute.Paths[i]
		for j, amountReduction := range amountReductionInPath {
			poolAddress := path.PoolsOrder[j]
			key := fmt.Sprintf("%s-%s-%s", path.TokensOrder[j], path.TokensOrder[j+1], poolAddress)

			if _, exist := amountReductionEachMergeSwap[key]; !exist {
				amountReductionEachMergeSwap[key] = new(big.Int)
			}

			amountReductionEachMergeSwap[key].Add(amountReductionEachMergeSwap[key], amountReduction)
		}
	}

	simulatorBucket := finderCommon.NewSimulatorBucket(params.Pools, params.SwapLimits, customFuncs)
	singleSwapPaths := []*finderCommon.ConstructPath{}

	for _, path := range constructRoute.Paths {
		var currentAmountIn big.Int
		currentAmountIn.Set(path.AmountIn)

		for i, poolAddress := range path.PoolsOrder {
			fromToken := path.TokensOrder[i]
			toToken := path.TokensOrder[i+1]

			singleSwapPath := finderCommon.NewConstructPath(&currentAmountIn, customFuncs)
			singleSwapPath.AddToken(fromToken)
			singleSwapPath.AddToken(toToken)
			singleSwapPath.AddPoolByPoolSimulator(simulatorBucket.GetPool(poolAddress))

			if _, err := singleSwapPath.RefreshPathWithUpdateBalance(ctx, params, simulatorBucket); err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%s-%s-%s", fromToken, toToken, poolAddress)
			if amountReduction, exist := amountReductionEachMergeSwap[key]; exist && amountReduction.Sign() != 0 {
				singleSwapPath.SetAmountOutAndPrice(
					singleSwapPath.AmountOut.Sub(singleSwapPath.AmountOut, amountReduction),
					params.Tokens[toToken].Decimals,
					params.Prices[toToken],
				)
			}

			singleSwapPaths = append(singleSwapPaths, singleSwapPath)

			currentAmountIn.Set(singleSwapPath.AmountOut)
		}
	}

	// Add each single swap path to the adjusted route, using topo order to ensure the correct order of tokens.
	for _, token := range tokenTopoOrder {
		for _, singleSwapPath := range singleSwapPaths {
			if singleSwapPath.TokensOrder[0] != token {
				continue
			}

			r.AddPath(singleSwapPath)
		}
	}

	// Now, we **manually** refresh the adjusted route.
	// This is necessary because, after merging the amounts,
	// the `amountOut` from the merge step may differ from before merging.
	// We need to manually refresh to get the correct `amountOut`.
	// Since the adjusted route follows the topological order of tokens,
	// we can loop each path in the adjusted route and refresh it in order.
	simulatorBucket.ResetChangedData()
	simulatorBucket.ClearBackupPools()
	r.AmountOut.SetUint64(0)

	finalizedRoute := make([][]finderEntity.Swap, 0, len(constructRoute.Paths))
	gasUsed := business.BaseGas
	l1GasFeePrice := params.L1GasFeePriceOverhead

	mapTotalAmount := map[string]*big.Int{params.TokenIn: params.AmountIn}
	mapUsedAmount := map[string]*big.Int{}
	mapPathNum := map[string]int{}

	for _, path := range r.Paths {
		fromToken := path.TokensOrder[0]
		mapPathNum[fromToken]++
		if mapUsedAmount[fromToken] == nil {
			mapUsedAmount[fromToken] = new(big.Int)
		}
	}

	for _, path := range r.Paths {
		poolAddress := path.PoolsOrder[0]
		fromToken := path.TokensOrder[0]
		toToken := path.TokensOrder[1]

		if mapPathNum[fromToken] == 1 {
			path.AmountIn.Sub(mapTotalAmount[fromToken], mapUsedAmount[fromToken])
		}

		// Manual refresh path with update balance, to retrieve encode data.
		pool := simulatorBucket.GetPool(path.PoolsOrder[0])
		swapLimit := simulatorBucket.GetPoolSwapLimit(path.PoolsOrder[0])

		// Step 2.1.2: simulate swap through the pool
		tokenAmountIn := dexlibPool.TokenAmount{Token: fromToken, Amount: path.AmountIn}
		res, err := customFuncs.CalcAmountOut(ctx, pool, tokenAmountIn, toToken, swapLimit)

		if err != nil {
			return nil, errors.WithMessagef(
				finderFinalizer.ErrInvalidSwap,
				"[finalizer.safetyQuote] invalid swap. pool: [%s] err: [%v]",
				pool.GetAddress(), err,
			)
		}
		if !res.IsValid() {
			return nil, errors.WithMessagef(
				finderFinalizer.ErrCalcAmountOutEmpty,
				"[finalizer.safetyQuote] calc amount out empty. pool: [%s]",
				pool.GetAddress(),
			)
		}

		pool = simulatorBucket.ClonePoolById(ctx, path.PoolsOrder[0])
		swapLimit = simulatorBucket.CloneSwapLimitById(ctx, path.PoolsOrder[0])

		updateBalanceParams := dexlibPool.UpdateBalanceParams{
			TokenAmountIn:  tokenAmountIn,
			TokenAmountOut: *res.TokenAmountOut,
			Fee:            *res.Fee,
			SwapInfo:       res.SwapInfo,
			SwapLimit:      swapLimit,
		}
		pool.UpdateBalance(updateBalanceParams)

		key := fmt.Sprintf("%s-%s-%s", fromToken, toToken, poolAddress)

		if amountReduction, exist := amountReductionEachMergeSwap[key]; exist && amountReduction.Sign() != 0 {
			path.SetAmountOutAndPrice(
				path.AmountOut.Sub(res.TokenAmountOut.Amount, amountReduction),
				params.Tokens[toToken].Decimals,
				params.Prices[toToken],
			)
		} else {
			path.SetAmountOutAndPrice(
				res.TokenAmountOut.Amount,
				params.Tokens[toToken].Decimals,
				params.Prices[toToken],
			)
		}

		swap := finderEntity.Swap{
			Pool:       pool.GetAddress(),
			TokenIn:    tokenAmountIn.Token,
			TokenOut:   res.TokenAmountOut.Token,
			SwapAmount: tokenAmountIn.Amount,
			AmountOut:  path.AmountOut,
			Exchange:   valueobject.Exchange(pool.GetExchange()),
			PoolType:   pool.GetType(),

			LimitReturnAmount: constant.Zero,
			PoolLength:        len(pool.GetTokens()),
			PoolExtra:         pool.GetMetaInfo(tokenAmountIn.Token, res.TokenAmountOut.Token),
			Extra:             res.SwapInfo,
		}

		if toToken == params.TokenOut {
			r.AmountOut.Add(r.AmountOut, path.AmountOut)
			r.AmountOutPrice += path.AmountOutPrice
		}

		finalizedRoute = append(finalizedRoute, []finderEntity.Swap{swap})
		gasUsed += path.GasUsed
		l1GasFeePrice += params.L1GasFeePricePerPool * float64(len(path.PoolsOrder))

		// Update `mapTotalAmount`, `mapUsedAmount` and `mapPathNum`,
		// for path.amountIn update in the future loop.
		if mapUsedAmount[fromToken] == nil {
			mapUsedAmount[fromToken] = new(big.Int)
		}
		mapUsedAmount[fromToken].Add(mapUsedAmount[fromToken], path.AmountIn)

		if mapTotalAmount[toToken] == nil {
			mapTotalAmount[toToken] = new(big.Int)
		}
		mapTotalAmount[toToken].Add(mapTotalAmount[toToken], path.AmountOut)

		mapPathNum[fromToken]--
	}

	extra, ok := entityRoute.ExtraFinalizerData.(types.FinalizeExtraData)
	if !ok {
		return nil, ErrCastFinalizeExtraData
	}
	extra.RouteBeforeMergeSwap = entityRoute

	gasFee := new(big.Int).Mul(big.NewInt(gasUsed), params.GasPrice)
	route := &finderEntity.Route{
		TokenIn:       entityRoute.TokenIn,
		AmountIn:      entityRoute.AmountIn,
		AmountInPrice: entityRoute.AmountInPrice,
		TokenOut:      entityRoute.TokenOut,
		AmountOut:     r.AmountOut,
		AmountOutPrice: finderUtil.CalcAmountPrice(r.AmountOut, params.Tokens[params.TokenOut].Decimals,
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

	if compareRouteValue(params, route, entityRoute) <= 0 {
		return nil, ErrMergeSwapNotBetter
	}

	return route, nil
}

func canMergeSwap(
	_ context.Context,
	constructRoute *finderCommon.ConstructRoute,
) bool {
	// In SPFAv2, if we fail to refresh a multi-paths route after constructing it,
	// we will retry to construct the route, **without** merging the paths.
	// That optimization conflicts with this feature. So in order to make sure both logics work together,
	// if we can find a duplicated **path** in the route, we will not merge those swaps.
	if constructRoute.IsNoMergeMultiPathsRoute {
		return false
	}

	// Check if there are duplicate swaps in the route.
	mapDistinctSwaps := make(map[string]struct{})
	for _, path := range constructRoute.Paths {
		for idx, poolAddress := range path.PoolsOrder {
			key := fmt.Sprintf("%s-%s-%s", path.TokensOrder[idx], path.TokensOrder[idx+1], poolAddress)
			if _, exist := mapDistinctSwaps[key]; exist {
				return true
			}

			mapDistinctSwaps[key] = struct{}{}
		}
	}

	return false
}

func compareRouteValue(params finderEntity.FinderParams, x, y *finderEntity.Route) int {
	priceAvailable := x.AmountOutPrice != 0 || y.AmountOutPrice != 0

	if params.GasIncluded && priceAvailable {
		xValue := x.AmountOutPrice - x.GasFeePrice - x.L1GasFeePrice
		yValue := y.AmountOutPrice - y.GasFeePrice - y.L1GasFeePrice

		if finderUtil.AlmostEqual(xValue, yValue) {
			return x.AmountOut.Cmp(y.AmountOut)
		}

		if xValue < yValue {
			return -1
		} else {
			return 1
		}
	}

	return x.AmountOut.Cmp(y.AmountOut)
}

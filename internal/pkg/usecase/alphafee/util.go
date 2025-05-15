package alphafee

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kutils/klog"
	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const basisPointFloat = 10000.0

func convertConstructRouteToRouteInfoV2(ctx context.Context, route *common.ConstructRoute, simulatorBucket *common.SimulatorBucket) [][]swapInfoV2 {
	routeInfoV2 := make([][]swapInfoV2, len(route.Paths))
	for pathIdx, path := range route.Paths {
		pathInfoV2 := make([]swapInfoV2, len(path.PoolsOrder))

		for i, poolAddress := range path.PoolsOrder {
			pool := simulatorBucket.GetPool(poolAddress)
			if pool == nil {
				klog.Errorf(ctx, "pool %s not found in simulator bucket", poolAddress)
				continue
			}

			swapInfo := swapInfoV2{
				Pool:      poolAddress,
				TokenIn:   path.TokensOrder[i],
				TokenOut:  path.TokensOrder[i+1],
				AmountIn:  path.AmountIn,
				AmountOut: path.AmountOut,
				Exchange:  pool.GetExchange(),
			}

			pathInfoV2[i] = swapInfo
		}

		routeInfoV2[pathIdx] = pathInfoV2
	}

	return routeInfoV2
}

func convertRouteSummaryToRouteInfoV2(ctx context.Context, routeSummary valueobject.RouteSummary) [][]swapInfoV2 {
	routeInfoV2 := make([][]swapInfoV2, len(routeSummary.Route))
	for pathIdx, path := range routeSummary.Route {
		pathInfoV2 := make([]swapInfoV2, len(path))

		for i, swap := range path {
			swapInfo := swapInfoV2{
				Pool:      swap.Pool,
				TokenIn:   swap.TokenIn,
				TokenOut:  swap.TokenOut,
				AmountIn:  swap.SwapAmount,
				AmountOut: swap.AmountOut,
				Exchange:  string(swap.Exchange),
			}

			pathInfoV2[i] = swapInfo
		}

		routeInfoV2[pathIdx] = pathInfoV2
	}

	return routeInfoV2
}

func isPathContainsAlphaFeeSources(path []swapInfoV2) bool {
	for _, swap := range path {
		if privo.IsAlphaFeeSource(swap.Exchange) {
			return true
		}
	}

	return false
}

func countAlphaFeeSourcesInPath(path []swapInfoV2) int {
	alphaFeeSourceCount := 0
	for _, swap := range path {
		if privo.IsAlphaFeeSource(swap.Exchange) {
			alphaFeeSourceCount++
		}
	}

	return alphaFeeSourceCount
}

func LogAlphaFeeV2Info(alphaFee *routerEntity.AlphaFeeV2, routeId string, message string) {
	if alphaFee == nil {
		return
	}

	alphaFeeTokens := make([]string, 0, len(alphaFee.SwapReductions))
	alphaFeeAmounts := make([]*big.Int, 0, len(alphaFee.SwapReductions))
	alphaFeeAmountUsds := make([]string, 0, len(alphaFee.SwapReductions))

	for _, swapReduction := range alphaFee.SwapReductions {
		alphaFeeTokens = append(alphaFeeTokens, swapReduction.TokenOut)
		alphaFeeAmounts = append(alphaFeeAmounts, swapReduction.ReduceAmount)
		alphaFeeAmountUsds = append(alphaFeeAmountUsds, fmt.Sprintf("%.3f", swapReduction.ReduceAmountUsd))
	}

	if message == "" {
		message = "route has alpha fee"
	}
	logger.WithFields(logger.Fields{
		"routeId":            routeId,
		"alphaFeeTokens":     alphaFeeTokens,
		"alphaFeeAmounts":    alphaFeeAmounts,
		"alphaFeeAmountUsds": alphaFeeAmountUsds,
	}).Info(message)
}

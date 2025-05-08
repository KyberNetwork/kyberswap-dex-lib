package mergeswap

import (
	"context"
	"fmt"
	"slices"

	"github.com/KyberNetwork/kutils/klog"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/samber/lo"
)

func updateMergeSwapAlphaFee(
	ctx context.Context,
	mergeSwapRoute *finderEntity.Route,
	alphaFee *entity.AlphaFeeV2,
) {
	if alphaFee == nil {
		return
	}

	mergedSwapReductionsMap := map[string]entity.AlphaFeeV2SwapReduction{}

	for _, swapReduction := range alphaFee.SwapReductions {
		mergeKey := fmt.Sprintf("%s-%s-%s", swapReduction.PoolAddress, swapReduction.TokenIn, swapReduction.TokenOut)

		if _, ok := mergedSwapReductionsMap[mergeKey]; !ok {
			mergedSwapReductionsMap[mergeKey] = swapReduction
		} else {
			mergedSwap := mergedSwapReductionsMap[mergeKey]
			mergedSwap.ReduceAmount.Add(mergedSwap.ReduceAmount, swapReduction.ReduceAmount)
			mergedSwap.ReduceAmountUsd += swapReduction.ReduceAmountUsd

			mergedSwapReductionsMap[mergeKey] = mergedSwap
		}
	}

	// Update ExecutedId in merged swap reductions
	for i, swapReduction := range mergedSwapReductionsMap {
		executedId := 0
		found := false
		for _, path := range mergeSwapRoute.Route {
			for _, swap := range path {
				if swap.Pool == swapReduction.PoolAddress &&
					swap.TokenIn == swapReduction.TokenIn &&
					swap.TokenOut == swapReduction.TokenOut {
					swapReduction.ExecutedId = executedId
					mergedSwapReductionsMap[i] = swapReduction

					found = true
					break
				}
				executedId++
			}

			if found {
				break
			}
		}

		if !found {
			klog.WithFields(ctx, logger.Fields{
				"routeId":       requestid.GetRequestIDFromCtx(ctx),
				"swapReduction": swapReduction,
			}).Error("failed to find executed id for swap reduction")
		}
	}

	mergedSwapReductions := lo.Values(mergedSwapReductionsMap)
	slices.SortFunc(mergedSwapReductions, func(a, b entity.AlphaFeeV2SwapReduction) int {
		return a.ExecutedId - b.ExecutedId
	})
	alphaFee.SwapReductions = mergedSwapReductions
}

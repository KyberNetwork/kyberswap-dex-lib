package buildroute

import (
	"context"
	"runtime/debug"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func (uc *BuildRouteUseCase) convertAMMSwapsToPoolTrackers(route valueobject.RouteSummary, err error, command dto.BuildRouteCommand) []routerEntities.FaultyPoolTracker {
	failedCount := int64(0)

	// if SlippageTolerance >= 5%, we will consider a pool is faulty, otherwise, we do not encounter it
	// because in case that pool contains FOT token, slippage is high but that pool's state is not stale
	if isErrReturnAmountIsNotEnough(err) && slippageIsAboveMinThreshold(command.SlippageTolerance, uc.config.FaultyPoolsConfig) {
		failedCount = 1
	}

	trackers := []routerEntities.FaultyPoolTracker{}
	for _, path := range route.Route {
		for _, swap := range path {
			if uc.isPMMPoolsExceptLimitOrder(swap.PoolType) {
				continue
			}
			trackers = append(trackers, routerEntities.FaultyPoolTracker{
				Address:     swap.Pool,
				TotalCount:  1,
				FailedCount: failedCount,
			})
		}
	}

	return trackers
}

func (uc *BuildRouteUseCase) convertPMMSwapsToPoolTrackers(swaps []valueobject.Swap, err error) []routerEntities.FaultyPoolTracker {
	failedCount := int64(0)
	if isPMMFaultyPoolError(err) {
		failedCount = 1
	}

	trackers := []routerEntities.FaultyPoolTracker{}
	for _, swap := range swaps {
		if !uc.isPMMPoolsExceptLimitOrder(swap.PoolType) {
			continue
		}
		trackers = append(trackers, routerEntities.FaultyPoolTracker{
			Address:     swap.Pool,
			TotalCount:  1,
			FailedCount: failedCount,
		})
	}

	return trackers
}

func (uc *BuildRouteUseCase) isPMMPoolsExceptLimitOrder(poolType string) bool {
	if poolType == limitorder.DexTypeLimitOrder {
		return false
	}

	_, ok := valueobject.RFQSourceSet[valueobject.Exchange(poolType)]
	return ok
}

func (uc *BuildRouteUseCase) trackFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker, tokenIn, tokenOut string) {
	if !uc.config.FeatureFlags.IsFaultyPoolDetectorEnable {
		return
	}

	// requests to be tracked should only involve tokens that have been whitelisted or native token
	if !isTokenValid(tokenIn, uc.config.FaultyPoolsConfig, uc.config.ChainID) ||
		!isTokenValid(tokenOut, uc.config.FaultyPoolsConfig, uc.config.ChainID) {
		return
	}

	// pool-service will return InvalidArgument error if trackers list is empty
	if len(trackers) == 0 {
		return
	}
	results, err := uc.poolRepository.TrackFaultyPools(ctx, trackers)

	if err != nil {
		logger.WithFields(
			ctx,
			logger.Fields{
				"error":      err,
				"stacktrace": string(debug.Stack()),
				"trackPools": results,
				"requestId":  requestid.GetRequestIDFromCtx(ctx),
			}).Error("fail to add faulty pools")
	}

}

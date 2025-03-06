package buildroute

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/crypto"
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
				Tokens:      []string{swap.TokenIn, swap.TokenOut},
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
			Tokens:      []string{swap.TokenIn, swap.TokenOut},
		})
	}

	return trackers
}

func (uc *BuildRouteUseCase) isPMMPoolsExceptLimitOrder(poolType string) bool {
	if poolType == limitorder.DexTypeLimitOrder {
		return false
	}

	return dexValueObject.IsRFQSource(valueobject.Exchange(poolType))
}

func (uc *BuildRouteUseCase) shouldTrackTokens(ctx context.Context, tokens mapset.Set[string]) bool {
	unwhiteListTokens := make([]string, 0, 2)
	tokens.Each(func(s string) bool {
		if !isTokenWhiteList(s, uc.config.FaultyPoolsConfig, uc.config.ChainID) {
			unwhiteListTokens = append(unwhiteListTokens, s)
		}
		return false
	})

	if len(unwhiteListTokens) == 0 {
		return true
	}

	// fetch token info to check if the token is fot token or honeypot
	tokenInfo, err := uc.tokenRepository.FindTokenInfoByAddress(ctx, unwhiteListTokens)
	if err != nil {
		logger.Errorf(ctx, "shouldTrackTokens failed to find token info from token catalog: %v", err)
		return false
	}

	for _, info := range tokenInfo {
		logger.Debugf(ctx, "FindTokenInfoByAddress tokenInfo address %s, isFOT %t, isHoneyPot %t", info.Address, info.IsFOT, info.IsHoneypot)
		if isInvalid(info) {
			return false
		}
	}

	return true

}

func (uc *BuildRouteUseCase) trackFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker, isTrackFaultyPools bool) {
	if !isTrackFaultyPools {
		return
	}

	allTokens := mapset.NewThreadUnsafeSet[string]()
	for _, tracker := range trackers {
		allTokens.Append(tracker.Tokens...)
	}

	if !uc.shouldTrackTokens(ctx, allTokens) {
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

func (uc *BuildRouteUseCase) IsValidToTrackFaultyPools(routeTimestamp int64) bool {
	now := time.Now().Unix()
	secondElapsed := time.Duration(now-routeTimestamp) * time.Second
	return secondElapsed <= valueobject.DefaultDeadline
}

func (uc *BuildRouteUseCase) IsValidChecksum(route valueobject.RouteSummary, originalChecksum uint64) bool {
	checksum := crypto.NewChecksum(route, uc.config.Salt)
	return checksum.Verify(originalChecksum)
}

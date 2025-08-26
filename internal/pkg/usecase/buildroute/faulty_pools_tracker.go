package buildroute

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func (uc *BuildRouteUseCase) recordMetrics(ctx context.Context, route [][]valueobject.Swap, slippageTolerance float64,
	err error) {
	clientId := clientid.GetClientIDFromCtx(ctx)
	isSuccess := err == nil

	for _, path := range route {
		for _, swap := range path {
			metrics.CountEstimateGas(ctx, isSuccess, string(swap.Exchange), clientId)
		}
	}

	metrics.RecordEstimateGasWithSlippage(ctx, slippageTolerance, isSuccess)
}

func (uc *BuildRouteUseCase) handleFaultyPools(
	ctx context.Context,
	routeSummary *valueobject.RouteSummary,
	originalSlippage float64,
	estimatedSlippage float64,
	err error,
	isFaultyPoolTrackEnable bool,
) {
	uc.recordMetrics(ctx, routeSummary.Route, originalSlippage, err)

	// Handle faulty pools if needed
	if isSwapSinglePoolFailed(err) {
		uc.blockFaultyPool(ctx, routeSummary.Route, err)
	} else if isFaultyPoolTrackEnable && (err == nil || estimatedSlippage > 0) {
		uc.monitorFaultyPools(ctx, uc.createAMMPoolTrackers(ctx, routeSummary, err, estimatedSlippage))
	}
}

func (uc *BuildRouteUseCase) blockFaultyPool(ctx context.Context, route [][]valueobject.Swap, err error) {
	lg := log.Ctx(ctx).With().Str("clientId", clientid.GetClientIDFromCtx(ctx)).Logger()

	sequence, hop, ok := ExtractPoolIndexFromError(err)
	if !ok {
		lg.Err(err).Msg("Failed to extract swap error indices")
		return
	}

	if sequence < 0 || sequence >= len(route) {
		lg.Err(err).
			Int("sequence", sequence).
			Int("pathLen", len(route)).
			Msg("Invalid sequence index")
		return
	}

	if hop < 0 || hop >= len(route[sequence]) {
		lg.Err(err).
			Int("sequence", sequence).
			Int("hop", hop).
			Int("swapLen", len(route[sequence])).
			Msg("Invalid hop index")
		return
	}

	swap := route[sequence][hop]

	if err := uc.poolRepository.AddFaultyPools(ctx, []routerEntities.FaultyPool{
		{
			Address:   swap.Pool,
			ExpiresAt: time.Now().UTC().Add(uc.config.FaultyPoolsConfig.ExpireTime),
		},
	}); err != nil {
		log.Ctx(ctx).Err(err).
			Str("exchange", string(swap.Exchange)).
			Str("pool", swap.Pool).
			Msg("Failed to add faulty pool")
	}

	log.Ctx(ctx).Info().Err(err).
		Str("exchange", string(swap.Exchange)).
		Str("pool", swap.Pool).
		Msg("EstimateGas failed")
}

func (uc *BuildRouteUseCase) monitorFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker) {
	// pool-service will return InvalidArgument error if trackers list is empty
	if len(trackers) == 0 {
		return
	}

	allTokens := mapset.NewThreadUnsafeSet[string]()
	for _, tracker := range trackers {
		allTokens.Append(tracker.Tokens...)
	}

	if !uc.shouldTrackTokens(ctx, allTokens) {
		return
	}

	results, err := uc.poolRepository.TrackFaultyPools(ctx, trackers)
	if err != nil {
		pools := lo.Map(trackers, func(item routerEntities.FaultyPoolTracker, _ int) string {
			return item.Address
		})

		log.Ctx(ctx).Err(err).Bytes("stack", debug.Stack()).
			Strs("trackPools", pools).
			Msg("failed to add faulty pools")
	}

	log.Ctx(ctx).Debug().Strs("trackPools", results).
		Msg("added faulty pools")
}

func (uc *BuildRouteUseCase) createAMMPoolTrackers(
	ctx context.Context,
	route *valueobject.RouteSummary,
	err error,
	estimatedSlippage float64,
) []routerEntities.FaultyPoolTracker {
	var trackers []routerEntities.FaultyPoolTracker
	var failedCount int64
	// Get token group type
	tokenGroupType, _ := uc.config.TokenGroups.GetTokenGroupType(valueobject.TokenGroupParams{
		TokenIn:  route.TokenIn,
		TokenOut: route.TokenOut,
	})

	// if estimatedSlippage > MinSlippageThreshold, we will consider a pool is faulty, otherwise, we do not encounter it
	// because in case that pool contains FOT token, slippage is high but that pool's state is not stale
	if isSlippageAboveMinThreshold(estimatedSlippage, tokenGroupType,
		uc.config.FaultyPoolsConfig.SlippageConfigByGroup) {
		failedCount = 1
	}

	poolTags := make([]string, 0)

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

			poolTags = append(poolTags, string(swap.Exchange)+":"+swap.Pool)
		}
	}

	if err != nil {
		log.Ctx(ctx).Info().Err(err).
			Str("clientId", clientid.GetClientIDFromCtx(ctx)).
			Strs("poolTags", poolTags).
			Msg("EstimateGas failed")
	}

	return trackers
}

func (uc *BuildRouteUseCase) createPMMPoolTrackers(swaps []valueobject.Swap,
	err error) []routerEntities.FaultyPoolTracker {
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
		log.Ctx(ctx).Err(err).Msg("shouldTrackTokens failed to find token info from token catalog")
		return false
	}

	for _, info := range tokenInfo {
		log.Ctx(ctx).Debug().Msgf("FindTokenInfoByAddress tokenInfo address %s, isFOT %t, isHoneyPot %t",
			info.Address, info.IsFOT, info.IsHoneypot)
		if isInvalid(info) {
			return false
		}
	}

	return true

}

func (uc *BuildRouteUseCase) IsValidChecksum(route *valueobject.RouteSummary) bool {
	return time.Since(time.Unix(route.Timestamp, 0)) <= valueobject.DefaultDeadline &&
		route.Checksum(uc.config.Salt) == route.OriginalChecksum
}

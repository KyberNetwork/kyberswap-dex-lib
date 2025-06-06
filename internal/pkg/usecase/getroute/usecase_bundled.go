package getroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type bundledUseCase struct {
	*useCase
	aggregator IBundledAggregator
}

func NewBundledUseCase(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainpriceRepository IOnchainPriceRepository,
	routeCacheRepository IRouteCacheRepository,
	gasRepository IGasRepository,
	l1FeeEstimator IL1FeeEstimator,
	poolManager IPoolManager,
	poolFactory IPoolFactory,
	finderEngine finderEngine.IPathFinderEngine,
	config Config,
) *bundledUseCase {
	aggregator := NewBundledAggregator(
		poolRankRepository,
		tokenRepository,
		onchainpriceRepository,
		poolManager,
		poolFactory,
		config.Aggregator,
		finderEngine,
	)

	uc := &useCase{
		tokenRepository:        tokenRepository,
		gasRepository:          gasRepository,
		l1FeeEstimator:         l1FeeEstimator,
		config:                 config,
		onchainpriceRepository: onchainpriceRepository,
	}
	return &bundledUseCase{uc, aggregator}
}

func (u *bundledUseCase) ApplyConfig(config Config) {
	u.useCase.ApplyConfig(config)
	u.aggregator.ApplyConfig(config)
}

func (u *bundledUseCase) Handle(ctx context.Context, query dto.GetBundledRoutesQuery) (*dto.GetBundledRoutesResult,
	error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] bundledUseCase.Handle")
	defer span.End()

	originalTokensIn := make([]string, len(query.Pairs))
	originalTokensOut := make([]string, len(query.Pairs))
	for i, pair := range query.Pairs {
		originalTokensIn[i] = pair.TokenIn
		originalTokensOut[i] = pair.TokenOut

		wrappedTokenIn, err := eth.ConvertEtherToWETH(pair.TokenIn, u.config.ChainID)
		if err != nil {
			return nil, err
		}

		wrappedTokenOut, err := eth.ConvertEtherToWETH(pair.TokenOut, u.config.ChainID)
		if err != nil {
			return nil, err
		}

		pair.TokenIn = wrappedTokenIn
		pair.TokenOut = wrappedTokenOut
	}

	params, err := u.getAggregateBundledParams(ctx, query)
	if err != nil {
		return nil, err
	}

	routesSummary, err := u.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}

	for i, s := range routesSummary {
		s.TokenIn = originalTokensIn[i]
		s.TokenOut = originalTokensOut[i]
	}

	return &dto.GetBundledRoutesResult{
		RoutesSummary: routesSummary,
		RouterAddress: u.config.RouterAddress,
	}, nil
}

func (u *bundledUseCase) getAggregateBundledParams(ctx context.Context,
	query dto.GetBundledRoutesQuery) (*types.AggregateBundledParams, error) {
	pairs := lo.Map(query.Pairs, func(p *dto.GetBundledRoutesQueryPair, _ int) types.AggregateBundledParamsPair {
		return types.AggregateBundledParamsPair{
			TokenIn:  p.TokenIn,
			TokenOut: p.TokenOut,
			AmountIn: p.AmountIn,
		}
	})

	gasPrice, err := u.getGasPrice(ctx, query.GasPrice)
	if err != nil {
		return nil, err
	}

	var l1FeeOverhead, l1FeePerPool *big.Int
	if valueobject.IsL1FeeEstimateSupported(u.config.ChainID) {
		if l1FeeOverhead, l1FeePerPool, err = u.l1FeeEstimator.EstimateL1Fees(ctx); err != nil {
			klog.Errorf(ctx, "failed to estimate l1 fees: %v", err)
		}
	}

	sources := u.getSources(query.ClientId, query.BotScore, query.IncludedSources, query.ExcludedSources,
		query.OnlyScalableSources)

	var overridePools []*entity.Pool
	err = json.Unmarshal(query.OverridePools, &overridePools)
	if err != nil {
		return nil, err
	}

	index := valueobject.NativeTvl
	if u.config.Aggregator.FeatureFlags.IsLiquidityScoreIndexEnable {
		if query.Index != "" {
			index = valueobject.IndexType(query.Index)
		} else {
			index = valueobject.IndexType(u.config.DefaultPoolsIndex)
		}
	}

	var kyberLimitOrderAllowedSenders string
	if u.config.Aggregator.FeatureFlags.IsKyberPrivateLimitOrdersEnabled && query.ClientId == clientid.KyberSwap {
		kyberLimitOrderAllowedSenders = u.config.KyberExecutorAddress
	}

	return &types.AggregateBundledParams{
		Pairs:                         pairs,
		GasToken:                      u.config.GasTokenAddress,
		Sources:                       sources,
		GasInclude:                    query.GasInclude,
		GasPrice:                      gasPrice,
		L1FeeOverhead:                 l1FeeOverhead,
		L1FeePerPool:                  l1FeePerPool,
		IsHillClimbEnabled:            u.config.Aggregator.FeatureFlags.IsHillClimbEnabled,
		Index:                         index,
		ExcludedPools:                 query.ExcludedPools,
		ForcePoolsForToken:            u.config.ForcePoolsForTokenByClient[query.ClientId],
		ClientId:                      query.ClientId,
		OverridePools:                 overridePools,
		ExtraWhitelistedTokens:        query.ExtraWhitelistedTokens,
		KyberLimitOrderAllowedSenders: kyberLimitOrderAllowedSenders,
	}, nil
}

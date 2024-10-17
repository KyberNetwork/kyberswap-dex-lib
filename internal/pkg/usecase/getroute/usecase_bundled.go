package getroute

import (
	"context"

	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/samber/lo"
)

type bundledUseCase struct {
	*useCase
	aggregator IBundledAggregator
}

func NewBundledUseCase(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	routeCacheRepository IRouteCacheRepository,
	gasRepository IGasRepository,
	poolManager IPoolManager,
	finderEngine finderEngine.IPathFinderEngine,
	config Config,
) *bundledUseCase {
	aggregator := NewBundledAggregator(
		poolRankRepository,
		tokenRepository,
		priceRepository,
		onchainpriceRepository,
		poolManager,
		config.Aggregator,
		finderEngine,
	)

	uc := &useCase{
		tokenRepository: tokenRepository,
		priceRepository: priceRepository,
		gasRepository:   gasRepository,
		config:          config,

		onchainpriceRepository: onchainpriceRepository,
	}
	return &bundledUseCase{uc, aggregator}
}

func (u *bundledUseCase) Handle(ctx context.Context, query dto.GetBundledRoutesQuery) (*dto.GetBundledRoutesResult, error) {
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

func (u *bundledUseCase) getAggregateBundledParams(ctx context.Context, query dto.GetBundledRoutesQuery) (*types.AggregateBundledParams, error) {
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

	sources := u.getSources(query.IncludedSources, query.ExcludedSources)

	isHillClimbEnabled := u.config.Aggregator.FeatureFlags.IsHillClimbEnabled

	return &types.AggregateBundledParams{
		Pairs:              pairs,
		GasToken:           u.config.GasTokenAddress,
		Sources:            sources,
		SaveGas:            query.SaveGas,
		GasInclude:         query.GasInclude,
		GasPrice:           gasPrice,
		IsHillClimbEnabled: isHillClimbEnabled,
		ExcludedPools:      query.ExcludedPools,
		ClientId:           query.ClientId,
	}, nil
}

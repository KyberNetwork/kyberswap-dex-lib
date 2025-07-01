package getroute

import (
	"context"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type bundledUseCase struct {
	*useCase
	aggregator IBundledAggregator
}

func NewBundledUseCase(
	config Config,
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	onchainPriceRepository IOnchainPriceRepository,
	gasRepository IGasRepository,
	alphaFeeRepository, alphaFeeMigrationRepository IAlphaFeeRepository,
	l1FeeEstimator IL1FeeEstimator,
	poolManager IPoolManager,
	poolFactory IPoolFactory,
	finderEngine finderEngine.IPathFinderEngine,
) *bundledUseCase {
	return &bundledUseCase{useCase: &useCase{
		config:                      config,
		tokenRepository:             tokenRepository,
		gasRepository:               gasRepository,
		alphaFeeRepository:          alphaFeeRepository,
		alphaFeeMigrationRepository: alphaFeeMigrationRepository,
		l1FeeEstimator:              l1FeeEstimator,
		onchainPriceRepository:      onchainPriceRepository,
	}, aggregator: NewBundledAggregator(
		config.Aggregator,
		poolRankRepository,
		tokenRepository,
		onchainPriceRepository,
		poolManager,
		poolFactory,
		finderEngine,
	)}
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

	routeSummaries, err := u.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}

	routeID := requestid.GetRequestIDFromCtx(ctx)

	for i, routeSummary := range routeSummaries {
		routeID := routeID + "-" + strconv.Itoa(i)
		// Only save routes including alphaFee
		if routeSummary.AlphaFee != nil {
			if err = u.alphaFeeRepository.Save(ctx, routeID, routeSummary.AlphaFee); err != nil {
				return nil, err
			}

			if u.alphaFeeMigrationRepository != nil {
				if err = u.alphaFeeMigrationRepository.Save(ctx, routeID, routeSummary.AlphaFee); err != nil {
					klog.Errorf(ctx, "[Migration] failed to save alphaFee to new redis repository: %v", err)
				}
			}
		}

		u.checksumRouteSummary(routeSummary, originalTokensIn[i], originalTokensOut[i], routeID)
	}

	return &dto.GetBundledRoutesResult{
		RoutesSummary: routeSummaries,
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

	var tmpOverridePools, overridePools []*entity.Pool
	err = json.Unmarshal(query.OverridePools, &tmpOverridePools)
	if err != nil {
		return nil, err
	}

	for _, pool := range tmpOverridePools {
		if !query.ExcludedPools.Contains(pool.Address) {
			overridePools = append(overridePools, pool)
		}
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
		GasToken:                      u.config.GasTokenAddress,
		Sources:                       sources,
		OnlySinglePath:                query.OnlySinglePath,
		GasInclude:                    query.GasInclude,
		GasPrice:                      gasPrice,
		L1FeeOverhead:                 l1FeeOverhead,
		L1FeePerPool:                  l1FeePerPool,
		IsHillClimbEnabled:            u.config.Aggregator.FeatureFlags.IsHillClimbEnabled,
		Index:                         index,
		ExcludedPools:                 query.ExcludedPools,
		ForcePoolsForToken:            u.config.ForcePoolsForTokenByClient[query.ClientId],
		Pairs:                         pairs,
		OverridePools:                 overridePools,
		ExtraWhitelistedTokens:        query.ExtraWhitelistedTokens,
		ClientId:                      query.ClientId,
		IsScaleHelperClient:           lo.Contains(u.config.ScaleHelperClients, query.ClientId),
		KyberLimitOrderAllowedSenders: kyberLimitOrderAllowedSenders,
		EnableAlphaFee:                u.config.Aggregator.FeatureFlags.IsAlphaFeeReductionEnable,
		EnableHillClimbForAlphaFee:    u.config.Aggregator.FeatureFlags.IsHillClimbEnabledForAMMBestRoute,
	}, nil
}

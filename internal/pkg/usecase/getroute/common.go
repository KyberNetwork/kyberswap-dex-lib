package getroute

import (
	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finder/hillclimb"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finder/mergeswap"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finder/retry"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finder/spfav2"
	finderUtil "github.com/KyberNetwork/pathfinder-lib/pkg/util"
	"github.com/samber/lo"

	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/aevm"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/alphafee"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/safetyquote"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func simplifyRouteSummary(routeSummary *valueobject.RouteSummary) *valueobject.SimpleRoute {
	distributions := make([]uint64, 0, len(routeSummary.Route))
	simplePaths := make([][]valueobject.SimpleSwap, 0, len(routeSummary.Route))

	for _, path := range routeSummary.Route {
		simplePath := make([]valueobject.SimpleSwap, 0, len(path))
		for _, swap := range path {
			simpleSwap := valueobject.SimpleSwap{
				PoolAddress:     swap.Pool,
				TokenInAddress:  swap.TokenIn,
				TokenOutAddress: swap.TokenOut,
			}

			simplePath = append(simplePath, simpleSwap)
		}

		simplePaths = append(simplePaths, simplePath)
		distributions = append(distributions, business.CalcDistribution(routeSummary.AmountIn, path[0].SwapAmount))
	}

	return &valueobject.SimpleRoute{
		Distributions: distributions,
		Paths:         simplePaths,
	}
}

// CollectTokenAddresses extracts addresses of pool tokens, combines with addresses and returns
func CollectTokenAddresses(poolSet map[string]poolpkg.IPoolSimulator, addresses ...string) []string {
	tokenAddressSet := make(map[string]struct{}, len(poolSet)+len(addresses))
	for _, pool := range poolSet {
		for _, token := range pool.GetTokens() {
			tokenAddressSet[token] = struct{}{}
		}
	}

	for _, address := range addresses {
		tokenAddressSet[address] = struct{}{}
	}

	tokenAddresses := make([]string, 0, len(tokenAddressSet))
	for tokenAddress := range tokenAddressSet {
		tokenAddresses = append(tokenAddresses, tokenAddress)
	}

	return tokenAddresses
}

func ConvertToPathfinderParams(
	whitelistedTokenSet map[string]bool,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.Token,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) finderEntity.FinderParams {
	gasPriceBI, _ := params.GasPrice.Int(nil)

	tokens := lo.MapEntries(tokenByAddress, func(k string, v *entity.Token) (string, entity.Token) {
		return k, *v
	})

	whitelistTokens := lo.MapEntries(whitelistedTokenSet, func(k string, v bool) (string, struct{}) {
		return k, struct{}{}
	})

	prices := CollectTokenPrices(params, priceByAddress, tokenByAddress)

	var l1GasFeePriceOverhead, l1GasFeePricePerPool float64
	if params.L1FeeOverhead != nil && params.L1FeePerPool != nil {
		l1GasFeePriceOverhead = finderUtil.CalcAmountPrice(params.L1FeeOverhead, params.GasToken.Decimals,
			prices[params.GasToken.Address])
		l1GasFeePricePerPool = finderUtil.CalcAmountPrice(params.L1FeePerPool, params.GasToken.Decimals,
			prices[params.GasToken.Address])
	}

	findRouteParams := finderEntity.FinderParams{
		TokenIn:  params.TokenIn.Address,
		TokenOut: params.TokenOut.Address,
		AmountIn: params.AmountIn,

		WhitelistHopTokens: whitelistTokens,

		Pools:      state.Pools,
		SwapLimits: state.SwapLimit,
		Tokens:     tokens,
		Prices:     prices,

		GasIncluded: params.GasInclude,
		GasToken:    params.GasToken.Address,
		GasPrice:    gasPriceBI,

		L1GasFeePriceOverhead: l1GasFeePriceOverhead,
		L1GasFeePricePerPool:  l1GasFeePricePerPool,

		ClientId:                   params.ClientId,
		OnlySinglePath:             params.OnlySinglePath,
		ReturnAMMBestPath:          params.EnableAlphaFee,
		EnableHillClimbForAlphaFee: params.EnableHillClaimForAlphaFee,
	}

	return findRouteParams
}

func CollectTokenPrices(
	params *types.AggregateParams,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	tokenByAddress map[string]*entity.Token,
) map[string]float64 {
	prices := map[string]float64{}

	for tokenAddress := range tokenByAddress {
		if tokenAddress == params.TokenIn.Address {
			prices[tokenAddress] = params.TokenInPriceUSD
			continue
		}

		if tokenAddress == params.TokenOut.Address {
			prices[tokenAddress] = params.TokenOutPriceUSD
			continue
		}

		if tokenAddress == params.GasToken.Address {
			prices[tokenAddress] = params.GasTokenPriceUSD
			continue
		}

		onChainPrice, ok := priceByAddress[tokenAddress]
		if ok && onChainPrice != nil && onChainPrice.USDPrice.Buy != nil {
			tokenPrice, _ := onChainPrice.USDPrice.Buy.Float64()
			prices[tokenAddress] = tokenPrice
			continue
		}
	}

	return prices
}

func GetPrice(
	tokenAddress string,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	isBuyPrice bool,
) float64 {
	onChainPrice, ok := priceByAddress[tokenAddress]
	if ok && onChainPrice != nil {
		if isBuyPrice && onChainPrice.USDPrice.Buy != nil {
			tokenPrice, _ := onChainPrice.USDPrice.Buy.Float64()
			return tokenPrice
		}
		if !isBuyPrice && onChainPrice.USDPrice.Sell != nil {
			tokenPrice, _ := onChainPrice.USDPrice.Sell.Float64()
			return tokenPrice
		}
	}

	return float64(0)
}

func ConvertToRouteSummary(params *types.AggregateParams, route *finderEntity.Route) *valueobject.RouteSummary {
	paths := make([][]valueobject.Swap, 0, len(route.Route))
	for _, path := range route.Route {
		swaps := make([]valueobject.Swap, 0, len(path))

		for _, swap := range path {
			swaps = append(swaps, valueobject.Swap{
				Pool:              swap.Pool,
				TokenIn:           swap.TokenIn,
				TokenOut:          swap.TokenOut,
				LimitReturnAmount: swap.LimitReturnAmount,
				SwapAmount:        swap.SwapAmount,
				AmountOut:         swap.AmountOut,
				Exchange:          swap.Exchange,
				PoolLength:        swap.PoolLength,
				PoolType:          swap.PoolType,
				PoolExtra:         swap.PoolExtra,
				Extra:             swap.Extra,
			})
		}

		paths = append(paths, swaps)
	}

	routeSummary := &valueobject.RouteSummary{
		TokenIn:     route.TokenIn,
		AmountIn:    route.AmountIn,
		AmountInUSD: route.AmountInPrice,

		TokenOut:     route.TokenOut,
		AmountOut:    route.AmountOut,
		AmountOutUSD: route.AmountOutPrice,

		Gas:      route.GasUsed,
		GasPrice: params.GasPrice,
		GasUSD:   route.GasFeePrice,
		L1FeeUSD: route.L1GasFeePrice,
		ExtraFee: params.ExtraFee,
		Route:    paths,
	}

	return routeSummary
}

func InitializeFinderEngine(
	config Config,
	aevmClient aevmclient.Client,
) (finderEngine.IFinder, finderEngine.IFinalizer, error) {
	customFuncs := routerpoolpkg.NewCustomFuncs(config.Aggregator.DexUseAEVM)

	finderOptions := config.Aggregator.FinderOptions
	var baseFinder finderEngine.IFinder

	spfaFinder, err := spfav2.NewFinder(spfav2.Config{
		MaxHops:                      finderOptions.MaxHops,
		DistributionPercent:          finderOptions.DistributionPercent,
		MaxPathsInRoute:              finderOptions.MaxPathsInRoute,
		MaxPathsInFallbackRoute:      finderOptions.MaxPathsInFallbackRoute,
		MaxPathsToGenerate:           finderOptions.MaxPathsToGenerate,
		MaxPathsToReturn:             finderOptions.MaxPathsToReturn,
		MinPartUSD:                   finderOptions.MinPartUSD,
		ExtraPathsPerNodeByTokens:    finderOptions.ExtraPathsPerNodeByTokens,
		FullAmountGeneratePathsPrice: finderOptions.FullAmountGeneratePathsPrice,
	})
	if err != nil {
		return nil, nil, err
	}
	spfaFinder.SetCustomFuncs(customFuncs)
	baseFinder = spfaFinder

	if finderOptions.Type == valueobject.FinderTypes.RetryDynamicPools {
		baseFinder = retry.NewRetryFinder(baseFinder)
	}

	if config.Aggregator.FeatureFlags.IsHillClimbEnabled {
		baseFinder = hillclimb.NewHillClimbFinder(
			baseFinder,
			int(finderOptions.HillClimbIteration),
			finderOptions.HillClimbMinPartUSD,
		)
	}

	if config.Aggregator.FeatureFlags.IsDerivativeHillClimbEnabled {
		baseFinder = hillclimb.NewDerivativeFinder(
			baseFinder,
			finderOptions.DerivativeHillClimbIteration,
			finderOptions.DerivativeHillClimbImproveThreshold,
			config.Aggregator.DexUseAEVM,
		)
	}

	if config.Aggregator.FeatureFlags.IsMergeDuplicateSwapEnabled {
		baseFinder = mergeswap.NewFinder(baseFinder)
	}

	baseFinder = aevm.NewAEVMLocalFinder(
		baseFinder,
		aevmClient,
		finderOptions,
	)

	routeFinalizer := findroute.NewFeeReductionRouteFinalizer(
		safetyquote.NewSafetyQuoteReduction(config.SafetyQuoteConfig),
		alphafee.NewAlphaFeeCalculation(config.AlphaFeeConfig, customFuncs),
		customFuncs,
	)

	return baseFinder, routeFinalizer, nil
}

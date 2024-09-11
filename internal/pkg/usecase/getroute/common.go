package getroute

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/samber/lo"

	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
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
	priceUSDByAddress map[string]float64,
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

	prices := CollectTokenPrices(params, priceUSDByAddress, priceByAddress, tokenByAddress)

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
	}

	return findRouteParams
}

func CollectTokenPrices(
	params *types.AggregateParams,
	priceUSDByAddress map[string]float64,
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

		// Fallback to legacy price-service
		prices[tokenAddress] = priceUSDByAddress[tokenAddress]
	}

	return prices
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

		ExtraFee: params.ExtraFee,

		Route: paths,
	}

	return routeSummary
}

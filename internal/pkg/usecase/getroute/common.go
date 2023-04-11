package getroute

import (
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
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

// collectTokenAddresses extracts addresses of pool tokens, combines with addresses and returns
func collectTokenAddresses(poolSet map[string]poolpkg.IPool, addresses ...string) []string {
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

// extractBestRoute returns the best routes among routes
func extractBestRoute(routes []*valueobject.Route) *valueobject.Route {
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

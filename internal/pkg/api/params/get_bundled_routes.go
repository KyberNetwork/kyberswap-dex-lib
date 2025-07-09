package params

import "github.com/goccy/go-json"

type GetBundledRoutesParams struct {
	// TokensIn addresses of token to be swapped
	TokensIn []string `form:"tokensIn"`

	// TokensOut addresses of token to be received
	TokensOut []string `form:"tokensOut"`

	// AmountsIn amounts of TokensIn
	AmountsIn []string `form:"amountsIn"`

	// IncludedSources name of sources are included in the route, separated by comma
	IncludedSources string `form:"includedSources"`

	// ExcludedSources name of sources are excluded in the route, separated by comma
	ExcludedSources string `form:"excludedSources"`

	// OnlyScalableSources whether to only include scalable sources and exclude all unscalable sources
	OnlyScalableSources bool `form:"onlyScalableSources"`

	// OnlyDirectPools whether to only include direct pools
	OnlyDirectPools bool `form:"onlyDirectPools"`

	// GasInclude gas is included when finding route
	GasInclude bool `form:"gasInclude"`

	// GasPrice custom gas price
	GasPrice string `form:"gasPrice"`

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools string `form:"excludedPools"`

	ClientId string `form:"clientId"`

	OverridePools json.RawMessage `form:"overridePools"`

	// ExtraWhitelistedTokens name of token addresses are included in whitelisted when finding route, separated by comma
	ExtraWhitelistedTokens string `form:"extraWhitelistedTokens" binding:"max=215"`

	Index string `form:"index"`

	// Origin is the address that will submit the transaction
	Origin string `form:"origin"`
}

type GetBundledRoutesResponse struct {
	RoutesSummary []*RouteSummary `json:"routesSummary"`
	RouterAddress string          `json:"routerAddress"`
}

package params

type GetRoutesParams struct {
	// TokenIn address of token to be swapped
	TokenIn string `form:"tokenIn"`

	// TokenOut address of token to be received
	TokenOut string `form:"tokenOut"`

	// AmountIn amount of TokenIn
	AmountIn string `form:"amountIn"`

	// IncludedSources name of sources are included in the route, separated by comma
	IncludedSources string `form:"includedSources"`

	// ExcludedSources name of sources are excluded in the route, separated by comma
	ExcludedSources string `form:"excludedSources"`

	// ExcludeRFQSources whether to exclude RFQ sources in the route
	ExcludeRFQSources bool `form:"excludeRFQSources"`

	// OnlyScalableSources whether to only include scalable sources and exclude all unscalable sources
	OnlyScalableSources bool `form:"onlyScalableSources"`

	// OnlySinglePath only find & return single path route
	OnlySinglePath bool `form:"onlySinglePath"`

	// GasInclude gas is included when finding route
	GasInclude bool `form:"gasInclude,default=true"`

	// GasPrice custom gas price
	GasPrice string `form:"gasPrice"`

	// FeeAmount custom fee
	FeeAmount string `form:"feeAmount"`

	// ChargeFeeBy custom fee will be charged on currency in or currency out
	ChargeFeeBy string `form:"chargeFeeBy"`

	// IsInBps is true when FeeAmount is in bip base
	IsInBps bool `form:"isInBps"`

	// FeeReceiver address to be received custom fee
	FeeReceiver string `form:"feeReceiver"`

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools string `form:"excludedPools"`

	ClientId string `form:"clientId"`

	// Index type
	Index string `form:"index"`
}

type GetRoutesResponse struct {
	RouteSummary  *RouteSummary `json:"routeSummary"`
	RouterAddress string        `json:"routerAddress"`
}

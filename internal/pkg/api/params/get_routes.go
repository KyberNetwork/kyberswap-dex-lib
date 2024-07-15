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

	// SaveGas best route is a single path route
	SaveGas bool `form:"saveGas"`

	// GasInclude gas is included when finding route
	GasInclude bool `form:"gasInclude"`

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

	// IsPathGeneratorEnabled is true, then router-service will use pregen paths from path-generator service
	IsPathGeneratorEnabled bool `form:"isPathGeneratorEnabled"`

	// IsHillClimbEnabled is true, then router-service will use hill climb finder
	IsHillClimbEnabled bool `form:"isHillClimbEnabled"`

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools string `form:"excludedPools"`

	ClientId string `form:"clientId"`
}

type GetRoutesResponse struct {
	RouteSummary  *RouteSummary `json:"routeSummary"`
	RouterAddress string        `json:"routerAddress"`
}

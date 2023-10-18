package params

type (
	BuildRouteParams struct {
		RouteSummary RouteSummary `json:"routeSummary"`

		// Sender address of sender wallet
		Sender string `json:"sender"`

		// Recipient address of recipient wallet
		Recipient string `json:"recipient"`

		Deadline          int64  `json:"deadline"`
		SlippageTolerance int64  `json:"slippageTolerance"`
		Referral          string `json:"referral"`
		Source            string `json:"source"`

		// enable gas estimation, default is false
		EnableGasEstimation bool `json:"enableGasEstimation"`

		// Permit allows user to swap without approving token beforehand
		Permit string `json:"permit"`
	}
)

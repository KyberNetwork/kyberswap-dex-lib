package params

type (
	BuildRouteParams struct {
		RouteSummary RouteSummary `json:"routeSummary"`

		Sender    string `json:"sender"`    // sender wallet
		Recipient string `json:"recipient"` // recipient wallet

		Permit string `json:"permit"` // allows user to swap without approving token beforehand

		Deadline             int64   `json:"deadline"`
		SlippageTolerance    float64 `json:"slippageTolerance"`    // in bps
		IgnoreCappedSlippage bool    `form:"ignoreCappedSlippage"` // allow slippage up to 100%
		EnableGasEstimation  bool    `json:"enableGasEstimation"`  // enable gas estimation and tx success check

		Source   string `json:"source"`
		Referral string `json:"referral"`
	}
)

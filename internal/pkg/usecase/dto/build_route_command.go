package dto

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type BuildRouteCommand struct {
	RouteSummary valueobject.RouteSummary

	Sender    string
	Recipient string

	Permit []byte

	Deadline            int64
	SlippageTolerance   float64
	EnableGasEstimation bool

	Source   string
	Origin   string
	Referral string
}

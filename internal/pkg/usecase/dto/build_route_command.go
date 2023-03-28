package dto

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type BuildRouteCommand struct {
	RouteSummary valueobject.RouteSummary

	Sender    string
	Recipient string

	Deadline          int64
	SlippageTolerance int64
	Referral          string
	Source            string

	Permit []byte
}

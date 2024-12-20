package dto

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type BuildRouteCommand struct {
	RouteSummary valueobject.RouteSummary
	Checksum     uint64

	Sender    string
	Recipient string

	Deadline          int64
	SlippageTolerance int64
	Referral          string
	Source            string

	EnableGasEstimation bool

	Permit []byte
}

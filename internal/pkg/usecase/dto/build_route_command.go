package dto

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type BuildRouteCommand struct {
	RouteSummary      *valueobject.RouteSummary
	OriginalAmountOut *big.Int

	Sender    string
	Recipient string
	Origin    string

	Permit []byte

	Deadline            int64
	SlippageTolerance   float64
	EnableGasEstimation bool

	Source   string
	Referral string
}

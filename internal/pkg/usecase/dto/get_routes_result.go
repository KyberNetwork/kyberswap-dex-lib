package dto

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type GetRoutesResult struct {
	RouteSummary  *valueobject.RouteSummary `json:"routeSummary"`
	RouterAddress string                    `json:"routerAddress"`
}

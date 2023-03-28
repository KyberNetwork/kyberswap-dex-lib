package dto

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type GetRoutesResult struct {
	RouteSummary  *valueobject.RouteSummary `json:"routeSummary"`
	RouterAddress string                    `json:"routerAddress"`
}

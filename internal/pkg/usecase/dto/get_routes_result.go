package dto

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type GetRoutesResult struct {
	RouteSummary  *valueobject.RouteSummary `json:"routeSummary"`
	Checksum      uint64                    `json:"checksum"`
	RouterAddress string                    `json:"routerAddress"`
}

type GetBundledRoutesResult struct {
	RoutesSummary []*valueobject.RouteSummary `json:"routesSummary"`
	RouterAddress string                      `json:"routerAddress"`
}

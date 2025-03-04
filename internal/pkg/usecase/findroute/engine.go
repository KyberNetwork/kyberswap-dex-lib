package findroute

import (
	"context"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	ErrInvalidSwap   = errors.New("invalid swap")
	ErrRouteNotFound = errors.New("route not found")
)

type PathFinderEngine struct {
	baseFinder                 IFinder
	FeeReductionRouteFinalizer IFinalizer
}

func NewPathFinderEngine(
	coreFinder IFinder,
	FeeReductionRouteFinalizer IFinalizer,
) *PathFinderEngine {
	return &PathFinderEngine{
		baseFinder:                 coreFinder,
		FeeReductionRouteFinalizer: FeeReductionRouteFinalizer,
	}
}

func (p *PathFinderEngine) SetFinder(finder IFinder) {
	p.baseFinder = finder
}

func (p *PathFinderEngine) SetFinalizer(finalizer IFinalizer) {
	p.FeeReductionRouteFinalizer = finalizer
}

func (p *PathFinderEngine) GetFinalizer() IFinalizer {
	return p.FeeReductionRouteFinalizer
}

func (p *PathFinderEngine) Find(
	ctx context.Context,
	input Input,
	data FinderData,
	requestParams *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	routes, err := p.baseFinder.Find(ctx, input, data)
	if err != nil {
		return nil, err
	}

	route := extractBestRoute(routes)
	if route == nil {
		return nil, ErrRouteNotFound
	}

	data.Refresh()

	return p.FeeReductionRouteFinalizer.FinalizeRoute(ctx, route, data.PoolBucket.PerRequestPoolsByAddress, data.SwapLimits, requestParams)
}

func extractBestRoute(routes []*valueobject.Route) *valueobject.Route {
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

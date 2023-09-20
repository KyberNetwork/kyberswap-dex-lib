package spfa

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	defaultSpfaMaxHops             uint32  = 3
	defaultSpfaDistributionPercent uint32  = 5
	defaultSpfaMinPartUSD          float64 = 500

	// number of paths to generate and return in BestPathExactIn, not meant to be configurable
	defaultSpfaMaxPathsToGenerate uint32 = 1
	defaultSpfaMaxPathsToReturn   uint32 = 1
)

// spfaFinder finds route using Shortest spfaPath Faster Algorithm (SPFA)
type spfaFinder struct {
	// maxHops maximum hops performed
	maxHops uint32

	// distributionPercent the portion of amountIn to split. It should be a divisor of 100.
	// e.g. distributionPercent = 5, we split amountIn into portions of 5%, 10%, 15%, ..., 100%
	distributionPercent uint32

	// minPartUSD minimum amount in USD of each part
	minPartUSD float64
}

func NewSPFAFinder(maxHops, distributionPercent uint32, minPartUSD float64) *spfaFinder {
	return &spfaFinder{maxHops, distributionPercent, minPartUSD}
}

func NewDefaultSPFAFinder() *spfaFinder {
	return NewSPFAFinder(defaultSpfaMaxHops, defaultSpfaDistributionPercent, defaultSpfaMinPartUSD)
}

func (f *spfaFinder) Find(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfaFinder.Find")
	defer span.End()

	bestRoute, err := f.bestRouteExactIn(ctx, input, data)
	if err != nil {
		return nil, err
	}
	if bestRoute == nil {
		return nil, nil
	}

	return []*valueobject.Route{bestRoute}, nil
}

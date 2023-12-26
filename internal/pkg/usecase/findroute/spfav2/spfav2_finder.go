package spfav2

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	defaultSpfav2MaxHops             uint32 = 3
	defaultSpfav2DistributionPercent uint32 = 5
	defaultSpfav2MaxPathsInRoute     uint32 = 20
	defaultSpfav2MaxPathsToGenerate  uint32 = 5
	defaultSpfav2MaxPathsToReturn    uint32 = 200

	defaultSpfav2MinPartUSD              float64 = 500
	defaultSpfav2MinThresholdAmountInUSD float64 = 0
	defaultSpfav2MaxThresholdAmountInUSD float64 = 100_000_000
)

var (
	defaultSpfav2WhitelistedTokenSet = map[string]bool{}
)

// spfav2Finder finds route by splitting amountIn and sequentially finding the best paths multiple times
type spfav2Finder struct {
	// maxHops maximum hops performed
	maxHops uint32

	// whitelistedTokenSet tokens that are allowed to be used as hop tokens
	whitelistedTokenSet map[string]bool

	// distributionPercent the portion of amountIn to split. It should be a divisor of 100.
	// e.g. distributionPercent = 5, we split amountIn into portions of 5%, 10%, 15%, ..., 100%
	distributionPercent uint32

	// max number of paths in a route
	maxPathsInRoute uint32

	// number of paths to generate in GenKthBestPaths
	maxPathsToGenerate uint32

	// number of paths to return in GenKthBestPaths
	maxPathsToReturn uint32

	// minPartUSD minimum amount in USD of each part
	minPartUSD float64

	// if minThreshold < amountInUSD < maxThreshold: run similar to spfaFinder, else run the new strategy
	minThresholdAmountInUSD float64
	maxThresholdAmountInUSD float64

	getGeneratedBestPaths func(sourceHash uint64, tokenIn string, tokenOut string) []*entity.MinimalPath
}

func NewSPFAv2Finder(
	maxHops uint32,
	whitelistedTokenSet map[string]bool,
	distributionPercent uint32,
	maxPathsInRoute uint32,
	maxPathsToGenerate,
	maxPathsToReturn uint32,
	minPartUSD float64,
	minThresholdAmountInUSD float64,
	maxThresholdAmountInUSD float64,
	getGeneratedBestPaths func(sourceHash uint64, tokenIn string, tokenOut string) []*entity.MinimalPath,
) *spfav2Finder {
	return &spfav2Finder{
		maxHops:                 maxHops,
		whitelistedTokenSet:     whitelistedTokenSet,
		distributionPercent:     distributionPercent,
		maxPathsInRoute:         maxPathsInRoute,
		maxPathsToGenerate:      maxPathsToGenerate,
		maxPathsToReturn:        maxPathsToReturn,
		minPartUSD:              minPartUSD,
		minThresholdAmountInUSD: minThresholdAmountInUSD,
		maxThresholdAmountInUSD: maxThresholdAmountInUSD,
		getGeneratedBestPaths:   getGeneratedBestPaths,
	}
}

func NewDefaultSPFAv2Finder() *spfav2Finder {
	return NewSPFAv2Finder(
		defaultSpfav2MaxHops,
		defaultSpfav2WhitelistedTokenSet,
		defaultSpfav2DistributionPercent,
		defaultSpfav2MaxPathsInRoute,
		defaultSpfav2MaxPathsToGenerate,
		defaultSpfav2MaxPathsToReturn,
		defaultSpfav2MinPartUSD,
		defaultSpfav2MinThresholdAmountInUSD,
		defaultSpfav2MaxThresholdAmountInUSD,
		func(sourceHash uint64, tokenIn string, tokenOut string) []*entity.MinimalPath { return nil },
	)
}

func (f *spfav2Finder) Find(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.Find")
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

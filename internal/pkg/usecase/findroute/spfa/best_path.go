package spfa

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// bestPathExactIn Find the best path to token out
// we represent graph node as pair (token, hops) because we want to handle negative cycles
// edges are now from (X, hop) to (Y, hop + 1) => make the graph a DAG => no cycle
// Perform SPFA from (tokenIn,0) to find the best path to token out
// Because we are performing SPFA and that only edges between (X, hop) -> (Y, hop+1) exist
// => The order of traversal looks like: (, 0) ... (, 0) (, 1) ... (, 1) ... (, hop-1), ... (,hop-1), (,hop)... (, hop)
func (f *spfaFinder) bestPathExactIn(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*valueobject.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "spfaFinder.bestPathExactIn")
	defer span.End()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// only pick one best path, so set maxPathsToGenerate = 1.
	paths, err := common.GenKthBestPaths(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut, f.maxHops, defaultSpfaMaxPathsToGenerate, defaultSpfaMaxPathsToReturn)
	if err != nil {
		return nil, err
	}
	var bestPath *valueobject.Path
	for _, path := range paths {
		if path != nil && path.CompareTo(bestPath, input.GasInclude && data.PriceUSDByAddress[path.Output.Token] != 0) < 0 {
			bestPath = path
		}
	}
	return bestPath, nil
}

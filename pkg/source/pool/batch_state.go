package pool

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// IBatchPoolStateTracker refreshes many pools of one dex with a single upstream request
// (e.g. an HTTP batch API). pool-service calls it once per batch task.
//
// The returned slices are aligned with pools: errs[i] != nil means pools[i] failed and must be
// skipped (its stored state is kept); other pools are unaffected.
type IBatchPoolStateTracker interface {
	GetNewPoolStates(ctx context.Context, pools []entity.Pool) (newPools []entity.Pool, errs []error)
}

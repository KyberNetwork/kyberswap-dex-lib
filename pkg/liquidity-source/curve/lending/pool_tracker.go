package lending

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/llamma"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, llamma.NewPoolTracker)

// curve-lending registers llamma.NewPoolTracker directly (it has no PoolTracker type of
// its own), so it inherits llamma.PoolTracker.GetNewPoolStateWithOverrides for free. This
// assertion documents that curve-lending trackers satisfy pool.IPoolTrackerWithOverrides
// too.
var _ pool.IPoolTrackerWithOverrides = (*llamma.PoolTracker)(nil)

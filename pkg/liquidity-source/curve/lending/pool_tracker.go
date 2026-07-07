package lending

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/llamma"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, llamma.NewPoolTracker)

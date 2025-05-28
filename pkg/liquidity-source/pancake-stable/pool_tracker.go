package pancakestable

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE(DexType, curve.NewPoolTracker)

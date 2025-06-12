package euler

import (
	eulerswap "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE(DexType, eulerswap.NewPoolTracker)

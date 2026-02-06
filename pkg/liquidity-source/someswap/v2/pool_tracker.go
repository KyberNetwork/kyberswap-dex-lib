package someswapv2

import (
	v1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/someswap/v1"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, v1.NewPoolTracker)

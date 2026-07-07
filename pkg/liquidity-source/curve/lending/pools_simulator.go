package lending

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/llamma"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory0(DexType, llamma.NewPoolSimulator)

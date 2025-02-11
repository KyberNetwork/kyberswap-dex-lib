package ironstable

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
)

var _ = pool.RegisterFactory0(DexTypeIronStable, saddle.NewPoolSimulator)

package pancakestable

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory0(DexType, base.NewPoolSimulator)

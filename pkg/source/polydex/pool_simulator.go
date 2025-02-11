package polydex

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
)

var _ = pool.RegisterFactory0(DexTypePolydex, uniswap.NewPoolSimulator)

package frxusd

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory(DexType, erc4626.NewPoolSimulatorFromParams)

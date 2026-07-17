package erc7575

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The simulator is pure arithmetic over Tokens[0]/Tokens[1] and the cached rates, agnostic to whether the
// share equals the vault, so it is reused verbatim from erc4626.
var _ = pool.RegisterFactory0(DexType, erc4626.NewPoolSimulator)

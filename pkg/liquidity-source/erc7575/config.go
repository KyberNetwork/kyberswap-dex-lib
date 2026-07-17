package erc7575

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
)

// Config reuses erc4626's config shape (chainId, dexId, vaults keyed by vault entrypoint address).
type Config = erc4626.Config

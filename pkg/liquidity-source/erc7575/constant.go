package erc7575

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// DexType handles ERC7575/ERC7540-style vaults where the share ERC20 is a separate contract from the vault
// entrypoint (share() != vault). All pricing/state logic is reused from the erc4626 package; only the pool
// wiring differs: Tokens[0] is the share token, the pool Address is the vault entrypoint (call target), and
// totalSupply is read from the share token.
const DexType = valueobject.ExchangeERC7575

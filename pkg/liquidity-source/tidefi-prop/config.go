package tidefiprop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is the per-dex-id configuration. TideFi is a single fixed contract
// backing every pair (not a per-pair router/lens), so the supported token
// set is config-static rather than on-chain discovered -- see
// pools_list_updater.go.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Address is the TideFi swapper contract: both the source of quote()
	// and the swap entrypoint the executor calls.
	Address string `json:"address"`

	Tokens []string `json:"tokens"`
	Buffer int64    `json:"buffer"`
}

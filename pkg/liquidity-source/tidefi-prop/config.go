package tidefiprop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is the per-dex-id configuration. TideFi is a single fixed contract
// backing every pair (not a per-pair router/lens); the supported token set
// is discovered from the Taker API's tidefi_markets message rather than
// on-chain -- see pools_list_updater.go.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Address is the TideFi swapper contract: both the source of quote()
	// and the swap entrypoint the executor calls.
	Address string `json:"address"`

	// Vault is the actual liquidity-holding/quoting engine Address forwards
	// to under the hood -- a hardcoded bytecode literal inside Address's
	// contract (confirmed via decompilation), not discoverable through any
	// getter. Reserves are read from Vault's own token balances, not
	// Address's (which never holds funds itself).
	Vault string `json:"vault"`

	// TakerAPIURL is the Taker API websocket endpoint; AuthToken is its
	// bearer token. Both are provisioned per-integrator by TideFi at
	// onboarding (see docs.ulmomarkets.com/tidefi_propamm.html#taker-api),
	// not derivable or public.
	TakerAPIURL string `json:"takerApiUrl"`
	AuthToken   string `json:"authToken"`

	Buffer int64 `json:"buffer"`
}

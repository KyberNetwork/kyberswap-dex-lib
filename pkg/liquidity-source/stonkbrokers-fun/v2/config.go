package stonkbrokersfunv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config lists the fixed Smart Launch V2 pads for this chain. Pads are
// statically deployed per quote lane -- there is no factory to discover them
// from (see output/explorer.md's `findings.explorer.pool_discovery:
// view_enum`). dex-implement-tracker's pool list updater enumerates
// launchCount() and getLaunch(id) directly on each configured pad.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Pads is the fixed set of StonkSafeLaunchpadV2 addresses to scan, one
	// per quote lane (WETH/STONK/USDG/GME/NVDA/AAPL/SPCX/USO on Robinhood
	// Chain -- see output/explorer.md's findings.contracts.other).
	Pads []string `json:"pads"`

	// Lens is the shared SafeLaunchLensV2 address used for quoting.
	Lens string `json:"lens"`
}

package everlongcvamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config lists the CvammALM deployments to integrate on one chain. Everything that
// prices a fill (curve band, coordinate, scale, anchor, reserves, fees, pause) is read
// from the ALM itself by the lister/tracker, so adding a chain or a venue is pure
// configuration — no code constants per deployment.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainID"`
	ALMs    []ALMConfig         `json:"alms"`
}

// ALMConfig describes one CvammALM venue.
type ALMConfig struct {
	// Address is the CvammALM proxy: the venue itself — sole book, swap entrypoint,
	// approval target (it pulls the input with transferFrom).
	Address string `json:"address"`
	// Adapter optionally points at the ClammPoolSwapAdapter (ISwapRouter02.exactInputSingle
	// shim). Note it fails closed on partial fills; direct ALM.swap does not.
	Adapter string `json:"adapter,omitempty"`
	// Quoter optionally points at the stateless CvammQuoter for off-band verification.
	Quoter string `json:"quoter,omitempty"`
	// GasStableIn/GasVolatileIn override the default per-direction gas estimates.
	GasStableIn   int64 `json:"gasStableIn,omitempty"`
	GasVolatileIn int64 `json:"gasVolatileIn,omitempty"`
}

package everlongcollvault

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config describes one Everlong CollVault deployment (one CollateralRebalancer +
// CollateralRebalancerSwapper pair). One config entry per chain.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainID"`
	// Rebalancer is the CollateralRebalancer proxy (exchangeState; resolves the
	// CollVault and the settlement Swapper on-chain).
	Rebalancer string `json:"rebalancer"`
	// Stable/Volatile are the two swap legs (e.g. NECT 18d / WBTC 8d on Berachain).
	Stable   string `json:"stable"`
	Volatile string `json:"volatile"`
	// CurveParams overrides the built-in per-chain deployed-curve constants; leave nil
	// to use the built-ins (required for chains without a built-in entry).
	CurveParams *CurveParams `json:"curveParams,omitempty"`
	// GasSwap overrides the default per-swap gas estimate when non-zero.
	GasSwap int64 `json:"gasSwap,omitempty"`
}

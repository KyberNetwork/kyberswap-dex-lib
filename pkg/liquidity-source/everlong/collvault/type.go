package everlongcollvault

import (
	"math/big"
)

type StaticExtra struct {
	// Resolved on-chain at listing time from the configured rebalancer.
	Rebalancer string `json:"reb"`
	Swapper    string `json:"swapper"` // CollateralRebalancerSwapper — swap call + approval target
	CollVault  string `json:"cv"`
	ALM        string `json:"alm"` // the swapper's ALM adapter (reserves/valuation reads)
	// 18 - collVault.assetDecimals() (constant per deployment).
	CvDecimalsOffset uint8 `json:"cdo"`
	// The deployed rebalancer's frozen curve constants.
	CurveParams CurveParams `json:"curve"`
}

// Extra is the per-refresh vault snapshot the simulator prices from.
type Extra = VaultState

// SwapInfo carries the exact fill so UpdateBalance replays it without recomputation,
// and the executor learns the exact contract args.
//
// Leverage (volatile -> stable): the executor calls
// swapVolatileForStable(CollVaultShares, maxStableIn, maxVolatileIn, minNetStableOut, receiver)
// paying VolatileLeg wei of the volatile token; the flash-minted stable leg nets out and
// the caller receives the quoted net stable.
//
// Deleverage (stable -> volatile): the executor calls
// swapStableForVolatile(GrossStableIn, maxNetStableIn, minVolatileOut, receiver).
// NOTE the swapper transferFroms maxNetStableIn UP FRONT and refunds the excess within
// the same call — the payer must hold and approve maxNetStableIn (>= the true net; a
// snug cap risks an on-chain revert on state drift), even though only the net is spent.
type SwapInfo struct {
	IsLeverage bool `json:"lev"`
	// CollVaultShares: shares minted (leverage) or burned (deleverage).
	CollVaultShares *big.Int `json:"shares"`
	// GrossStableIn: deleverage only — the stableDebtIn contract argument.
	GrossStableIn *big.Int `json:"grossIn,omitempty"`
	// StableLeg / VolatileLeg: token amounts entering (leverage) or leaving
	// (deleverage) the ALM.
	StableLeg   *big.Int `json:"stableLeg"`
	VolatileLeg *big.Int `json:"volatileLeg"`
	// AlmShares: ALM shares minted/burned by the CollVault for this fill.
	AlmShares *big.Int `json:"almShares"`
	// Post-fill position state.
	NewCollateral *big.Int `json:"newC"`
	NewDebt       *big.Int `json:"newD"`
}

// PoolMeta tells the executor where and how to settle.
type PoolMeta struct {
	Swapper    string `json:"swapper"`
	Rebalancer string `json:"rebalancer"`
	// UpfrontNetStablePull: deleverage pulls maxNetStableIn up front (see SwapInfo).
	UpfrontNetStablePull bool `json:"upfrontNetStablePull"`
}

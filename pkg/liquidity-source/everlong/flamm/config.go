package everlongflamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is one chain's Everlong FLAMM integration. Pools are discovered from the hook registry (hook_registry.go),
// confirmed against Factory and listed only when their whole wiring is registered; everything that prices a fill is
// read from the pool at a pinned block by the tracker. The margins trade quote coverage for robustness between the
// quote and the fill's block: none of them changes a quoted amount, they only refuse. An unset margin takes its
// default (constant.go); an explicit 0 turns it off.
//
// The keys are decoded from pool-service's properties map by pool.PropertiesToStruct, which matches them
// case-insensitively only while the struct stays at or below goccy/go-json's optimized field count
// (internal/decoder.allowOptimizeMaxFieldLen = 16; above it only the exact tag matches). Every key below is
// therefore accepted in any case -- the spelling of its tag, and the Go field name the README's table uses --
// while the count stays within the limit: the gas overrides are nested in Gas to keep it at thirteen, and
// TestConfigFieldCount fails if a field is added past it. DexID is tagged with the exact key the lister and
// tracker factories inject (properties["DexID"], poollist/pooltrack factory.go), so the dex id lands through the
// exact-tag lookup as well and does not depend on that limit at all.
type Config struct {
	DexID   string              `json:"DexID"`
	ChainID valueobject.ChainID `json:"chainID"`
	// Factory is the FLAMMFactory the registered pools are confirmed against (FLAMMFactory.isPool); it must be a
	// registered deployment's.
	Factory string `json:"factory"`
	// Pools optionally restricts listing to these pool addresses (still intersected with the registered ones).
	Pools []string `json:"pools,omitempty"`

	// LeverRouting enables the leverage venue (adapter venue 1). Off, only the swap venue is quoted.
	LeverRouting bool `json:"leverRouting,omitempty"`
	// LeverMinEdgeBps keeps a routed quote on the swap venue unless the leverage venue pays more than this many bps
	// above it, so the leverage venue's larger gas is not spent for a thin gain (constant.go documents the sizing).
	LeverMinEdgeBps *uint64 `json:"leverMinEdgeBps,omitempty"`
	// PriceBandMarginBps is how far the pool asset's feed answer may move before a fill is refused: the price bands
	// are tightened by it, and a fill whose size or whose other gates the feed prices -- a swap sell, every leverage
	// fill -- is re-run at the answer moved both ways and refused unless it settles identically.
	PriceBandMarginBps *uint64 `json:"priceBandMarginBps,omitempty"`
	// PriceAgeMarginSec refuses every quote once a feed round the fill reads would be stale this many seconds
	// after the quote.
	PriceAgeMarginSec *uint64 `json:"priceAgeMarginSec,omitempty"`
	// SpreadAgeMarginSec refuses a leverage quote once the keeper's spread post would lapse this many seconds
	// after the quote.
	SpreadAgeMarginSec *uint64 `json:"spreadAgeMarginSec,omitempty"`
	// DebtDriftSec re-runs the chosen fill this many seconds later (Morpho interest accrued exactly) and refuses
	// it unless it still settles; a swap must also return the same amounts.
	DebtDriftSec *uint64 `json:"debtDriftSec,omitempty"`
	// MaxSnapshotAgeSec refuses every quote once the clock is this many seconds past the refresh's block
	// timestamp: a refresh that keeps failing leaves pool-service with the last entity, which no longer sees the
	// fills since.
	MaxSnapshotAgeSec *uint64 `json:"maxSnapshotAgeSec,omitempty"`
	// QuoteDonatedVenues quotes a pool whose venue account holds Morpho collateral or supply shares beyond what
	// the Router manages (a donation made on its behalf). Off, such a pool is refused (Policy.QuoteDonatedVenues).
	QuoteDonatedVenues bool `json:"quoteDonatedVenues,omitempty"`

	// Gas overrides the measured gas defaults per venue and direction.
	Gas GasConfig `json:"gas,omitempty"`
}

// GasConfig overrides one gas default per venue and direction (a zero keeps the measured default, constant.go). A
// clipped swap sell adds SwapSellCapEval per curve solve of the hook's cap bisection and SwapSellPass per funding
// pass after the first.
type GasConfig struct {
	SwapSell        int64 `json:"swapSell,omitempty"`
	SwapSellCapEval int64 `json:"swapSellCapEval,omitempty"`
	SwapSellPass    int64 `json:"swapSellPass,omitempty"`
	SwapBuy         int64 `json:"swapBuy,omitempty"`
	LeverUp         int64 `json:"leverUp,omitempty"`
	LeverDown       int64 `json:"leverDown,omitempty"`
}

// policy is the part of the configuration the simulator enforces; the tracker stamps it into Extra with every
// unset margin at its default (a zero gas keeps the default, applied by the simulator).
func (c *Config) policy() Policy {
	margin := func(v *uint64, def uint64) uint64 {
		if v == nil {
			return def
		}
		return *v
	}
	return Policy{LeverRouting: c.LeverRouting,
		LeverMinEdgeBps:    margin(c.LeverMinEdgeBps, defaultLeverMinEdgeBps),
		PriceBandMarginBps: margin(c.PriceBandMarginBps, defaultPriceBandMarginBps),
		PriceAgeMarginSec:  margin(c.PriceAgeMarginSec, defaultPriceAgeMarginSec),
		SpreadAgeMarginSec: margin(c.SpreadAgeMarginSec, defaultSpreadAgeMarginSec),
		DebtDriftSec:       margin(c.DebtDriftSec, defaultDebtDriftSec),
		MaxSnapshotAgeSec:  margin(c.MaxSnapshotAgeSec, defaultMaxSnapshotAgeSec),
		QuoteDonatedVenues: c.QuoteDonatedVenues,
		GasSwapSell:        c.Gas.SwapSell, GasSwapSellCapEval: c.Gas.SwapSellCapEval,
		GasSwapSellPass: c.Gas.SwapSellPass, GasSwapBuy: c.Gas.SwapBuy, GasLeverUp: c.Gas.LeverUp,
		GasLeverDown: c.Gas.LeverDown}
}

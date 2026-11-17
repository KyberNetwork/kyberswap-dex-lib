package everlongflamm

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is per dex id. It names no pool or venue: every pool a configured Factory created is found on chain and
// each pair is priced by the pool's own quote. What it names are the trust gates: the factories, the price feeds,
// the pool implementations and the hook code families quoted.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`

	// Factories whose pools are listed, several per chain. `isPool` there is a pool's provenance, read on every
	// refresh.
	Factories []string `json:"factories"`

	// PriceFeeds are the price feeds quoted, by address, each chain's protocol feeds: a pool behind any other is left
	// unrouted, and with none named nothing is routed. A pool's creator picks its feed at birth and the hook prices
	// from it, so a feed that tells a probe from a swap makes an honest hook quote one price and fill another. A
	// code hash would not vet it: a build of the protocol's feed takes any aggregators at construction.
	PriceFeeds []string `json:"priceFeeds"`

	// CodeVersions are the `poolCodeVersion()` values quoted; empty: [1]. A pool behind any other implementation
	// is left unrouted until the list names it.
	CodeVersions []uint32 `json:"codeVersions,omitempty"`

	// HookFamilies are the pricing-hook code families quoted: a pair is routed only where the hook pricing it (the
	// pool's pair hook for the pairs its mask names, else the swap hook) runs code of a listed family, and the gas
	// table has cells for that build. Empty: the protocol-listed builds (defaultHookFamilies); a list replaces them,
	// so a new family is an entry here beside them, never code, and is routed once the table measures it.
	HookFamilies []HookFamily `json:"hookFamilies,omitempty"`

	// VenueKinds narrows the Router's venue kinds routed (0 Morpho Blue, 1 Aave V3, 2 Venus Core, 3 Lista Moolah)
	// to these, among the kinds the chain routes (chainKinds); empty: all of them. It never adds a kind: a pair with
	// a live venue of a kind not routed is not routed.
	VenueKinds []uint8 `json:"venueKinds,omitempty"`

	// AnyHook adds the builds the gas table measured (gasFamilies) to HookFamilies, so a list that names others
	// still routes them, and lifts no other gate: a hook of no measured build has no gas cells and is never routed.
	AnyHook bool `json:"anyHook,omitempty"`

	// SkipFrontPriced does nothing in this build: the pairs a PropAMM front prices (the swap hook of a pool whose
	// cheap lane is on, or any pricer the Factory lists as a front) are never routed, whatever it says. It stays for
	// the PropAMM listings' own entries.
	SkipFrontPriced *bool `json:"skipFrontPriced,omitempty"`

	// Taker is the address the pool sees every probe come from, as the hook prices the swap for its caller: the
	// sampler runs there by a state override, and the plain path sends its calls from it. Empty: the KyberSwap
	// executor.
	Taker string `json:"taker,omitempty"`

	// ProbeGas is the sampler's cap on each call to the pool or hook at a pool's first refresh, doubled for a pool
	// whose probe starves at it, up to maxProbeGas, and back once an answer spends under 7/8 of it on every probe;
	// zero: defaultProbeGas. The plain path's calls take plainProbeGas each.
	ProbeGas uint64 `json:"probeGas,omitempty"`

	// SampleGas is the gas of the sampler's eth_call, stated as a node's own default may be far above or below;
	// zero: defaultSampleGas. It is raised to minSampleGas, the least that lets a probe at maxProbeGas pass the
	// sampler's gas gate.
	SampleGas uint64 `json:"sampleGas,omitempty"`

	// DisableSampler sends every refresh down the plain path: one eth_call a probe, in JSON-RPC batches, with no
	// bisection. Off by default: the sampler prices a pool in one call, and a node that refuses it falls back to the
	// plain path by itself.
	DisableSampler bool `json:"disableSampler,omitempty"`

	// HaircutBps is taken off every amount interpolated between two probed sizes; nil: defaultHaircutBps.
	HaircutBps *uint64 `json:"haircutBps,omitempty"`

	// Rungs is how many sizes each direction is seeded at past its first on every refresh after a pool's first,
	// beside one size under the last ladder's first point; zero: defaultRungs, and held to [minRungs, maxRungs] and to
	// the direction's probe budget. A first refresh seeds firstRefreshSeeds sizes, the plain path
	// plainSeedsPerDirection.
	Rungs int `json:"rungs,omitempty"`
}

// HookFamily is one hook code family: the keccak256 of a hook's runtime code with its immutables zeroed, so every
// deployment of one build shares it, and the shape its output states, which settles its ladders. The gas of its
// fills is the table's (gas_table.go), by its CodeHash.
type HookFamily struct {
	Name       string   `json:"name"`
	CodeHash   string   `json:"codeHash"`
	Immutables [][2]int `json:"immutables,omitempty"` // [offset, length] byte ranges of the runtime code
	// Shape is how the family's output behaves: "everlong" (shapeEverlong), the only one. Empty or unknown: its pairs
	// are not routed.
	Shape string `json:"shape,omitempty"`
	Front bool   `json:"front,omitempty"` // a front's build: not routed by this build, kept for the PropAMM listings
}

// UnmarshalJSON refuses a config that names a price feed which is not an address, or is the zero address: read as
// an address, either would be the zero address, which no feed is.
func (c *Config) UnmarshalJSON(data []byte) error {
	type plain Config
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	for _, f := range p.PriceFeeds {
		if !common.IsHexAddress(f) || common.HexToAddress(f) == (common.Address{}) {
			return fmt.Errorf("everlong-flamm: price feed %q is not a non-zero address", f)
		}
	}
	*c = Config(p)
	return nil
}

func (c *Config) families() []HookFamily {
	fs := orDefault(c.HookFamilies, defaultHookFamilies)
	if !c.AnyHook {
		return fs
	}
	out := slices.Clone(fs)
	for _, g := range gasFamilies {
		if !slices.ContainsFunc(out, func(f HookFamily) bool { return strings.EqualFold(f.CodeHash, g.CodeHash) }) {
			out = append(out, g)
		}
	}
	return out
}

func (c *Config) codeVersionOk(v uint32) bool {
	return slices.Contains(orDefault(c.CodeVersions, defaultCodeVersions), v)
}

func (c *Config) taker() common.Address {
	if c.Taker == "" {
		return defaultTaker
	}
	return common.HexToAddress(c.Taker)
}

func (c *Config) probeGas() uint64 {
	if c.ProbeGas == 0 {
		return defaultProbeGas
	}
	return c.ProbeGas
}

// sampleGas is the sampler call's gas: SampleGas or defaultSampleGas, and at least minSampleGas.
func (c *Config) sampleGas() uint64 {
	if c.SampleGas == 0 {
		return defaultSampleGas
	}
	return max(c.SampleGas, minSampleGas)
}

func (c *Config) haircutBps() uint64 {
	if c.HaircutBps == nil {
		return defaultHaircutBps
	}
	return min(*c.HaircutBps, 10_000)
}

func (c *Config) rungs() int {
	if c.Rungs == 0 {
		return defaultRungs
	}
	return min(max(c.Rungs, minRungs), maxRungs)
}

func (c *Config) hasFactory(f string) bool {
	return slices.ContainsFunc(c.Factories, func(g string) bool { return strings.EqualFold(f, g) })
}

func (c *Config) hasFeed(f common.Address) bool {
	return slices.ContainsFunc(c.PriceFeeds, func(g string) bool { return common.HexToAddress(g) == f })
}

func orDefault[T any](v, fallback []T) []T {
	if len(v) == 0 {
		return fallback
	}
	return v
}

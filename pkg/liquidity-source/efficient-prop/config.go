package efficientprop

import (
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config names the PropAMM fronts listed on one chain. Each front holds any number of listings, and each listing is
// reached through its own door, which the front's door factory creates at the CREATE2 address of the listing id.
// Nothing here is a default: every address comes from the consumer's per-dex properties.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`

	// Fronts are the fronts listed, each with its listing-door factory.
	Fronts []Front `json:"fronts"`

	// ApprovedDesks are the desk contracts whose listings are listed and quoted, the maker gate. A listing's maker is
	// its desk, and anyone may open a desk and a listing on a front, so a listing is quoted only where its desk's
	// contract (the front's deskCellOf of its doorView's desk) is named here. EMPTY BY DEFAULT, AND EMPTY MEANS NOTHING
	// IS LISTED OR QUOTED (unless ApprovedDesksURL's list names a desk).
	ApprovedDesks []string `json:"approvedDesks,omitempty"`
	// ApprovedDesksURL, where set, is an https url answering the approved desks of this chain as
	// {"chainId":1,"updatedAt":<unix s>,"desks":["0x…"]}. Its list is fetched every 15 s at most and rules while its
	// last fetch taken is at most ApprovedDesksMaxAge old, so a desk it drops is quoted no more within one fetch. Once
	// it is older, ApprovedDesks rules, less the desks that list had left out: a failing url never brings a desk in.
	ApprovedDesksURL string `json:"approvedDesksURL,omitempty"`
	// ApprovedDesksMaxAge is how long a fetched list rules, seconds; zero: 600.
	ApprovedDesksMaxAge uint64 `json:"approvedDesksMaxAge,omitempty"`

	// MaxListings caps how many listing ids the lister reads from one front in one poll; zero: defaultMaxListings.
	MaxListings int `json:"maxListings,omitempty"`

	// MinTipCapWei is the least nonzero taker tip cap, wei per gas above the base fee, of a listing listed and quoted:
	// a fill sent at a higher tip than its listing's cap reverts, and nothing here sees the taker's tip. A listing whose
	// cap is 0 passes (on a storage build it fills nothing, and quotes no size; on any other it has no cap). Zero: the
	// chain's default (defaultMinTipCapWei: 2 gwei on Ethereum, none on any other chain); 1 lists every listing.
	MinTipCapWei uint64 `json:"minTipCapWei,omitempty"`

	// InclusionAllowance is the time a routed fill takes to land, seconds: route finding quotes a listing only where a
	// fill landing that much later still finds its word standing. Zero: 1 s on a chain whose blocks are under a second
	// apart (Base), 3 s on any other (README, Config); on a front read through /state (Front.Titan), 1 s before its
	// target block's second.
	InclusionAllowance uint64 `json:"inclusionAllowance,omitempty"`
}

// Front is one PropAMM front (its proxy) and the factory of its listing doors. The config names no door code hash: the
// source reads DOOR_CODEHASH() from the deployed factory, and derives no door from anything else. On a builder chain
// (Ethereum), whose fronts are read by the per-desk coverage rule, it also names the front's Layout and CellCodehash,
// its CELL_CODEHASH() fixed when the front is created: the front must answer it, else it is refused, and the desk
// contracts are the CREATE2 children of the front under it. On any other chain, which is read at the head, both are
// unset and refused if set.
type Front struct {
	Address      string `json:"address"`
	DoorFactory  string `json:"doorFactory"`
	CellCodehash string `json:"cellCodehash"`
	// Layout names the layout of the front's quote contract, the one this source reads. Required on a builder chain
	// (Ethereum), where the only layout is LayoutBuilder: a front with another value, or none, is refused at load. Unset
	// on any other chain.
	Layout string `json:"layout"`
	// FillMode is the entry the adapter fills this front's listings through, FillModeFront or FillModeDoor. It is
	// required: neither is a default. Its pools carry it (StaticExtra.Mode) from their listing on.
	FillMode string `json:"fillMode"`
	// ApprovedSources are the price sources a listing of this front may read and still be quoted: a PRINT or SESSION
	// listing is quoted only where every source it is bound to is one of them (a tied listing is never quoted). A
	// source is any contract anyone registers with the front, and its code runs in the quote and in the fill; approve
	// one only by the README's rule (How to approve a source). None by default: then only the listings that read no
	// source (WORD and NONE) are quoted, until an operator of this integration names the protocol's own sources (a
	// storage build's stored price lanes, each the address in its entry).
	ApprovedSources []string `json:"approvedSources,omitempty"`
	// Implementations are the front implementations its proxy may run while its listings are quoted, each read from
	// the proxy's ERC-1967 slot at the refresh block. The source reads the front's storage by its layout (a listing's
	// controls, sources and session words, the source entries) and mirrors its clock rule and the way its prices age,
	// so a front upgraded to an implementation not listed is quoted no more until an operator of this integration lists
	// it. Required: at least one, with no default.
	Implementations []string `json:"implementations"`
	// BaseTime is the chain's millisecond-time contract, whose timestampMs() the front's block-count clock (clock mode
	// 2) reads in a block its anchor does not place in that block's own second (the front's own constant, on Base the
	// OP Stack predeploy 0x4200000000000000000000000000000000000030). The refresh reads it beside the front's clock and
	// sets its stored sub-second by a state override to place a sample. Required where the front's clock is the
	// block-count mode: such a front's refresh fails while it is unset. A front on another clock does not read it.
	BaseTime string `json:"baseTime,omitempty"`
	// Titan is the /state read of a front on a builder chain (Ethereum), where a maker's update lands only in a block a
	// builder built with it ahead of a fill, so the chain's head never holds a quotable word: the urls answering
	// titan_getPammStateOverrides (the api's /v1/<chain>/state and the builders' data hosts), in the order a listing's
	// answer is chosen by (of the highest block's, the first holding its entries), and the read's timeout. Every
	// refresh samples the door under the overrides and the target block of that answer. Required on a builder chain,
	// refused on any other.
	Titan *titan.Config `json:"titan,omitempty"`
	// CreditListings are the listings with a credit mode this front's listings quote, each with its fills' gas. A
	// listing with a credit mode fills inside its pool's custody where that fills it whole, and through its pool's own
	// swap past it (its credit route, which may borrow), so its fill's output and gas depend on the size. Its ladder
	// marks each size the front routes (README, Credit-mode listings). The gas of a route belongs to its pool (its
	// venue, its tokens, its caps), so no figure of this source's own can stand for it: only a listing named here, with
	// figures measured on its pool, is quoted. Only on a builder chain (a front read through /state), where each state
	// is sampled in the blocks it can fill in alone; refused on any other. None by default.
	CreditListings []CreditListing `json:"creditListings,omitempty"`
	// ListingGas are listings charged their own measured gas in place of their class's figure (README, Execution and
	// gas). A class's figure is the least of the listings it was measured on, and two listings of one class may differ
	// by a few thousand gas (their tokens' own transfer costs), so the dearer one is under-charged by the class: name
	// it here with its own figures. On a builder chain in the front's mode, where only some classes have a measured
	// figure, a listing of a class with none is quoted only where it is named here. A listing with a credit mode takes
	// its figures from CreditListings, never from here. None by default.
	ListingGas []ListingGas `json:"listingGas,omitempty"`
}

// ListingGas is one listing's own gas, with its base token in and with its quote token in: each the least first fill
// of that input seen on the listing through the adapter in the front's configured fill mode, the adapter call frame's
// gas less the refund the fill earns, as every figure of this source is stated (README, Execution and gas).
type ListingGas struct {
	Listing uint32 `json:"listing"`
	BaseIn  int64  `json:"baseIn"`
	QuoteIn int64  `json:"quoteIn"`
}

// listingGas is the front's configured gas of listing, nil where it names none.
func (f Front) listingGas(listing uint32) *ListingGas {
	for i := range f.ListingGas {
		if f.ListingGas[i].Listing == listing {
			return &f.ListingGas[i]
		}
	}
	return nil
}

// CreditListing is a listing with a credit mode its front quotes, and its fills' gas: each the least first fill of its
// class and input seen on the listing's own pool through the adapter in the front's configured fill mode, the adapter
// call frame's gas less the refund the fill earns, as every figure of this source (README, Execution and gas). A
// custody fill is one the front fills inside the pool's custody (through its named entry in either mode, since the
// front names no paying pool for a listing with a credit mode); a route fill one the front sends to the pool's own
// swap. Base in and quote in are the pair's base and quote token as the input. A route figure of 0 says that side's
// route was never measured on the pool: its routed sizes are then not quoted, and its sizes inside the custody are (a
// pool whose credit mode routes one side alone, or whose route no fill has taken yet). The custody figures are
// required.  A route figure is the gas a route is charged, so it is the least route fill of its side: where the pair
// may borrow at more than one lending venue, the cheaper venue's. The front's plan tells a route from a custody fill
// and nothing of the venue the pool will draw at, so this source cannot charge each venue its own. A route's gas limit
// is another matter, and must cover the dearest: RouteLimitBaseIn and RouteLimitQuoteIn, where set, are the least gas
// limit that let the dearest route fill of the side through (the dearer venue's, with its margin), which a routed quote
// carries beside its gas (SwapInfo.GasLimit). Each is 0 (none stated) or at least its side's route figure.
type CreditListing struct {
	Listing           uint32 `json:"listing"`
	CustodyBaseIn     int64  `json:"custodyBaseIn"`
	CustodyQuoteIn    int64  `json:"custodyQuoteIn"`
	RouteBaseIn       int64  `json:"routeBaseIn"`
	RouteQuoteIn      int64  `json:"routeQuoteIn"`
	RouteLimitBaseIn  int64  `json:"routeLimitBaseIn,omitempty"`
	RouteLimitQuoteIn int64  `json:"routeLimitQuoteIn,omitempty"`
}

// maxCreditGas bounds a configured gas figure, a credit listing's or a listing's own: a route fill measured on
// Ethereum costs about 950,000 to 990,000 on a pool that borrows from one lending venue, and a figure far past that
// is a typo.
const maxCreditGas = 3_000_000

// maxCreditGasLimit bounds a configured route gas limit: a limit runs above the gas it covers.
const maxCreditGasLimit = 2 * maxCreditGas

// creditListing is the front's configured credit listing of id listing, nil where it names none.
func (f Front) creditListing(listing uint32) *CreditListing {
	for i := range f.CreditListings {
		if f.CreditListings[i].Listing == listing {
			return &f.CreditListings[i]
		}
	}
	return nil
}

// builder is whether the chain is read through /state: a builder chain's fronts keep their quote contract by the
// per-desk rule, which needs their layout and desk code hash.
func (c *Config) builder() bool { return builderChains[c.ChainID] }

// builder is whether the front is read through /state, as a front on a builder chain is.
func (f Front) builder() bool {
	return f.Titan != nil
}

// approves is whether src is one of the front's approved price sources.
func (f Front) approves(src common.Address) bool {
	if src == (common.Address{}) {
		return false
	}
	for _, a := range f.ApprovedSources {
		if common.HexToAddress(a) == src {
			return true
		}
	}
	return false
}

// implements is whether impl is one of the front's listed implementations.
func (f Front) implements(impl common.Address) bool {
	if impl == (common.Address{}) {
		return false
	}
	for _, a := range f.Implementations {
		if common.HexToAddress(a) == impl {
			return true
		}
	}
	return false
}

// LayoutBuilder is the quote contract layout of the builder code, a desk contract with a pair slot for each listing
// (Ethereum, the one builder chain): the only layout this source reads.
const LayoutBuilder = "builder"

// The fill modes a front may be configured with, and the adapter's mode word for each (its data's fourth word).
//   - FillModeFront: the front's own entries. Where the front names the listing's pool (payPoolOf), the adapter pays
//     the input to that pool and calls the front's paid entry; else it approves the front and calls its named entry.
//   - FillModeDoor: the listing door's pull entry. The adapter approves the door. The fill's Swapped is logged once in
//     either mode: by the door on a storage build, by the listing's quote contract on a builder build.
const (
	FillModeFront = "front"
	FillModeDoor  = "door"

	modeDoor  uint8 = 0
	modeFront uint8 = 1
)

// mode is the adapter's mode word for the front's fill mode; ok is false where none is set or it is another.
func (f Front) mode() (m uint8, ok bool) {
	switch f.FillMode {
	case FillModeFront:
		return modeFront, true
	case FillModeDoor:
		return modeDoor, true
	}
	return 0, false
}

// validate checks that the chain and every address are set and well formed, and lowercases the addresses.
func (c *Config) validate() error {
	if c.ChainID == 0 {
		return fmt.Errorf("%w: chainID is not set", ErrInvalidConfig)
	}
	if unreadChains[c.ChainID] {
		return fmt.Errorf("%w: chain %d's front is a builder build, and this source has no next-block read for it",
			ErrInvalidConfig, c.ChainID)
	}
	if len(c.Fronts) == 0 {
		return fmt.Errorf("%w: no fronts", ErrInvalidConfig)
	}
	if c.MaxListings < 0 {
		return fmt.Errorf("%w: maxListings=%d", ErrInvalidConfig, c.MaxListings)
	}
	for k, d := range c.ApprovedDesks {
		if !common.IsHexAddress(d) || common.HexToAddress(d) == (common.Address{}) {
			return fmt.Errorf("%w: approvedDesks[%d]=%q", ErrInvalidConfig, k, d)
		}
		c.ApprovedDesks[k] = strings.ToLower(d)
	}
	if c.ApprovedDesksURL != "" {
		if u, err := neturl.Parse(c.ApprovedDesksURL); err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("%w: approvedDesksURL is not an https url", ErrInvalidConfig)
		}
	}
	seen := make(map[string]struct{}, len(c.Fronts))
	for i := range c.Fronts {
		f := &c.Fronts[i]
		for name, addr := range map[string]*string{"address": &f.Address, "doorFactory": &f.DoorFactory} {
			if !common.IsHexAddress(*addr) || common.HexToAddress(*addr) == (common.Address{}) {
				return fmt.Errorf("%w: fronts[%d].%s=%q", ErrInvalidConfig, i, name, *addr)
			}
			*addr = strings.ToLower(*addr)
		}
		if builderChains[c.ChainID] {
			// the next-block path reads a desk contract by the per-desk rule: its layout and its code hash are named
			if f.Layout != LayoutBuilder {
				return fmt.Errorf("%w: fronts[%d].layout=%q, want %q: no other quote contract layout is read",
					ErrInvalidConfig, i, f.Layout, LayoutBuilder)
			}
			if h, err := hexutil.Decode(f.CellCodehash); err != nil || len(h) != common.HashLength ||
				common.BytesToHash(h) == (common.Hash{}) {
				return fmt.Errorf("%w: fronts[%d].cellCodehash=%q", ErrInvalidConfig, i, f.CellCodehash)
			}
			f.CellCodehash = strings.ToLower(f.CellCodehash)
		} else if f.Layout != "" || f.CellCodehash != "" {
			return fmt.Errorf("%w: fronts[%d].layout=%q and cellCodehash=%q are set on chain %d, which is read at the "+
				"head: no quote contract is read there, so neither applies", ErrInvalidConfig, i, f.Layout,
				f.CellCodehash, c.ChainID)
		}
		for k, src := range f.ApprovedSources {
			if !common.IsHexAddress(src) || common.HexToAddress(src) == (common.Address{}) {
				return fmt.Errorf("%w: fronts[%d].approvedSources[%d]=%q", ErrInvalidConfig, i, k, src)
			}
			f.ApprovedSources[k] = strings.ToLower(src)
		}
		if len(f.Implementations) == 0 {
			return fmt.Errorf("%w: fronts[%d].implementations is empty", ErrInvalidConfig, i)
		}
		for k, impl := range f.Implementations {
			if !common.IsHexAddress(impl) || common.HexToAddress(impl) == (common.Address{}) {
				return fmt.Errorf("%w: fronts[%d].implementations[%d]=%q", ErrInvalidConfig, i, k, impl)
			}
			f.Implementations[k] = strings.ToLower(impl)
		}
		if f.BaseTime != "" {
			if !common.IsHexAddress(f.BaseTime) || common.HexToAddress(f.BaseTime) == (common.Address{}) {
				return fmt.Errorf("%w: fronts[%d].baseTime=%q", ErrInvalidConfig, i, f.BaseTime)
			}
			f.BaseTime = strings.ToLower(f.BaseTime)
		}
		if _, ok := f.mode(); !ok {
			return fmt.Errorf("%w: fronts[%d].fillMode=%q, want %q or %q", ErrInvalidConfig, i, f.FillMode,
				FillModeFront, FillModeDoor)
		}
		if err := c.validateBuilder(i); err != nil {
			return err
		}
		if _, ok := seen[f.Address]; ok {
			return fmt.Errorf("%w: front %s listed twice", ErrInvalidConfig, f.Address)
		}
		seen[f.Address] = struct{}{}
	}
	return nil
}

// validateBuilder checks front i's /state read against its chain: on a builder chain a titan block with at least one
// url, in either fill mode (the fill's Swapped, which a builder matches to place the desk's update, is the quote
// contract's there in either mode, and the door's entries are the front's own, called with the listing); on any other
// chain none, since the read takes its block's time from the beacon chain's slot. On every chain, the front's listing
// gas: each a listing id named once and by no credit listing, both figures measured.
func (c *Config) validateBuilder(i int) error {
	f := &c.Fronts[i]
	named := make(map[uint32]bool, len(f.ListingGas))
	for k, g := range f.ListingGas {
		if g.Listing == 0 || g.Listing >= 1<<24 || named[g.Listing] || f.creditListing(g.Listing) != nil {
			return fmt.Errorf("%w: fronts[%d].listingGas[%d].listing=%d: not a listing id, named twice, or a credit "+
				"listing, whose figures are its creditListings entry's", ErrInvalidConfig, i, k, g.Listing)
		}
		named[g.Listing] = true
		for name, v := range map[string]int64{"baseIn": g.BaseIn, "quoteIn": g.QuoteIn} {
			if v <= 0 || v > maxCreditGas {
				return fmt.Errorf("%w: fronts[%d].listingGas[%d].%s=%d, want a measured figure in (0, %d]",
					ErrInvalidConfig, i, k, name, v, maxCreditGas)
			}
		}
	}
	if !builderChains[c.ChainID] {
		if f.Titan != nil {
			return fmt.Errorf("%w: fronts[%d].titan on chain %d, which no builder orders", ErrInvalidConfig, i,
				c.ChainID)
		}
		if len(f.CreditListings) > 0 {
			return fmt.Errorf("%w: fronts[%d].creditListings on chain %d, which is read at the head: a credit "+
				"listing is quoted only where each state is sampled in the blocks it can fill in", ErrInvalidConfig, i,
				c.ChainID)
		}
		return nil
	}
	seen := make(map[uint32]bool, len(f.CreditListings))
	for k, cl := range f.CreditListings {
		if cl.Listing == 0 || cl.Listing >= 1<<24 || seen[cl.Listing] {
			return fmt.Errorf("%w: fronts[%d].creditListings[%d].listing=%d: not a listing id, or named twice",
				ErrInvalidConfig, i, k, cl.Listing)
		}
		seen[cl.Listing] = true
		for name, g := range map[string]int64{"custodyBaseIn": cl.CustodyBaseIn, "custodyQuoteIn": cl.CustodyQuoteIn} {
			if g <= 0 || g > maxCreditGas {
				return fmt.Errorf("%w: fronts[%d].creditListings[%d].%s=%d, want a measured figure in (0, %d]",
					ErrInvalidConfig, i, k, name, g, maxCreditGas)
			}
		}
		// a route figure of 0 is a side whose route was never measured: its routed sizes are not quoted
		for name, g := range map[string]int64{"routeBaseIn": cl.RouteBaseIn, "routeQuoteIn": cl.RouteQuoteIn} {
			if g < 0 || g > maxCreditGas {
				return fmt.Errorf("%w: fronts[%d].creditListings[%d].%s=%d, want a measured figure in (0, %d], or 0 "+
					"for a route never measured", ErrInvalidConfig, i, k, name, g, maxCreditGas)
			}
		}
		// a route's gas limit covers its dearest fill: never under the side's charged figure, none without one
		for name, g := range map[string][2]int64{"routeLimitBaseIn": {cl.RouteLimitBaseIn, cl.RouteBaseIn},
			"routeLimitQuoteIn": {cl.RouteLimitQuoteIn, cl.RouteQuoteIn}} {
			if g[0] != 0 && (g[1] == 0 || g[0] < g[1] || g[0] > maxCreditGasLimit) {
				return fmt.Errorf("%w: fronts[%d].creditListings[%d].%s=%d, want 0 or a limit in [its route figure %d, "+
					"%d] on a side with a route figure", ErrInvalidConfig, i, k, name, g[0], g[1], maxCreditGasLimit)
			}
		}
	}
	if f.Titan == nil || len(f.Titan.URLs) == 0 {
		return fmt.Errorf("%w: fronts[%d].titan.urls is empty on chain %d", ErrInvalidConfig, i, c.ChainID)
	}
	if f.Titan.Timeout < 0 || f.Titan.Timeout > maxStateTimeout {
		return fmt.Errorf("%w: fronts[%d].titan.timeout=%s, want 0 (%s) or at most %s", ErrInvalidConfig, i,
			f.Titan.Timeout, defaultStateTimeout, maxStateTimeout)
	}
	for k, u := range f.Titan.URLs {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("%w: fronts[%d].titan.urls[%d] is empty", ErrInvalidConfig, i, k)
		}
	}
	return nil
}

// front is the configured front at address, nil where none is.
func (c *Config) front(address string) *Front {
	for i := range c.Fronts {
		if strings.EqualFold(c.Fronts[i].Address, address) {
			return &c.Fronts[i]
		}
	}
	return nil
}

func (c *Config) maxListings() int {
	if c.MaxListings == 0 {
		return defaultMaxListings
	}
	return c.MaxListings
}

// minTipCapWei is the tip cap floor: the configured one, else the chain's default.
func (c *Config) minTipCapWei() uint64 {
	if c.MinTipCapWei != 0 {
		return c.MinTipCapWei
	}
	return defaultMinTipCapWei[c.ChainID]
}

// tipCapOK reports whether a listing's taker tip cap (wei) is 0 or at the floor or above it.
func (c *Config) tipCapOK(capWei uint64) bool {
	return capWei == 0 || capWei >= c.minTipCapWei()
}

func (c *Config) inclusionAllowance() uint64 {
	switch {
	case c.InclusionAllowance != 0:
		return c.InclusionAllowance
	case subSecondChains[c.ChainID]:
		return subSecondInclusionAllowance
	}
	return defaultInclusionAllowance
}

// builderAllowance is the inclusion allowance of a front read through /state: the configured one, else
// builderInclusionAllowance.
func (c *Config) builderAllowance() uint64 {
	if c.InclusionAllowance != 0 {
		return c.InclusionAllowance
	}
	return builderInclusionAllowance
}

func (c *Config) dexID() string {
	if c.DexID == "" {
		return DexType
	}
	return c.DexID
}

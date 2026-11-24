package efficientprop

import "github.com/holiman/uint256"

// StaticExtra is a listing door's immutable routing data. The pool's address is the door itself.
type StaticExtra struct {
	Front   string `json:"front"`
	Listing uint32 `json:"listing"`
	// Custody is the listing's pool: it holds the tokens a fill pays out, shared by every listing of that pool.
	Custody string `json:"custody"`
	// Door is the listing's door, the pool's address.
	Door string `json:"door"`
	// Mode is the adapter's fill mode, from the front's config: 0 the door, 1 the front (Front.FillMode).
	Mode uint8 `json:"mode"`
}

// Point is one sampled quote: door.quote(tokenIn, tokenOut, Point[0]) answered Point[1] at the refresh block.
type Point [2]*uint256.Int

// Extra is a listing's state at the refresh block: one ladder a direction (0: token0 in, 1: token1 in), strictly
// ascending in input with the output never falling, and the listing's settings a fill reads. A listing with a credit
// mode keeps its ladders in Credit and none here (ladders).
type Extra struct {
	Ladders [2][]Point `json:"l"`
	// TipCapWei is the listing's taker tip cap, wei per gas above the base fee: a fill sent at a higher tip
	// reverts. Zero on a storage build: the listing fills nothing; on any other: no cap.
	TipCapWei uint64 `json:"tipCapWei,omitempty"`
	// CreditMode is the listing's credit mode. A listing with one is quoted only where its front's config names it
	// (Front.CreditListings), on a front read through /state (see the tracker); Credit is then set.
	CreditMode uint8 `json:"creditMode,omitempty"`
	// Credit, on a listing with a credit mode that is quoted, is its ladders, where each steps from the pool's custody
	// to its credit route, and each class's gas. Nil on any other listing.
	Credit *Credit `json:"credit,omitempty"`
	// ValidUntil is the last unix second a fill quoted from the state may land: the end of the window its ladders were
	// sampled over, never after the last second in which a fill certainly finds the sampled word standing and the
	// sampled sides' depth unexpired (validUntil); zero where the listing had no ladder. On a front read through /state
	// (Builder) it is the last second a route quoted from the state may be built: its target block's second, or the
	// next block's where it carries, less the inclusion allowance.
	ValidUntil uint64 `json:"validUntil,omitempty"`
	// Allowance is the time a routed fill takes to land, s (Config.InclusionAllowance): route finding quotes a fill
	// only where it lands by ValidUntil.
	Allowance uint64 `json:"allowance,omitempty"`
	// Idle is whether the listing is not quoted for a reason no new block lifts (its desk not approved, its fill mode,
	// its tip cap, its class or credit mode with no gas figure, not live): its refresh is asked for less often
	// (NextUpdateAt).
	Idle bool `json:"idle,omitempty"`
	// BaseToken1 is whether the pair's base is token 1 (the pair word's bit 8 clear), which sets each direction's gas.
	BaseToken1 bool `json:"base1,omitempty"`
	// Print is whether the listing's fill reads a price source or an asset module, which sets each direction's gas: a
	// PRINT or SESSION listing, or any listing of an asset-module pair.
	Print bool `json:"print,omitempty"`
	// Guarded is whether the source the listing reads now is guarded (its fill asks the source's lane for the side's
	// worst held print), which sets each direction's gas: the guard of its live source word (record word 2), or where
	// that word is not read, of either entry it is bound to.
	Guarded bool `json:"guarded,omitempty"`
	// Past is whether the listing's pair is past its pool's hook pair (doorView's base index not 0 or quote index not
	// 1), which sets each direction's gas.
	Past bool `json:"past,omitempty"`
	// NoAsset is whether neither of the listing's tokens is its pool's asset (doorView's base and quote index both not
	// 0), which sets each direction's gas on a front read through /state (Builder).
	NoAsset bool `json:"noAsset,omitempty"`
	// AssetQuote is whether the listing's quote token is its pool's asset (doorView's quote index 0), which picks each
	// direction's gas past the hook pair on a front read through /state (Builder): the side whose input is the asset.
	AssetQuote bool `json:"assetQuote,omitempty"`
	// Builder is whether the listing's front is read through /state (Config's Front.Titan, a builder chain's): its
	// ladders hold for the answer's target block, and the next where it carries; its ValidUntil is that block's second
	// less the inclusion allowance, so its Allowance is 0; and it is refreshed on the wall clock (NextUpdateAt).
	Builder bool `json:"builder,omitempty"`
	// Gas, where set, is the listing's own gas with its base token in ([0]) and with its quote token in ([1]), from
	// its front's config (Front.ListingGas): it stands in place of the class's figure the flags above pick.
	Gas *[2]int64 `json:"gas,omitempty"`
	// NoneHeld's bit i is set where the listing's pool held none of token i at the state's block (its balance read,
	// and zero): a fill with that token in writes the pool's balance of it from zero, and is charged gasFirstOfInput
	// more. Reserves does not say it: a side with no ladder zeroes its output's reserve.
	NoneHeld uint8 `json:"noneHeld,omitempty"`
}

// Credit is a credit-mode listing's ladders, their classes and gas. A ladder's points from From[dir] on are sizes the
// front fills through the pool's own swap (its credit route), those before it sizes it fills inside the pool's
// custody: the front's plan said so at every sample (the tip-taking quoteFrom's ROUTED tag), and no custody point
// follows a routed one. An amount strictly between the last custody point and the first routed one is not quoted. The
// ladders are kept here, not in Extra.Ladders, so a simulator that predates credit listings, which would draw a chord
// across the step and charge a custody fill's gas for a route, finds no ladder and quotes nothing.
type Credit struct {
	Ladders [2][]Point `json:"l"`
	// From is, per direction, the index of its ladder's first routed point; the ladder's length where none routes.
	From [2]int `json:"from"`
	// Gas is, per direction, a fill's gas inside the custody ([0]) and on the credit route ([1]), from the front's
	// config (CreditListing) by the direction's input, base or quote. A route figure is 0 only on a direction with no
	// routed point (a side whose route was never measured is not quoted past its custody).
	Gas [2][2]int64 `json:"gas"`
	// Limit is, per direction, the gas limit its config states for a route fill (CreditListing.RouteLimitBaseIn and
	// RouteLimitQuoteIn), 0 where it states none; nil where it states none for either.
	Limit *[2]int64 `json:"limit,omitempty"`
}

// ladders is the listing's ladders: Credit's on a listing with a credit mode, else Ladders.
func (e *Extra) ladders() [2][]Point {
	if e.Credit != nil {
		return e.Credit.Ladders
	}
	return e.Ladders
}

// PoolMeta is what the encoder needs: the executor data is abi.encode(door, front, listing, mode). In the door's mode
// (0) the adapter approves the door, which pulls the input (swapFrom). In the front's (1) it pays the input to the
// listing's pool and calls the front's paid entry, or where the front names no pool, approves the front and calls
// its named entry. ApprovalAddress is the one contract the adapter may approve: the door, or the front.
type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	Door            string `json:"door"`
	Front           string `json:"front"`
	Listing         uint32 `json:"listing"`
	Mode            uint8  `json:"mode"`
	ApprovalAddress string `json:"approvalAddress"`
	TipCapWei       uint64 `json:"tipCapWei,omitempty"`
}

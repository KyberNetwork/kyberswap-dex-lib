package thogprop

import "github.com/holiman/uint256"

// Layout of the pool state that getPoolData returns: storage words 2..127, each
// packing fixed-width fields. Offsets count bits from the least significant bit.
// Names follow the contract's storage constants. Every field is specified in
// https://github.com/thogard785/thog-amm-integration-guide/blob/main/REFERENCE.md

const (
	schemaVersion = 8   // getPoolData's schemaVersion for this layout
	firstPageSlot = 2   // getPoolData returns words firstPageSlot..127
	pageWords     = 126 // in slot order
)

// bitField locates one field inside one storage word.
type bitField struct{ slot, offset, width int }

// ref is how getPoolData's metadata names a field: slot<<16 | offset<<8 | width.
func (f bitField) ref() uint32 { return uint32(f.slot<<16 | f.offset<<8 | f.width) }

// packed locates item index of an array that stores perWord fields of width bits
// in each consecutive word from slot. Every caller passes constants, so the
// divisions are resolved at compile time.
func packed(slot, width, perWord, index int) bitField {
	return bitField{slot + index/perWord, index % perWord * width, width}
}

// recordField locates a field inside one token or category record.
type recordField struct{ offset, width int }

func (r recordField) of(record bitField) bitField {
	return bitField{record.slot, record.offset + r.offset, r.width}
}

// Control word (slot 2): pool-wide switches and the parameters every quote uses.
var (
	controlLayoutVersion = bitField{2, 0, 8}    // state layout; this adapter reads version 8
	controlTokenCount    = bitField{2, 8, 6}    // listed tokens
	controlCategoryCount = bitField{2, 14, 6}   // categories: groups of tokens that track one asset
	controlSequence      = bitField{2, 20, 16}  // counter of price updates
	controlBlock         = bitField{2, 36, 32}  // block of the latest price update
	controlPaused        = bitField{2, 132, 1}  // swaps are switched off
	controlReady         = bitField{2, 133, 3}  // prices enabled, risk parameters set, wrapper parameters set
	controlMaxAge        = bitField{2, 138, 8}  // blocks a price update stays quotable
	controlWiden         = bitField{2, 146, 16} // spread added per block of price age, in 0.01 bp
	controlLambda        = bitField{2, 162, 32} // weight of the portfolio risk charge, scaled by 1e6
)

const (
	supportedLayout = 8 // the only controlLayoutVersion this adapter decodes
	controlAllReady = 7 // controlReady with all three bits set
)

// Two bits per token follow from bit 68 of the control word: the pool accepts the
// token as swap input, then the pool pays it as swap output.
const controlSidesOffset = 68

func controlSides(token int) bitField    { return bitField{2, controlSidesOffset + 2*token, 2} }
func controlSellSide(token int) bitField { return bitField{2, controlSidesOffset + 2*token, 1} }
func controlBuySide(token int) bitField  { return bitField{2, controlSidesOffset + 2*token + 1, 1} }

// Metadata (slot 8): which state format the contract writes.
var (
	metadataVersions   = bitField{8, 160, 24} // layout, encoding and pricing model, one byte each
	metadataGeneration = bitField{8, 184, 32} // counts token listings and removals; never zero
)

// supportedVersions is layout 8, encoding 8, pricing model 3.
const supportedVersions = 8 | 8<<8 | 3<<16

// Per-token and per-category arrays, indexed by the token's or category's
// position in the pool.

// tokenRecord is one record per token; its fields are listed below.
func tokenRecord(token int) bitField { return packed(9, 75, 3, token) }

// registryIndex maps the contract's fixed token ID to its position in the pool plus one.
func registryIndex(catalogID int) bitField { return packed(19, 6, 32, catalogID) }

// priceField is a token's USD price: a 56-bit mantissa and an 8-bit decimal exponent.
func priceField(token int) bitField { return packed(20, 64, 4, token) }

// categoryRecord is one record per category; its fields are listed below.
func categoryRecord(category int) bitField { return packed(27, 85, 3, category) }

// covarianceField is signed and scaled by 1e9; categories a <= b are at covarianceIndex(a, b).
func covarianceField(index int) bitField { return packed(37, 16, 16, index) }

// pairSpreadField is the extra spread of a token pair, at pairIndex(a, b), in 0.01 bp.
func pairSpreadField(index int) bitField { return packed(63, 16, 16, index) }

// wrapperCoefficient is the rate of a token's per-token cost, scaled by 1e18.
func wrapperCoefficient(token int) bitField { return packed(87, 64, 4, token) }

// categoryTarget is the holding the per-token cost is scaled to, in millionths of a USD.
func categoryTarget(category int) bitField { return packed(94, 64, 4, category) }

// tokenCap is the most of a token the pool will hold, in raw units (16-bit mantissa, 8-bit shift).
func tokenCap(token int) bitField { return packed(101, 24, 10, token) }

// categoryCap is the most of a category the pool will hold, in USD scaled by 1e18 (same encoding).
func categoryCap(category int) bitField { return packed(104, 24, 10, category) }

// inventoryField is the pool's recorded holding: shares if the token is share-based, else raw units.
func inventoryField(token int) bitField { return packed(107, 128, 2, token) }

// Fields of a token record.
var (
	tokenCatalogID     = recordField{0, 5}   // the contract's fixed ID for this token
	tokenDecimals      = recordField{5, 5}   // ERC-20 decimals
	tokenCategory      = recordField{10, 5}  // category the token belongs to
	tokenBidSpread     = recordField{15, 16} // spread when the pool receives the token, in 0.01 bp
	tokenWrapperKind   = recordField{31, 2}  // per-token cost: 0 none, 1 fixed rate, 2 grows with holdings
	tokenWrapperRefund = recordField{33, 10} // share of that cost credited back on output, in 0.1%
	tokenAskSpread     = recordField{43, 16} // spread when the pool pays the token, in 0.01 bp
	tokenVolumeRate    = recordField{59, 16} // charge that grows with trade size, 0.01 bp per $1,000 of input
)

// Values of tokenWrapperKind.
const (
	wrapperNone  = 0
	wrapperFixed = 1 // any other value: the cost grows with the pool's holding of the token
)

// minPriceMantissa is the smallest mantissa of a valid price; a smaller one
// means the token has no usable price.
const minPriceMantissa = 2_000_000

// Fields of a category record.
var (
	categoryOffset = recordField{0, 64}  // signed holding outside the pool less its target, in millionths of the anchor token
	categorySpread = recordField{64, 16} // spread for every token of the category, in 0.01 bp
	categoryAnchor = recordField{80, 5}  // token whose price values the offset
)

func tokenField(token int, f recordField) bitField { return f.of(tokenRecord(token)) }

func categoryField(category int, f recordField) bitField { return f.of(categoryRecord(category)) }

// pairIndex and covarianceIndex locate an unordered pair in its triangular array.
func pairIndex(a, b int) int {
	if a > b {
		a, b = b, a
	}
	return b*(b-1)/2 + a
}

func covarianceIndex(a, b int) int {
	if a > b {
		a, b = b, a
	}
	return b*(b+1)/2 + a
}

func (s *State) field(f bitField) uint256.Int {
	return bits(&s.Words[f.slot], uint(f.offset), uint(f.width))
}

// small reads a field of at most 64 bits.
func (s *State) small(f bitField) uint64 {
	return bits(&s.Words[f.slot], uint(f.offset), uint(f.width))[0]
}

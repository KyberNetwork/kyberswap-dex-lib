package nightlyprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
)

type StaticExtra struct {
	RouterAddress  string `json:"routerAddress"`
	PartnerAddress string `json:"partnerAddress"`
}

// Extra is the pool's state that the router's getPoolStateV1 returned for the record's partner at
// the record's BlockNumber. The record has zero reserves unless swapError finds no reason that the
// partner's swaps revert. Marshal it through a pointer, as math.Pool requires.
type Extra struct {
	Routed       bool `json:"routed"`
	RouterPaused bool `json:"routerPaused"`
	PoolPaused   bool `json:"poolPaused"`
	// RouterSwapper is whether the pool grants the router SWAPPER_ROLE.
	RouterSwapper bool `json:"routerSwapper"`
	// PoolPriceReader is whether the oracle grants the pool PRICE_READER_ROLE.
	PoolPriceReader bool `json:"poolPriceReader"`
	// PartnerRole is whether the partner holds the router's PARTNER_ROLE.
	PartnerRole bool `json:"partnerRole"`
	// PartnerFeeRecipientSet is whether the router has a fee recipient for the partner.
	PartnerFeeRecipientSet bool   `json:"partnerFeeRecipientSet"`
	PriceAvailable         bool   `json:"priceAvailable"`
	BlockTimestamp         uint64 `json:"blockTimestamp"`
	// Price is zero when PriceAvailable is false. A stale price stays, with its time.
	Price math.OraclePrice `json:"price"`
	Pool  math.Pool        `json:"pool"`
}

// poolsListUpdaterMetadata is the cursor of GetNewPools: the pools that the router listed on the
// last call. It is a set, because getRoutes returns the pools in no fixed order.
type poolsListUpdaterMetadata struct {
	KnownPools []string `json:"knownPools"`
}

// The rpc types below receive ABI outputs. The decoder copies several outputs into a struct by
// name. It copies a single output into the first field of a struct, so a single tuple output goes
// into a tuple, and the tuple's components are copied by field order.

type tuple[T any] struct {
	Value T
}

type poolTokens struct {
	Base  common.Address
	Quote common.Address
}

// rpcPoolStateV1 is the router's PoolStateV1, field by field.
type rpcPoolStateV1 struct {
	BlockNumber            *big.Int
	BlockTimestamp         *big.Int
	Base                   common.Address
	Quote                  common.Address
	BaseDecimals           uint8
	QuoteDecimals          uint8
	Routed                 bool
	RouterPaused           bool
	PoolPaused             bool
	RouterSwapper          bool
	PoolPriceReader        bool
	PartnerRole            bool
	PartnerFeeRecipientSet bool
	PriceAvailable         bool
	MidPrice               uint64
	BidSpread              uint32
	AskSpread              uint32
	PriceTimestampMs       *big.Int
	MaxPriceAgeMs          uint32
	PositionLevelID        int16
	PositionFraction       *big.Int
	LowestLevelID          int16
	Levels                 []rpcPoolStateV1PriceLevel
}

// rpcPoolStateV1PriceLevel is the router's PoolStateV1PriceLevel, field by field.
type rpcPoolStateV1PriceLevel struct {
	LowerOffset    int32
	UpperOffset    int32
	BidExtraSpread uint32
	AskExtraSpread uint32
	BaseAmount     *big.Int
	QuoteAmount    *big.Int
}

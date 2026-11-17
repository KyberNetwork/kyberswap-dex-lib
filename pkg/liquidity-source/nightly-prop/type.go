package nightlyprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

// Extra is the pool's state that the router's getPoolStateV1 returned at the record's
// BlockNumber. The record has zero reserves unless the pool is swappable: the router routes its
// pair through it, neither the router nor the pool is paused, and the pool has a price.
type Extra struct {
	Routed         bool   `json:"routed"`
	RouterPaused   bool   `json:"routerPaused"`
	PoolPaused     bool   `json:"poolPaused"`
	PriceAvailable bool   `json:"priceAvailable"`
	BlockTimestamp uint64 `json:"blockTimestamp"`
	BaseDecimals   uint8  `json:"baseDecimals"`
	QuoteDecimals  uint8  `json:"quoteDecimals"`
	// Price is zero when PriceAvailable is false.
	Price         Price    `json:"price"`
	MaxPriceAgeMs uint32   `json:"maxPriceAgeMs"`
	Position      Position `json:"position"`
	LowestLevelID int16    `json:"lowestLevelId"`
	// Levels holds each level from LowestLevelID up to the highest level, without gaps.
	Levels []Level `json:"levels"`
}

// Price is the oracle price that the pool accepted. Spreads are millionths.
type Price struct {
	MidPrice    uint64 `json:"midPrice"`
	BidSpread   uint32 `json:"bidSpread"`
	AskSpread   uint32 `json:"askSpread"`
	TimestampMs uint64 `json:"timestampMs"`
}

type Position struct {
	LevelID  int16        `json:"levelId"`
	Fraction *uint256.Int `json:"fraction"`
}

// Level is one price level's configuration and its balances in native token units. Offsets and
// extra spreads are millionths.
type Level struct {
	LowerOffset    int32        `json:"lowerOffset"`
	UpperOffset    int32        `json:"upperOffset"`
	BidExtraSpread uint32       `json:"bidExtraSpread"`
	AskExtraSpread uint32       `json:"askExtraSpread"`
	BaseAmount     *uint256.Int `json:"baseAmount"`
	QuoteAmount    *uint256.Int `json:"quoteAmount"`
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
	BlockNumber      *big.Int
	BlockTimestamp   *big.Int
	Base             common.Address
	Quote            common.Address
	BaseDecimals     uint8
	QuoteDecimals    uint8
	Routed           bool
	RouterPaused     bool
	PoolPaused       bool
	PriceAvailable   bool
	MidPrice         uint64
	BidSpread        uint32
	AskSpread        uint32
	PriceTimestampMs *big.Int
	MaxPriceAgeMs    uint32
	PositionLevelID  int16
	PositionFraction *big.Int
	LowestLevelID    int16
	Levels           []rpcPoolStateV1PriceLevel
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

package thogprop

import "github.com/holiman/uint256"

type StaticExtra struct {
	PoolId string `json:"poolId"`
}
type Token struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
	Category uint8  `json:"category"`
}

// State is everything a quote needs, all read at BlockNumber. Fixed arrays copy
// by value; Tokens is immutable after construction.
type State struct {
	// Tokens lists the pool's tokens in pool order.
	Tokens        []Token `json:"tokens"`
	CategoryCount int     `json:"categoryCount"`
	// Words holds the pool's packed pricing state by storage slot; getPoolData
	// fills slots 2..127. layout.go names every field.
	Words [128]uint256.Int `json:"words"`
	// Balances is what the pool actually holds of each token. A share-based
	// token's balance is its shares valued at the pool's stored multiplier.
	Balances [maxTokens]uint256.Int `json:"balances"`
	// NativeBalance is the pool's native MON. It counts toward the risk of WMON's
	// category but cannot be paid out as WMON.
	NativeBalance uint256.Int `json:"nativeBalance"`
	BlockNumber   uint64      `json:"blockNumber"`
	BaseFee       uint256.Int `json:"baseFee"`
	// Shares holds the pool's share-based tokens; a zero multiplier marks an
	// ordinary ERC-20.
	Shares [maxTokens]ShareBalance `json:"shares"`
}
type Extra struct {
	State            State  `json:"state"`
	CapturedAtMs     int64  `json:"capturedAtMs,omitempty"`
	BlockTimestamp   uint64 `json:"blockTimestamp,omitempty"`
	BlockTimeMs      int64  `json:"blockTimeMs,omitempty"`
	MaxSnapshotAgeMs int64  `json:"maxSnapshotAgeMs,omitempty"`
}

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
	QuoteBlock      uint64 `json:"quoteBlock"`
	SnapshotAgeMs   int64  `json:"snapshotAgeMs"`
}

// ExecutionContext supplies actual fee and warmth conditions for settlement.
// The router advances its read-only quote block with elapsed snapshot time.
type ExecutionContext struct {
	BlockNumber        uint64
	BaseFee, GasPrice  uint256.Int
	AuctionHandlerWarm bool
}
type Quote struct {
	AmountIn, AmountOut, Notional, Penalty, VolumeCost uint256.Int
	PostedBlock                                        uint64
}
type SwapInfo struct {
	InputIndex, OutputIndex     int
	InputBalance, OutputBalance uint256.Int
	InputUnits, OutputUnits     uint256.Int
	QuoteBlock                  uint64 `json:"quoteBlock"`
	SnapshotAgeMs               int64  `json:"snapshotAgeMs"`
}
type Fraction struct{ Numerator, Denominator uint256.Int }

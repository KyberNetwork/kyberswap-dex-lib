package maverickv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

type State struct {
	ReserveA           *big.Int `json:"reserveA"`
	ReserveB           *big.Int `json:"reserveB"`
	LastTimestamp      int64    `json:"lastTimestamp"`
	LastTwaD8          int64    `json:"lastTwaD8"`
	LastLogPriceD8     int64    `json:"lastLogPriceD8"`
	ActiveTick         int32    `json:"activeTick"`
	IsLocked           bool     `json:"isLocked"`
	BinCounter         uint32   `json:"binCounter"`
	ProtocolFeeRatioD3 uint8    `json:"protocolFeeRatioD3"`
	FeeAIn             uint64   `json:"feeAIn"` // Fee for tokenA -> tokenB swaps
	FeeBIn             uint64   `json:"feeBIn"` // Fee for tokenB -> tokenA swaps
}

// FullPoolState represents the complete pool state from pool lens
type FullPoolState struct {
	TickStateMapping       []TickStateMapping       `json:"tickStateMapping"`
	BinStateMapping        []BinStateMapping        `json:"binStateMapping"`
	BinIdByTickKindMapping []BinIdByTickKindMapping `json:"binIdByTickKindMapping"`
	State                  PoolStateFromLens        `json:"state"`
	ProtocolFees           ProtocolFees             `json:"protocolFees"`
}

// FullPoolStateWrapper wraps the FullPoolState in a struct to match the contract's return type
type FullPoolStateWrapper struct {
	PoolState FullPoolState `json:"poolState"`
}

type TickStateMapping struct {
	ReserveA     *big.Int  `json:"reserveA"`
	ReserveB     *big.Int  `json:"reserveB"`
	TotalSupply  *big.Int  `json:"totalSupply"`
	BinIdsByTick [4]uint32 `json:"binIdsByTick"`
}

type BinStateMapping struct {
	MergeBinBalance *big.Int `json:"mergeBinBalance"`
	TickBalance     *big.Int `json:"tickBalance"`
	TotalSupply     *big.Int `json:"totalSupply"`
	Kind            uint8    `json:"kind"`
	Tick            int32    `json:"tick"`
	MergeId         uint32   `json:"mergeId"`
}

type BinIdByTickKindMapping struct {
	Values [4]*big.Int `json:"values"`
}

type PoolStateFromLens struct {
	ReserveA           *big.Int `json:"reserveA"`
	ReserveB           *big.Int `json:"reserveB"`
	LastTwaD8          int64    `json:"lastTwaD8"`
	LastLogPriceD8     int64    `json:"lastLogPriceD8"`
	LastTimestamp      *big.Int `json:"lastTimestamp"`
	ActiveTick         int32    `json:"activeTick"`
	IsLocked           bool     `json:"isLocked"`
	BinCounter         uint32   `json:"binCounter"`
	ProtocolFeeRatioD3 uint8    `json:"protocolFeeRatioD3"`
}

type ProtocolFees struct {
	AmountA *big.Int `json:"amountA"`
	AmountB *big.Int `json:"amountB"`
}

// MoveBinsParams contains parameters needed for the moveBins operation
type MoveBinsParams struct {
	StartingTick int32
	EndTick      int32
	OldTwaD8     int64
	NewTwaD8     int64
	Threshold    *uint256.Int
}

// MoveData contains parameters for bin movement operations
type MoveData struct {
	Kind            uint8             `json:"kind"`
	TickSearchStart int32             `json:"tickSearchStart"`
	TickSearchEnd   int32             `json:"tickSearchEnd"`
	TickLimit       int32             `json:"tickLimit"`
	FirstBinTick    int32             `json:"firstBinTick"`
	FirstBinId      uint32            `json:"firstBinId"`
	MergeBinBalance *uint256.Int      `json:"mergeBinBalance"`
	TotalReserveA   *uint256.Int      `json:"totalReserveA"`
	TotalReserveB   *uint256.Int      `json:"totalReserveB"`
	MergeBins       map[uint32]uint32 `json:"mergeBins"` // counter -> binId mapping
	Counter         uint32            `json:"counter"`
}

type StaticExtra struct {
	TickSpacing uint32 `json:"tickSpacing"`
}

type MaverickPoolState struct {
	FeeAIn           uint64
	FeeBIn           uint64
	ProtocolFeeRatio uint8
	Bins             map[uint32]Bin
	Ticks            map[int32]Tick
	TickSpacing      uint32
	ActiveTick       int32
	LastTwaD8        int64  // Time-weighted average tick data
	Timestamp        int64  // Current timestamp
	BinCounter       uint32 // Counter for bin IDs
}

type Extra struct {
	FeeAIn           uint64         `json:"feeAIn"`
	FeeBIn           uint64         `json:"feeBIn"`
	ProtocolFeeRatio uint8          `json:"protocolFeeRatio"`
	Bins             map[uint32]Bin `json:"bins"`
	Ticks            map[int32]Tick `json:"ticks"`
	ActiveTick       int32          `json:"activeTick"`
	LastTwaD8        int64          `json:"lastTwaD8"`
	Timestamp        int64          `json:"timestamp"`
}

type Bin struct {
	MergeBinBalance  *uint256.Int `json:"mergeBinBalance"`
	MergeId          uint32       `json:"mergeId"`
	TotalSupply      *uint256.Int `json:"totalSupply"`
	Kind             uint8        `json:"kind"`
	Tick             int32        `json:"tick"`
	TickBalance      *uint256.Int `json:"tickBalance"`
	CurrentLiquidity *uint256.Int `json:"currentLiquidity,omitempty"` // Added for TypeScript compatibility
}

type Delta struct {
	DeltaInBinInternal *uint256.Int
	DeltaInErc         *uint256.Int
	DeltaOutErc        *uint256.Int
	Excess             *uint256.Int
	TokenAIn           bool
	ExactOutput        bool
	TickLimit          int32
	SqrtLowerTickPrice *uint256.Int
	SqrtUpperTickPrice *uint256.Int
	SqrtPrice          *uint256.Int
	SwappedToMaxPrice  bool
	SkipCombine        bool
}

// Tick represents a tick's liquidity state
type Tick struct {
	ReserveA     *uint256.Int
	ReserveB     *uint256.Int
	TotalSupply  *uint256.Int
	BinIdsByTick map[uint8]uint32
}

type TickData struct {
	CurrentReserveA  *uint256.Int
	CurrentReserveB  *uint256.Int
	CurrentLiquidity *uint256.Int
}

type maverickSwapInfo struct {
	activeTick       int32
	bins             map[uint32]Bin
	ticks            map[int32]Tick
}

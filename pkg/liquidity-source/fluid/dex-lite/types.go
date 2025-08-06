package dexLite

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	DexKey          DexKey `json:"dexKey"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

// DexKey represents the unique identifier for a FluidDexLite pool
type DexKey struct {
	Token0 common.Address `json:"token0"`
	Token1 common.Address `json:"token1"`
	Salt   common.Hash    `json:"salt"`
}

// PoolState represents the 4 storage variables for a FluidDexLite pool
type PoolState struct {
	DexVariables     *uint256.Int `json:"dexVariables"`     // Packed dex variables
	CenterPriceShift *uint256.Int `json:"centerPriceShift"` // Center price shift variables
	RangeShift       *uint256.Int `json:"rangeShift"`       // Range shift variables
	ThresholdShift   *uint256.Int `json:"thresholdShift"`   // Threshold shift variables
}

func (p *PoolState) Clone() *PoolState {
	return &PoolState{
		DexVariables:     p.DexVariables.Clone(),
		CenterPriceShift: p.CenterPriceShift.Clone(),
		RangeShift:       p.RangeShift.Clone(),
		ThresholdShift:   p.ThresholdShift.Clone(),
	}
}

// UnpackedDexVariables represents the unpacked dex variables for easier access
type UnpackedDexVariables struct {
	Fee                         *uint256.Int `json:"fee"`
	RevenueCut                  *uint256.Int `json:"revenueCut"`
	RebalancingStatus           *uint256.Int `json:"rebalancingStatus"`
	CenterPriceShiftActive      bool         `json:"centerPriceShiftActive"`
	CenterPrice                 *uint256.Int `json:"centerPrice"`
	CenterPriceContractAddress  *uint256.Int `json:"centerPriceContractAddress"`
	RangePercentShiftActive     bool         `json:"rangePercentShiftActive"`
	UpperPercent                *uint256.Int `json:"upperPercent"`
	LowerPercent                *uint256.Int `json:"lowerPercent"`
	ThresholdPercentShiftActive bool         `json:"thresholdPercentShiftActive"`
	UpperShiftThresholdPercent  *uint256.Int `json:"upperShiftThresholdPercent"`
	LowerShiftThresholdPercent  *uint256.Int `json:"lowerShiftThresholdPercent"`
	Token0TotalSupplyAdjusted   *uint256.Int `json:"token0TotalSupplyAdjusted"`
	Token1TotalSupplyAdjusted   *uint256.Int `json:"token1TotalSupplyAdjusted"`
}

// PoolWithState represents a pool with its current state
type PoolWithState struct {
	DexId    [8]byte   `json:"dexId"`    // bytes8 dex identifier
	DexKey   DexKey    `json:"dexKey"`   // DexKey struct
	State    PoolState `json:"state"`    // Current pool state
	Fee      *big.Int  `json:"fee"`      // Pool fee
	IsActive bool      `json:"isActive"` // Whether pool is active
}

// StaticExtra represents static configuration that doesn't change
type StaticExtra struct {
	DexLiteAddress string `json:"dexLiteAddress"`
	HasNative      bool   `json:"hasNative"`
}

// PoolExtra represents the essential FluidDexLite pool data
type PoolExtra struct {
	DexKey         DexKey    `json:"dexKey"`         // The pool's key (token0, token1, salt)
	DexId          [8]byte   `json:"dexId"`          // Unique identifier for this pool
	PoolState      PoolState `json:"poolState"`      // The 4 storage variables
	BlockTimestamp uint64    `json:"blockTimestamp"` // Block timestamp when state was fetched
}

// SwapInfo contains information passed during swap execution
type SwapInfo struct {
	NewPoolState *PoolState `json:"-"`
}

// PricingResult represents the result of price calculations
type PricingResult struct {
	CenterPrice             *big.Int `json:"centerPrice"`
	UpperRangePrice         *big.Int `json:"upperRangePrice"`
	LowerRangePrice         *big.Int `json:"lowerRangePrice"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

// SwapResult represents the result of a swap calculation
type SwapResult struct {
	AmountOut *big.Int `json:"amountOut"`
	AmountIn  *big.Int `json:"amountIn"`
}

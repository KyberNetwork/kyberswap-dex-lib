package dexLite

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

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
	Fee                         *uint256.Int
	RevenueCut                  *uint256.Int
	RebalancingStatus           uint64
	CenterPriceShiftActive      bool
	CenterPrice                 *uint256.Int
	CenterPriceContractAddress  *uint256.Int
	RangePercentShiftActive     bool
	UpperPercent                *uint256.Int
	LowerPercent                *uint256.Int
	ThresholdPercentShiftActive bool
	UpperShiftThresholdPercent  *uint256.Int
	LowerShiftThresholdPercent  *uint256.Int
	Token0TotalSupplyAdjusted   *uint256.Int
	Token1TotalSupplyAdjusted   *uint256.Int
}

// StaticExtra represents static configuration that doesn't change
type StaticExtra struct {
	DexLiteAddress string `json:"dexLiteAddress"`
	DexKey         DexKey `json:"dexKey"` // The pool's key (token0, token1, salt)
	DexId          DexId  `json:"dexId"`  // Unique identifier for this pool
}

type DexId [8]byte

//goland:noinspection GoMixedReceiverTypes
func (dexId DexId) MarshalJSON() ([]byte, error) {
	return []byte(`"` + hexutil.Encode(dexId[:]) + `"`), nil
}

//goland:noinspection GoMixedReceiverTypes
func (dexId *DexId) UnmarshalJSON(data []byte) error {
	bytes, err := hexutil.Decode(string(data)[1 : len(data)-1])
	copy(dexId[:], bytes)
	return err
}

// PoolExtra represents the essential FluidDexLite pool data
type PoolExtra struct {
	PoolState      PoolState `json:"poolState"`      // The 4 storage variables
	BlockTimestamp uint64    `json:"blockTimestamp"` // Block timestamp when state was fetched
}

// PoolExtraMarshal marshals PoolExtra in a more compact and readable format
type PoolExtraMarshal struct {
	PoolState      PoolStateHex `json:"poolState"`      // The 4 storage variables
	BlockTimestamp uint64       `json:"blockTimestamp"` // Block timestamp when state was fetched
}

type PoolStateHex struct {
	DexVariables     string `json:"dexVariables"`     // Hex packed dex variables
	CenterPriceShift string `json:"centerPriceShift"` // Hex center price shift variables
	RangeShift       string `json:"rangeShift"`       // Hex range shift variables
	ThresholdShift   string `json:"thresholdShift"`   // Hex threshold shift variables
}

// SwapInfo contains information passed during swap execution
type SwapInfo struct {
	NewDexVars *UnpackedDexVariables `json:"-"`
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

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	DexKey          DexKey `json:"dexKey"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

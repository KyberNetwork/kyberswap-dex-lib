package dexLite

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

// StaticExtra represents static configuration that doesn't change
type StaticExtra struct {
	DexLiteAddress string `json:"l"`
	DexKey         DexKey `json:"k"` // The pool's key (token0, token1, salt)
	DexId          DexId  `json:"i"` // Unique identifier for this pool
}

// DexKey represents the unique identifier for a FluidDexLite pool
type DexKey struct {
	Token0 common.Address `json:"t0"`
	Token1 common.Address `json:"t1"`
	Salt   common.Hash    `json:"s"`
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
	PoolState      PoolState `json:"pS"` // The 4 storage variables
	BlockTimestamp uint64    `json:"ts"` // Block timestamp when state was fetched
}

// PoolExtraMarshal marshals PoolExtra in a more compact and readable format
type PoolExtraMarshal struct {
	PoolState      PoolStateHex `json:"pS"` // The 4 storage variables
	BlockTimestamp uint64       `json:"ts"` // Block timestamp when state was fetched
}

// PoolState represents the 4 storage variables for a FluidDexLite pool
type PoolState struct {
	DexVariables     *uint256.Int `json:"dV,omitempty"` // Packed dex variables
	CenterPriceShift *uint256.Int `json:"pS,omitempty"` // Center price shift variables
	RangeShift       *uint256.Int `json:"rS,omitempty"` // Range shift variables
	NewCenterPrice   *uint256.Int `json:"nP,omitempty"` // New center price from external source
}

type PoolStateHex struct {
	DexVariables     string `json:"dV,omitempty"` // Hex packed dex variables
	CenterPriceShift string `json:"pS,omitempty"` // Hex center price shift variables
	RangeShift       string `json:"rS,omitempty"` // Hex range shift variables
	NewCenterPrice   string `json:"nP,omitempty"` // Hex new center price from external source
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

// SwapInfo contains information passed during swap execution
type SwapInfo struct {
	NewDexVars *UnpackedDexVariables `json:"-"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	DexKey          DexKey `json:"dexKey"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

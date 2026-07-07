package maverickv1

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Metadata struct {
	LastCreateTime uint64
}

type SubgraphPool struct {
	ID          string        `json:"id"`
	Fee         string        `json:"fee"`
	TickSpacing string        `json:"tickSpacing"`
	Timestamp   string        `json:"timestamp"`
	TokenA      SubgraphToken `json:"tokenA"`
	TokenB      SubgraphToken `json:"tokenB"`
}

type SubgraphToken struct {
	ID       string `json:"id"`
	Decimals uint8  `json:"decimals"`
}

type Extra struct {
	Fee              *uint256.Int               `json:"fee"` // 18 decimals
	ProtocolFeeRatio *uint256.Int               `json:"protocFeeRatio"`
	Bins             map[uint32]Bin             `json:"bins"`
	BinPositions     map[int32]map[uint8]uint32 `json:"binPosMap"`
	BinMap           map[int16]*uint256.Int     `json:"binMap"`
	ActiveTick       int32                      `json:"tick"`

	// State to calculate TVL
	Liquidity    *uint256.Int `json:"liquidity"`
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
}

type StaticExtra struct {
	TickSpacing int32 `json:"tickSpacing"`
}

type MaverickPoolState struct {
	Fee              *uint256.Int // 18 decimals
	ProtocolFeeRatio *uint256.Int
	Bins             map[uint32]Bin
	BinPositions     map[int32]map[uint8]uint32
	BinMap           map[int16]*uint256.Int
	TickSpacing      int32
	ActiveTick       int32

	minBinMapIndex int16
	maxBinMapIndex int16
}

// maverickSwapInfo present the after state of a swap
type maverickSwapInfo struct {
	bins       map[uint32]Bin
	activeTick int32
}

type Bin struct {
	ReserveA  *uint256.Int `json:"rA"`
	ReserveB  *uint256.Int `json:"rB"`
	LowerTick int32        `json:"lT"`
	Kind      uint8        `json:"k"`
}

type Gas struct {
	Swap     int64
	CrossBin int64
}

type Delta struct {
	DeltaInBinInternal *uint256.Int
	DeltaInErc         *uint256.Int
	DeltaOutErc        *uint256.Int
	Excess             *uint256.Int
	SqrtPriceLimit     *uint256.Int
	SqrtLowerTickPrice *uint256.Int
	SqrtUpperTickPrice *uint256.Int
	SqrtPrice          *uint256.Int
	TokenAIn           bool
	ExactOutput        bool
	SwappedToMaxPrice  bool
	SkipCombine        bool
	DecrementTick      bool
}

type Active struct {
	Word uint8 // 4 bits
	Tick int32
}

type GetStateResult struct {
	State struct {
		ActiveTick       int32
		Status           uint8
		BinCounter       *big.Int
		ProtocolFeeRatio uint64
	}
}

type GetBinResult struct {
	BinState struct {
		ReserveA        *big.Int
		ReserveB        *big.Int
		MergeBinBalance *big.Int
		MergeID         *big.Int
		TotalSupply     *big.Int
		Kind            uint8
		LowerTick       int32
	}
}

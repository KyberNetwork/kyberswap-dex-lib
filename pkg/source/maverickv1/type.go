package maverickv1

import "math/big"

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

type StaticExtra struct {
	TickSpacing *big.Int `json:"tickSpacing"`
}

type Extra struct {
	Fee              *big.Int                       `json:"fee"`
	ProtocolFeeRatio *big.Int                       `json:"protocolFeeRatio"`
	ActiveTick       *big.Int                       `json:"activeTick"`
	BinCounter       *big.Int                       `json:"binCounter"`
	Bins             map[string]Bin                 `json:"bins"`
	BinPositions     map[string]map[string]*big.Int `json:"binPositions"`
	BinMap           map[string]*big.Int            `json:"binMap"`
}

type MaverickPoolState struct {
	TickSpacing      *big.Int                       `json:"tickSpacing"`
	Fee              *big.Int                       `json:"fee"`
	ProtocolFeeRatio *big.Int                       `json:"protocolFeeRatio"`
	ActiveTick       *big.Int                       `json:"activeTick"`
	BinCounter       *big.Int                       `json:"binCounter"`
	Bins             map[string]Bin                 `json:"bins"`
	BinPositions     map[string]map[string]*big.Int `json:"binPositions"`
	BinMap           map[string]*big.Int            `json:"binMap"`
}

// MaverickSwapInfo present the after state of a swap
type MaverickSwapInfo struct {
	Bins       map[string]Bin `json:"bins"`
	ActiveTick *big.Int       `json:"activeTick"`
}

type Bin struct {
	ReserveA  *big.Int `json:"reserveA"`
	ReserveB  *big.Int `json:"reserveB"`
	LowerTick *big.Int `json:"lowerTick"`
	Kind      *big.Int `json:"kind"`
	MergeID   *big.Int `json:"mergeId"`
}

type Gas struct {
	Swap int64
}

type Delta struct {
	DeltaInBinInternal *big.Int
	DeltaInErc         *big.Int
	DeltaOutErc        *big.Int
	Excess             *big.Int
	TokenAIn           bool
	EndSqrtPrice       *big.Int
	ExactOutput        bool
	SwappedToMaxPrice  bool
	SkipCombine        bool
	DecrementTick      bool
	SqrtPriceLimit     *big.Int
	SqrtLowerTickPrice *big.Int
	SqrtUpperTickPrice *big.Int
	SqrtPrice          *big.Int
}

type Active struct {
	Word *big.Int
	Tick *big.Int
}

type GetStateResult struct {
	State struct {
		ActiveTick       int32    `json:"activeTick"`
		Status           uint8    `json:"status"`
		BinCounter       *big.Int `json:"binCounter"`
		ProtocolFeeRatio uint64   `json:"protocolFeeRatio"`
	}
}

type GetBinResult struct {
	BinState struct {
		ReserveA        *big.Int `json:"reserveA"`
		ReserveB        *big.Int `json:"reserveB"`
		MergeBinBalance *big.Int `json:"mergeBinBalance"`
		MergeID         *big.Int `json:"mergeId"`
		TotalSupply     *big.Int `json:"totalSupply"`
		Kind            uint8    `json:"kind"`
		LowerTick       int32    `json:"lowerTick"`
	}
}

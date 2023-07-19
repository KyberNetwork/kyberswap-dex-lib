package maverick

import "math/big"

type Metadata struct {
	LastCreateTime *big.Int
}

type SubgraphPool struct {
	ID          string        `json:"id"`
	Fee         float64       `json:"fee"`
	TickSpacing *big.Int      `json:"tickSpacing"`
	Timestamp   *big.Int      `json:"timestamp"`
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

type Bin struct {
	ReserveA  *big.Int `json:"reserveA"`
	ReserveB  *big.Int `json:"reserveB"`
	LowerTick *big.Int `json:"lowerTick"`
	Kind      *big.Int `json:"kind"`
	MergeID   *big.Int `json:"mergeID"`
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
	ActiveTick       *big.Int `json:"activeTick"`
	Status           *big.Int `json:"status"`
	BinCounter       *big.Int `json:"binCounter"`
	ProtocolFeeRatio *big.Int `json:"protocolFeeRatio"`
}

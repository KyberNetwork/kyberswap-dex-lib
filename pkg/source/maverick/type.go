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
	TickSpacing      *big.Int
	Fee              *big.Int
	ProtocolFeeRatio *big.Int
	ActiveTick       *big.Int
	BinCounter       *big.Int
	Bins             map[string]Bin
	BinPositions     map[string]map[string]*big.Int
	BinMap           map[string]*big.Int
}

type Bin struct {
	ReserveA  *big.Int
	ReserveB  *big.Int
	LowerTick *big.Int
	Kind      *big.Int
	MergeID   *big.Int
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

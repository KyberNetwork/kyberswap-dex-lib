package carbon

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Order struct {
	Y *uint256.Int `json:"y,omitempty"`
	Z *uint256.Int `json:"z,omitempty"`
	A uint64       `json:"A,omitempty"`
	B uint64       `json:"B,omitempty"`
}

func (o *Order) Clone() Order {
	if o == nil {
		return Order{}
	}

	var y, z *uint256.Int
	if o.Y != nil {
		y = o.Y.Clone()
	}
	if o.Z != nil {
		z = o.Z.Clone()
	}

	return Order{Y: y, Z: z, A: o.A, B: o.B}
}

type Strategy struct {
	Id     *big.Int `json:"id"`
	Orders [2]Order `json:"orders"`
}

func (s *Strategy) Clone() Strategy {
	if s == nil {
		return Strategy{}
	}

	return Strategy{Id: s.Id, Orders: [2]Order{s.Orders[0].Clone(), s.Orders[1].Clone()}}
}

type Pair struct {
	Id     *big.Int          `json:"id"`
	Tokens [2]common.Address `json:"tokens"`
}

type Extra struct {
	Strategies       []Strategy `json:"strategies"`
	TradingFeePpm    uint32     `json:"tradingFeePpm"`
	LastFullScanTime int64      `json:"u,omitempty"`
	// StrategyCount is the on-chain strategiesByPairCount as of this state. Strategies
	// dust-filtered on both sides aren't stored at all, so this - not len(Strategies) - is
	// what incrementalScan diffs against to detect newly created strategies.
	StrategyCount int64 `json:"n,omitempty"`
}

type Meta struct {
	BlockNumber     uint64 `json:"bN"`
	IsNativeIn      bool   `json:"iN,omitempty"`
	IsNativeOut     bool   `json:"oN,omitempty"`
	TradingFeePpm   uint32 `json:"tF,omitempty"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type SwapInfo struct {
	TradeActions []TradeAction `json:"actions"`
}

type TradeAction struct {
	StrategyId      string       `json:"sI"`
	SourceAmount    *uint256.Int `json:"sA"`
	TargetAmount    *uint256.Int `json:"tA"`
	strategyIdx     int
	isToken0To1     bool
	newTargetOrderY *uint256.Int
	newSourceOrderY *uint256.Int
	newSourceOrderZ *uint256.Int
}

type StaticExtra struct {
	Token0     string `json:"t0"`
	Token1     string `json:"t1"`
	Controller string `json:"c"`
}

type TradeOutput struct {
	AmountOutAfterFee *uint256.Int
	FeeAmount         *uint256.Int
	TradeActions      []TradeAction
}

type TradeResults struct {
	Best *TradeOutput
	Fast *TradeOutput
}

type StrategyByPairResp struct {
	ID     *big.Int
	Owner  common.Address
	Tokens [2]common.Address
	Orders [2]struct {
		Y *big.Int
		Z *big.Int
		A uint64
		B uint64
	}
}

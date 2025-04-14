package quoting

import (
	"math/big"
)

type StateAfter struct {
	SqrtRatio       *big.Int
	Liquidity       *big.Int
	ActiveTickIndex int
}

type SwapInfo struct {
	SkipAhead           uint32     `json:"skipAhead"`
	IsToken1            bool       `json:"isToken1"`
	StateAfter          StateAfter `json:"-"`
	TickSpacingsCrossed uint32     `json:"-"`
}

type Quote struct {
	ConsumedAmount   *big.Int
	CalculatedAmount *big.Int
	FeesPaid         *big.Int
	Gas              int64
	SwapInfo         SwapInfo
}

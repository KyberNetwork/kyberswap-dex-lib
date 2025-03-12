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
	StateAfter StateAfter
	SkipAhead  uint32
}

type Quote struct {
	ConsumedAmount   *big.Int
	CalculatedAmount *big.Int
	FeesPaid         *big.Int
	Gas              int64
	SwapInfo         SwapInfo
}

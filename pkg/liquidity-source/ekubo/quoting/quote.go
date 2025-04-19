package quoting

import (
	"math/big"
)

type (
	SwapState = any

	SwapInfo struct {
		SkipAhead           uint32    `json:"skipAhead"`
		IsToken1            bool      `json:"isToken1"`
		SwapStateAfter      SwapState `json:"-"`
		TickSpacingsCrossed uint32    `json:"-"`
	}

	Quote struct {
		ConsumedAmount   *big.Int
		CalculatedAmount *big.Int
		FeesPaid         *big.Int
		Gas              int64
		SwapInfo         SwapInfo
	}
)

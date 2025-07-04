package quoting

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type (
	SwapState = any

	SwapInfo struct {
		SkipAhead           uint32         `json:"skipAhead"`
		IsToken1            bool           `json:"isToken1"`
		Forward             common.Address `json:"forward"`
		SwapStateAfter      SwapState      `json:"-"`
		TickSpacingsCrossed uint32         `json:"-"`
	}

	Quote struct {
		ConsumedAmount   *big.Int
		CalculatedAmount *big.Int
		FeesPaid         *big.Int
		Gas              int64
		SwapInfo         SwapInfo
	}
)

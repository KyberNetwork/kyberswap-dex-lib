package quoting

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type (
	SwapState = any

	SwapInfo struct {
		SkipAhead           uint32          `json:"skipAhead,omitempty"`
		IsToken1            bool            `json:"isToken1,omitempty"`
		Forward             *common.Address `json:"forward,omitempty"`
		SwapStateAfter      SwapState       `json:"-"`
		TickSpacingsCrossed uint32          `json:"-"`
	}

	Quote struct {
		ConsumedAmount   *big.Int
		CalculatedAmount *big.Int
		FeesPaid         *big.Int
		Gas              int64
		SwapInfo         SwapInfo
	}
)

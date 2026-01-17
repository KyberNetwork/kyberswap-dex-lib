package quoting

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type (
	SwapState = any

	SwapInfo struct {
		SkipAhead           uint32          `json:"skipAhead,omitempty"`
		IsToken1            bool            `json:"isToken1,omitempty"`
		PriceLimit          *uint256.Int    `json:"lim,omitempty"`
		Forward             *common.Address `json:"forward,omitempty"`
		SwapStateAfter      SwapState       `json:"-"`
		TickSpacingsCrossed uint32          `json:"-"`
	}

	Quote struct {
		ConsumedAmount   *uint256.Int
		CalculatedAmount *uint256.Int
		FeesPaid         *uint256.Int
		Gas              int64
		SwapInfo         SwapInfo
	}
)

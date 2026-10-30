package thogprop

import "github.com/holiman/uint256"

type StaticExtra struct {
	PoolId string `json:"poolId"`
}
type Token struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
	Category uint8  `json:"category"`
}

// State owns the complete schema-8/model-3 portfolio. Fixed arrays copy by value.
// Only active token metadata is a slice; it is immutable after construction.
type State struct {
	Tokens        []Token                 `json:"tokens"`
	CategoryCount int                     `json:"categoryCount"`
	Words         [128]uint256.Int        `json:"words"`
	Balances      [maxTokens]uint256.Int  `json:"balances"`
	NativeBalance uint256.Int             `json:"nativeBalance"`
	BlockNumber   uint64                  `json:"blockNumber"`
	BaseFee       uint256.Int             `json:"baseFee"`
	Shares        [maxTokens]ShareBalance `json:"shares"`
}
type Extra struct {
	State State `json:"state"`
}

// ExecutionContext supplies actual fee and warmth conditions for settlement.
// The router uses a separate, explicit one-block read-only quote policy.
type ExecutionContext struct {
	BlockNumber        uint64
	BaseFee, GasPrice  uint256.Int
	AuctionHandlerWarm bool
}
type Quote struct {
	AmountIn, AmountOut, Notional, Penalty, VolumeCost uint256.Int
	PostedBlock                                        uint64
}
type SwapInfo struct {
	InputIndex, OutputIndex     int
	InputBalance, OutputBalance uint256.Int
	InputUnits, OutputUnits     uint256.Int
}
type Fraction struct{ Numerator, Denominator uint256.Int }

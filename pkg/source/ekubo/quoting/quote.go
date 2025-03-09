package quoting

import (
	"math/big"
)

type Quote struct {
	ConsumedAmount   *big.Int
	CalculatedAmount *big.Int
	FeesPaid         *big.Int
	Gas              int64
	SkipAhead        uint32
}

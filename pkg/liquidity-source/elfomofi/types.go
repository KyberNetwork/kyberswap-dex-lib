package elfomofi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	FactoryAddress string `json:"factoryAddress"`
}

// CumulativeLevel is a helper probe, denominated in the tokens' raw units.
type CumulativeLevel struct {
	AmountIn  *big.Int
	AmountOut *big.Int
}

// HelperOrderbook is getOrderbook(base, quote)'s first return value.
// Asks spend quote for base; bids spend base for quote.
type HelperOrderbook struct {
	Base                common.Address
	Quote               common.Address
	AskCumulativeLevels []CumulativeLevel
	BidCumulativeLevels []CumulativeLevel
	BalanceBase         *big.Int
	BalanceQuote        *big.Int
}

// The helper has three return values; ethrpc decodes them into one struct.
type getOrderbookResult struct {
	Book           HelperOrderbook
	BlockNumber    *big.Int
	BlockTimestamp *big.Int
}

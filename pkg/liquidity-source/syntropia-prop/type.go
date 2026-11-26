package syntropiaprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	Pool string `json:"pool"`
}

// TokenPair is one entry of getPairs(), with token0 < token1.
type TokenPair struct {
	Token0 common.Address
	Token1 common.Address
}

// Level is one probe of the lens: amountIn spent in one swap returns amountOut, in raw units.
type Level struct {
	AmountIn  *big.Int
	AmountOut *big.Int
}

// Book is getOrderbook(base, quote)'s first return value. Asks spend quote for base and bids
// spend base for quote. A side the pool will not fill has no levels.
type Book struct {
	Base                common.Address
	Quote               common.Address
	AskCumulativeLevels []Level
	BidCumulativeLevels []Level
	BalanceBase         *big.Int
	BalanceQuote        *big.Int
}

// The lens has three return values; ethrpc decodes them into one struct.
type getOrderbookResult struct {
	Book           Book
	BlockNumber    *big.Int
	BlockTimestamp *big.Int
}

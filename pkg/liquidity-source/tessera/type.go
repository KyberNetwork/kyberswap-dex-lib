package tessera

import (
	"math/big"

	"github.com/holiman/uint256"
)

type poolSwapViewAmounts struct {
	AmountIn  *big.Int
	AmountOut *big.Int
}

type LiquidityLevel struct {
	Amount *uint256.Int `json:"amount"`
	Price  uint64       `json:"price"`
}

type StaticExtra struct {
	TesseraSwap string `json:"tesseraSwap"`
}

type Extra struct {
	BaseToQuotePrefetches []PrefetchRate `json:"b2q"`
	QuoteToBasePrefetches []PrefetchRate `json:"q2b"`

	// Max amounts from prefetch points
	// Only support swaps up to this limit with high accuracy
	// Quoter may accept larger amounts but interpolation has no data points beyond this range
	MaxBaseToQuoteAmount *uint256.Int `json:"maxB2Q,omitempty"`
	MaxQuoteToBaseAmount *uint256.Int `json:"maxQ2B,omitempty"`

	// Revert condition flags
	TradingEnabled bool `json:"tradingEnabled"`
	IsInitialised  bool `json:"isInitialised"`
}

type PrefetchRate struct {
	AmountIn *uint256.Int `json:"amountIn"`
	Rate     *uint256.Int `json:"rate"`
}

type poolStateLevel struct {
	Amount *big.Int `abi:"amount"`
	Price  *big.Int `abi:"price"`
	Active *big.Int `abi:"active"`
}

type poolStateResult struct {
	PoolOffset0       *big.Int           `abi:"poolOffset0"`
	PoolOffset1       *big.Int           `abi:"poolOffset1"`
	LpFeeRate         uint32             `abi:"lpFeeRate"`
	MtFeeRate         uint32             `abi:"mtFeeRate"`
	Side              uint8              `abi:"side"`
	TradingEnabled    bool               `abi:"tradingEnabled"`
	StartBlock        uint64             `abi:"startBlock"`
	DecayDuration     uint64             `abi:"decayDuration"`
	InitialFeeRate    uint32             `abi:"initialFeeRate"`
	MinimumFeeRate    uint32             `abi:"minimumFeeRate"`
	AnchorPrice       uint32             `abi:"tesseraAnchorPrice"`
	IsWhitelistActive bool               `abi:"isWhitelistActive"`
	WhitelistFeeRate  uint32             `abi:"whitelistFeeRate"`
	LiquidatorFeeRate uint32             `abi:"liquidatorFeeRate"`
	OrderBook0        [20]poolStateLevel `abi:"orderBook0"`
	OrderBook1        [20]poolStateLevel `abi:"orderBook1"`
}

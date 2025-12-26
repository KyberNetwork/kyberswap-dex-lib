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

type Extra struct {
	BaseToQuotePrefetches []PrefetchRate `json:"b2qPrefetches,omitempty"`
	QuoteToBasePrefetches []PrefetchRate `json:"q2bPrefetches,omitempty"`

	// Revert condition flags
	TradingEnabled bool `json:"tradingEnabled"`
	IsInitialised  bool `json:"isInitialised"`
}

type PrefetchRate struct {
	AmountIn *uint256.Int `json:"amountIn"`
	Rate     *uint256.Int `json:"rate"`
}

type StaticExtra struct {
	BaseToken  string `json:"baseToken"`
	QuoteToken string `json:"quoteToken"`
	EngineAddr string `json:"engineAddr"`
}

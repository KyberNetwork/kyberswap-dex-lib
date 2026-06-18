package tokentax

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
)

// Result is the normalized transfer-tax state persisted by the pool tracker.
// Tax rates are expressed in basis points.
type Result struct {
	Protocol     string
	TokenAddress string
	BuyTaxBps    *uint256.Int
	SellTaxBps   *uint256.Int
	Checked      bool
}

// Tracker appends protocol-specific reads to a shared multicall and normalizes their outputs.
type Tracker interface {
	AddTaxCalls(*ethrpc.Request) bool
	TaxResult() Result
}

// Handler applies normalized transfer tax around the AMM calculation.
type Handler interface {
	ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int
	ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int
}

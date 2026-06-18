package tokentax

import "github.com/holiman/uint256"

type NoopHandler struct{}

func (NoopHandler) ApplySellTax(_ string, amountIn *uint256.Int) *uint256.Int {
	return amountIn
}

func (NoopHandler) ApplyBuyTax(_ string, grossOut *uint256.Int) *uint256.Int {
	return grossOut
}

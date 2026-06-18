package uniswapv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

// TokenTaxHook applies a token's transfer tax around the AMM formula:
// sell tax shrinks the input the pair receives, buy tax shrinks the output the user receives.
// The hook is immutable, so it is safe to share across cloned simulators.
type TokenTaxHook interface {
	ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int
	ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int
}

// noopTokenTaxHook is used for pools without a tax token.
type noopTokenTaxHook struct{}

func (noopTokenTaxHook) ApplySellTax(_ string, amountIn *uint256.Int) *uint256.Int { return amountIn }
func (noopTokenTaxHook) ApplyBuyTax(_ string, grossOut *uint256.Int) *uint256.Int  { return grossOut }

// virtualTokenTaxHook deducts floor(amount * bps / 10000) on swaps involving taxToken.
type virtualTokenTaxHook struct {
	taxToken string
	buyTax   *uint256.Int
	sellTax  *uint256.Int
}

func newTokenTaxHook(taxToken string, buyTax, sellTax *uint256.Int) TokenTaxHook {
	if taxToken == "" {
		return noopTokenTaxHook{}
	}
	return &virtualTokenTaxHook{taxToken: taxToken, buyTax: buyTax, sellTax: sellTax}
}

func (h *virtualTokenTaxHook) ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int {
	if tokenIn != h.taxToken || h.sellTax == nil || h.sellTax.IsZero() {
		return amountIn
	}
	var tax uint256.Int
	tax.Div(tax.Mul(amountIn, h.sellTax), big256.UBasisPoint)
	return new(uint256.Int).Sub(amountIn, &tax)
}

func (h *virtualTokenTaxHook) ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int {
	if tokenOut != h.taxToken || h.buyTax == nil || h.buyTax.IsZero() {
		return grossOut
	}
	var tax uint256.Int
	tax.Div(tax.Mul(grossOut, h.buyTax), big256.UBasisPoint)
	return new(uint256.Int).Sub(grossOut, &tax)
}

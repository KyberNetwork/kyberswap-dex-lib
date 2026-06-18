package uniswapv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

// TokenTaxHandler applies a token's transfer tax around the AMM formula:
// sell tax shrinks the input the pair receives, buy tax shrinks the output the user receives.
// The taxHandler is immutable, so it is safe to share across cloned simulators.
type TokenTaxHandler interface {
	ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int
	ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int
}

// noopTokenTaxHandler is used for pools without a tax token.
type noopTokenTaxHandler struct{}

func (noopTokenTaxHandler) ApplySellTax(_ string, amountIn *uint256.Int) *uint256.Int {
	return amountIn
}
func (noopTokenTaxHandler) ApplyBuyTax(_ string, grossOut *uint256.Int) *uint256.Int { return grossOut }

// virtualTokenTaxHandler deducts floor(amount * bps / 10000) on swaps involving taxToken.
type virtualTokenTaxHandler struct {
	taxToken string
	buyTax   *uint256.Int
	sellTax  *uint256.Int
}

func NewTaxHandler(taxToken string, buyTax, sellTax *uint256.Int) TokenTaxHandler {
	if taxToken == "" {
		return noopTokenTaxHandler{}
	}
	return &virtualTokenTaxHandler{taxToken: taxToken, buyTax: buyTax, sellTax: sellTax}
}

func (h *virtualTokenTaxHandler) ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int {
	if tokenIn != h.taxToken || h.sellTax == nil || h.sellTax.IsZero() {
		return amountIn
	}
	var tax uint256.Int
	tax.Div(tax.Mul(amountIn, h.sellTax), big256.UBasisPoint)
	return new(uint256.Int).Sub(amountIn, &tax)
}

func (h *virtualTokenTaxHandler) ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int {
	if tokenOut != h.taxToken || h.buyTax == nil || h.buyTax.IsZero() {
		return grossOut
	}
	var tax uint256.Int
	tax.Div(tax.Mul(grossOut, h.buyTax), big256.UBasisPoint)
	return new(uint256.Int).Sub(grossOut, &tax)
}

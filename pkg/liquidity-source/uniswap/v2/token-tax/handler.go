package tokentax

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func NewHandler(info TaxInfo) Handler {
	return Handler{
		TokenAddress: info.Token,
		BuyTaxBps:    info.BuyTaxBps,
		SellTaxBps:   info.SellTaxBps,
	}
}

func (h Handler) HasSellTax(tokenIn string) bool {
	return tokenIn == h.TokenAddress && h.SellTaxBps != nil && !h.SellTaxBps.IsZero()
}

func (h Handler) ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int {
	if !h.HasSellTax(tokenIn) {
		return amountIn
	}
	return deductTax(amountIn, h.SellTaxBps)
}

func (h Handler) HasBuyTax(tokenOut string) bool {
	return tokenOut == h.TokenAddress && h.BuyTaxBps != nil && !h.BuyTaxBps.IsZero()
}

func (h Handler) ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int {
	if !h.HasBuyTax(tokenOut) {
		return grossOut
	}
	return deductTax(grossOut, h.BuyTaxBps)
}

func deductTax(amount, taxBps *uint256.Int) *uint256.Int {
	var tax uint256.Int
	tax.Div(tax.Mul(amount, taxBps), big256.UBasisPoint)
	return tax.Sub(amount, &tax)
}

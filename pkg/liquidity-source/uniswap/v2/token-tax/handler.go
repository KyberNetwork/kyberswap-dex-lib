package tokentax

import (
	"github.com/holiman/uint256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// NewHandler builds a Handler from persisted TaxInfo. info == nil (never tracked, or an ordinary
// non-taxed pool) yields the zero-value Handler, which every method already treats as a no-op.
func NewHandler(info *TaxInfo) Handler {
	if info == nil {
		return Handler{}
	}
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

func (h Handler) GrossUpSellTax(tokenIn string, netIn *uint256.Int) *uint256.Int {
	if !h.HasSellTax(tokenIn) {
		return netIn
	}
	return grossUpTax(netIn, h.SellTaxBps)
}

func (h Handler) GrossUpBuyTax(tokenOut string, netOut *uint256.Int) *uint256.Int {
	if !h.HasBuyTax(tokenOut) {
		return netOut
	}
	return grossUpTax(netOut, h.BuyTaxBps)
}

func deductTax(amount, taxBps *uint256.Int) *uint256.Int {
	var tax uint256.Int
	tax.Div(tax.Mul(amount, taxBps), big256.UBasisPoint)
	return tax.Sub(amount, &tax)
}

func grossUpTax(netAmount, taxBps *uint256.Int) *uint256.Int {
	var denom, gross uint256.Int
	denom.Sub(big256.UBasisPoint, taxBps)
	big256.MulDivUp(&gross, netAmount, big256.UBasisPoint, &denom)
	return &gross
}

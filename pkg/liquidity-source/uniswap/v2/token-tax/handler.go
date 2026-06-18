package tokentax

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func NewBasisPointHandler(result Result) Handler {
	if result.TokenAddress == "" {
		return NoopHandler{}
	}
	return basisPointHandler{
		tokenAddress: result.TokenAddress,
		buyTaxBps:    result.BuyTaxBps,
		sellTaxBps:   result.SellTaxBps,
	}
}

type basisPointHandler struct {
	tokenAddress string
	buyTaxBps    *uint256.Int
	sellTaxBps   *uint256.Int
}

func (h basisPointHandler) ApplySellTax(tokenIn string, amountIn *uint256.Int) *uint256.Int {
	if tokenIn != h.tokenAddress || h.sellTaxBps == nil || h.sellTaxBps.IsZero() {
		return amountIn
	}
	return deductTax(amountIn, h.sellTaxBps)
}

func (h basisPointHandler) ApplyBuyTax(tokenOut string, grossOut *uint256.Int) *uint256.Int {
	if tokenOut != h.tokenAddress || h.buyTaxBps == nil || h.buyTaxBps.IsZero() {
		return grossOut
	}
	return deductTax(grossOut, h.buyTaxBps)
}

func deductTax(amount, taxBps *uint256.Int) *uint256.Int {
	var tax uint256.Int
	tax.Div(tax.Mul(amount, taxBps), big256.UBasisPoint)
	return new(uint256.Int).Sub(amount, &tax)
}

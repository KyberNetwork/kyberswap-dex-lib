package lo1inch

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var amountIn uint256.Int
	if param.TokenAmountIn.Amount != nil && amountIn.SetFromBig(param.TokenAmountIn.Amount) {
		return nil, ErrCannotFulfillAmountIn
	}
	return p.quoteOut(param.TokenAmountIn.Token, param.TokenOut, &amountIn, param.Limit)
}

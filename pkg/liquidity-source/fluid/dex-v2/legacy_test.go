package dexv2

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if err := p.checkSupported(); err != nil {
		return nil, err
	}
	var amountIn uint256.Int
	if amt := param.TokenAmountIn.Amount; amt == nil || amt.Sign() < 0 || amountIn.SetFromBig(amt) {
		return nil, ErrAmountOutOfLimits
	}
	return p.calcAmountOut(param.TokenAmountIn.Token, param.TokenOut, &amountIn, param.Limit)
}

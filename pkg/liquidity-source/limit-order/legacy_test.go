package limitorder

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(param.TokenAmountIn, param.TokenOut, swapLimitFromBig(param.Limit))
}

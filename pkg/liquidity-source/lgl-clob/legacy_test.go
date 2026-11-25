package lglclob

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult,
	err error) {
	amtIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}
	var limit func(key string) *uint256.Int
	if param.Limit != nil {
		limit = func(key string) *uint256.Int { return toU256(param.Limit.GetLimit(key)) }
	}
	q, err := p.quote(param.TokenAmountIn.Token, amtIn, param.TokenOut, limit)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: q.tokenOut, Amount: q.amountOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: q.feeToken, Amount: q.fee.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: q.tokenIn, Amount: q.remainingIn.ToBig()},
		Gas:                    q.gas,
		SwapInfo:               q.swapInfo,
	}, nil
}

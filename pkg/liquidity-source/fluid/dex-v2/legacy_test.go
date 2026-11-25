package dexv2

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.
// The amount math was already uint256; only the amount-in, limit and result boundary was big.Int.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if err := p.checkSupported(); err != nil {
		return nil, err
	}
	var amountIn uint256.Int
	if amt := param.TokenAmountIn.Amount; amt == nil || amt.Sign() < 0 || amountIn.SetFromBig(amt) {
		return nil, ErrAmountOutOfLimits
	}
	var get limitGetter
	if limit := param.Limit; limit != nil {
		get = func(key string) *uint256.Int {
			shared := limit.GetLimit(key)
			var x uint256.Int
			if shared == nil || shared.Sign() < 0 || x.SetFromBig(shared) {
				return nil
			}
			return &x
		}
	}
	res, err := p.calcAmtOut(param.TokenAmountIn.Token, param.TokenOut, &amountIn, get)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
	}, nil
}

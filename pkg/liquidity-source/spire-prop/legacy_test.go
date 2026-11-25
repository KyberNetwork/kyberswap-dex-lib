package spireprop

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrAmount
	}
	amount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrAmount
	}
	q, err := s.quote(params.TokenAmountIn.Token, params.TokenOut, amount)
	if err != nil {
		return nil, err
	}
	if params.Limit != nil {
		limit := params.Limit.GetLimit(s.limitKey(s.Info.Tokens[q.out]))
		if limit == nil || q.output.ToBig().Cmp(limit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[q.out], Amount: q.output.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[q.in], Amount: new(big.Int)},
		Gas:            q.gas,
		SwapInfo:       q.swapInfo,
	}, nil
}

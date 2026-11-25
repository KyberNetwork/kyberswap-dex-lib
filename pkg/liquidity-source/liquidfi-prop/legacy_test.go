package liquidfiprop

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrZeroAmount
	}
	q, err := s.quote(in, out, amountIn)
	if err != nil {
		return nil, err
	}
	amountOutBI := q.amountOut.ToBig()
	if params.Limit != nil {
		if limit := params.Limit.GetLimit(s.limitKey(s.Info.Tokens[out])); limit == nil || amountOutBI.Cmp(limit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[out], Amount: amountOutBI},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[in], Amount: new(big.Int)},
		Gas:            defaultGas,
		SwapInfo:       q.swapInfo,
	}, nil
}

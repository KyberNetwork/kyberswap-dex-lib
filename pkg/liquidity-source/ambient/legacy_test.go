package ambient

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	q, err := p.quoteSwap(params.TokenAmountIn.Token, params.TokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if inventoryLimit := limit.GetLimit(p.ambientToken(q.tokenOutIndex)); inventoryLimit != nil &&
			q.amountOut.ToBig().Cmp(inventoryLimit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: q.amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: bignumber.ZeroBI,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: q.remainingIn.ToBig(),
		},
		Gas:      q.gas,
		SwapInfo: SwapInfo{nextCurve: q.nextCurve},
	}, nil
}

package ghost

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	amountInBig := param.TokenAmountIn.Amount
	if amountInBig == nil || amountInBig.Sign() <= 0 {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(amountInBig)
	if overflow {
		return nil, ErrOverflow
	}

	amountOut, fee, principal, err := p.quote(tokenIn, tokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	// Multiple ghost pools can share the same underlying targetRouter/token vault balance, so
	// param.Limit (aggregated across all ghost pools by the pathfinder) guards against
	// overselling that shared inventory within a single route build.
	if limit := param.Limit; limit != nil {
		if inventoryLimit := limit.GetLimit(tokenOut); inventoryLimit != nil && amountOut.ToBig().Cmp(inventoryLimit) > 0 {
			return nil, ErrInsufficientLiquidity
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: fee.ToBig(),
		},
		Gas:      p.gas,
		SwapInfo: SwapInfo{TotalFeeBps: totalFeeBps(principal, fee), Fee: *fee},
	}, nil
}

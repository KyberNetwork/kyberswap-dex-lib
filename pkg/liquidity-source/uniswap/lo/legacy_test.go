package uniswaplo

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	res, err := p.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{
			Token:     param.TokenAmountIn.Token,
			Amount:    *number.SetFromBig(param.TokenAmountIn.Amount),
			AmountUsd: param.TokenAmountIn.AmountUsd,
		},
		TokenOut: param.TokenOut,
	})
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  res.RemainingTokenAmountIn.Token,
			Amount: res.RemainingTokenAmountIn.Amount.ToBig(),
		},
	}, nil
}

package machima

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

// CalcAmountOut is the legacy big.Int entry point. It converts to uint256, quotes with CalcAmtOut, and
// converts back, so the tax rules live only in CalcAmtOut.
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount == nil || param.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrZeroAmount
	}
	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(param.TokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}

	res, err := p.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: param.TokenAmountIn.Token, Amount: amountIn},
		TokenOut:      param.TokenOut,
	})
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token: res.RemainingTokenAmountIn.Token, Amount: res.RemainingTokenAmountIn.Amount.ToBig(),
		},
		Fee: &pool.TokenAmount{Token: res.Fee.Token, Amount: res.Fee.Amount.ToBig()},
		Gas: res.Gas,
		// SwapInfo is the uint256 form that UpdateState reads.
		SwapInfo: res.SwapInfo,
	}, nil
}

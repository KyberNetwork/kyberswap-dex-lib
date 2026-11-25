package integral

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !p.isEnabled {
		return nil, ErrTR05
	}
	if params.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	tokenOut := params.TokenOut
	maxAmountOut := p.maxAmountOut(tokenOut, number.SetFromBig(params.Limit.GetLimit(tokenOut)))

	amountOut, fee, err := p.quote(params.TokenAmountIn.Token, tokenOut, number.SetFromBig(params.TokenAmountIn.Amount), maxAmountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			RelayerAddress: p.relayerAddress,
		},
	}, nil
}

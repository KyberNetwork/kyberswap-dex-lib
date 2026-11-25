package synthetix

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (*pool.CalcAmountOutResult, error) {
	if param.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	res, err := p.calcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{
			Token:     param.TokenAmountIn.Token,
			Amount:    *uint256.MustFromBig(param.TokenAmountIn.Amount),
			AmountUsd: param.TokenAmountIn.AmountUsd,
		},
		TokenOut: param.TokenOut,
	}, func(key string) *uint256.Int { return toU256(param.Limit.GetLimit(key)) })
	if res == nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Gas:            res.Gas,
	}, err
}
func toU256(b *big.Int) *uint256.Int {
	if b == nil {
		return nil
	}
	return uint256.MustFromBig(b)
}

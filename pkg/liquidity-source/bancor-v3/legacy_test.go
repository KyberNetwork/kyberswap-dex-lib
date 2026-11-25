package bancorv3

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	sourceAmount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	res, err := s.CalcAmtOut(poolpkg.CalcAmtOutParams{
		TokenAmountIn: poolpkg.TokenAmt{Token: params.TokenAmountIn.Token, Amount: *sourceAmount, AmountUsd: params.TokenAmountIn.AmountUsd},
		TokenOut:      params.TokenOut,
	})
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
	}, nil
}

package gsm4626

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	res, err := s.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: params.TokenAmountIn.Token, Amount: *amountIn, AmountUsd: params.TokenAmountIn.AmountUsd},
		TokenOut:      params.TokenOut,
	})
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
	}, nil
}

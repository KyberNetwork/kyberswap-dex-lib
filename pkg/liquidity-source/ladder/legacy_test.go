package ladder

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountInF, _ := params.TokenAmountIn.Amount.Float64()
	amountOut, err := s.quote(params.TokenAmountIn.Token, params.TokenOut, amountInF)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if l := limit.GetLimit(params.TokenOut); l == nil || amountOut.ToBig().Cmp(l) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignum.ZeroBI},
		Gas:            s.Gas,
	}, nil
}

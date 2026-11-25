package tessera

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenOut := strings.ToLower(params.TokenOut)
	amountOut, err := s.quote(params.TokenAmountIn.Token, tokenOut, uint256.MustFromBig(params.TokenAmountIn.Amount))
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if amountOut.CmpBig(inventoryLimit) > 0 {
			return nil, ErrSwapReverted
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: defaultGas,
	}, nil
}

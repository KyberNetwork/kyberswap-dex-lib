package weeth

import (
	"errors"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidAmountIn = errors.New("invalid amountIn")
)

type PoolSimulator struct {
	poolpkg.Pool

	totalShares      *big.Int
	totalPooledEther *big.Int

	gas Gas
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if param.TokenAmountIn.Token == s.Info.Tokens[0] {
		return &poolpkg.CalcAmountOutResult{
			TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: s.shareForAmount(param.TokenAmountIn.Amount)},
			Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
			Gas:            s.gas.Wrap,
		}, nil
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: s.amountForShare(param.TokenAmountIn.Amount)},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Unwrap,
	}, nil
}

func (s *PoolSimulator) shareForAmount(eETHAmount *big.Int) *big.Int {
	if s.totalPooledEther.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Div(new(big.Int).Mul(eETHAmount, s.totalShares), s.totalPooledEther)
}

func (s *PoolSimulator) amountForShare(weETHAmount *big.Int) *big.Int {
	if s.totalShares.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Div(new(big.Int).Mul(weETHAmount, s.totalPooledEther), s.totalShares)
}

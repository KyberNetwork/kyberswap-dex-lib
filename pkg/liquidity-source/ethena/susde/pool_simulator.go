package susde

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		totalAssets *uint256.Int
		totalSupply *uint256.Int
	}

	Gas struct {
		Deposit int64
	}

	PoolMetaInfo struct {
		BlockNumber uint64 `json:"blockNumber"`
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))
	)

	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		tokens[idx] = entityPool.Tokens[idx].Address
		reserves[idx] = bignumber.NewBig10(entityPool.Reserves[idx])
	}

	if len(tokens) != 2 && len(reserves) != 2 {
		return nil, ErrInvalidToken
	}

	poolInfo := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: uint64(entityPool.BlockNumber),
	}

	totalAssets, overflow := uint256.FromBig(reserves[0])
	if overflow {
		return nil, ErrOverflow
	}

	totalSupply, overflow := uint256.FromBig(reserves[1])
	if overflow {
		return nil, ErrOverflow
	}

	return &PoolSimulator{
		Pool:        pool.Pool{Info: poolInfo},
		totalAssets: totalAssets,
		totalSupply: totalSupply,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	if err := s.validate(tokenAmountIn.Token, tokenOut); err != nil {
		return nil, err
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	// calculate shares
	shares, _ := new(uint256.Int).MulDivOverflow(
		amountIn,
		new(uint256.Int).Add(s.totalSupply, number.Number_1),
		new(uint256.Int).Add(s.totalAssets, number.Number_1),
	)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: shares.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		},
		Gas: defaultGas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	shares, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}
	s.totalSupply.Add(s.totalSupply, shares)

	assets, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	s.totalAssets.Add(s.totalAssets, assets)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return PoolMetaInfo{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) validate(tokenIn string, tokenOut string) error {
	if tokenIn != s.Info.Tokens[0] || tokenOut != s.Info.Tokens[1] {
		return ErrInvalidToken
	}

	return nil
}

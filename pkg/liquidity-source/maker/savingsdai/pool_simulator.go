package savingsdai

import (
	"encoding/json"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		now *uint256.Int
		dsr *uint256.Int
		rho *uint256.Int
		chi *uint256.Int
	}

	SwapInfo struct {
		chi *uint256.Int
	}

	PoolMetaInfo struct {
		BlockNumber uint64 `json:"blockNumber"`
	}
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	tokens := lo.Map(entityPool.Tokens, func(token *entity.PoolToken, _ int) string {
		return token.Address
	})

	reserves := lo.Map(entityPool.Reserves, func(reserve string, _ int) *big.Int {
		return bignumber.NewBig10(reserve)
	})

	if len(tokens) != 2 && len(reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	poolInfo := poolpkg.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: uint64(entityPool.BlockNumber),
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolInfo},
		now:  extra.BlockTimestamp,
		dsr:  extra.DSR,
		rho:  extra.RHO,
		chi:  extra.CHI,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	if err := s.validate(tokenAmountIn.Token, tokenOut); err != nil {
		return nil, err
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	chi, err := s._chi()
	if err != nil {
		return nil, err
	}

	amountOut := lo.Ternary(
		tokenAmountIn.Token == dai,
		s.deposit(amountIn, chi),
		s.redeem(amountIn, chi),
	)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		},
		SwapInfo: SwapInfo{
			chi: chi,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	s.chi = swapInfo.chi
	s.rho = s.now
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return PoolMetaInfo{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) deposit(assets, chi *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Mul(assets, ray),
		chi,
	)
}

func (s *PoolSimulator) redeem(assets, chi *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Mul(assets, chi),
		ray,
	)
}

func (s *PoolSimulator) _chi() (*uint256.Int, error) {
	if s.now.Gt(s.rho) {
		return s.drip()
	}
	return s.chi, nil
}

func (s *PoolSimulator) drip() (*uint256.Int, error) {
	x, err := rpow(s.dsr, new(uint256.Int).Sub(s.now, s.rho), one)
	if err != nil {
		return nil, err
	}

	tmp, err := rmul(x, s.chi)
	if err != nil {
		return nil, err
	}

	return tmp, nil
}

func (s *PoolSimulator) validate(tokenIn, tokenOut string) error {
	if tokenIn == tokenOut {
		return ErrInvalidToken
	}
	inIdx, outIdx := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if inIdx < 0 || outIdx < 0 {
		return ErrInvalidToken
	}
	return nil
}

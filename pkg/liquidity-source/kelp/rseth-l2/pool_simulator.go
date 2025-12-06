package rsethl2

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
			Reserves:    lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignum.NewBig(e) }),
			BlockNumber: p.BlockNumber,
		}},
		Extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = params.TokenAmountIn
		tokenOut      = params.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn >= len(s.Info.Tokens)-1 || indexOut != len(s.Info.Tokens)-1 {
		return nil, ErrInvalidToken
	}

	amountAfterFee, rsETHAmount := new(big.Int), new(big.Int)
	amountAfterFee.Sub(
		params.TokenAmountIn.Amount,
		amountAfterFee.Mul(
			params.TokenAmountIn.Amount, s.Extra.Fee,
		).Div(amountAfterFee, BasisPoint),
	)
	rsETHAmount.Mul(amountAfterFee,
		lo.TernaryF(indexIn == len(s.Info.Tokens)-2,
			func() *big.Int { return ONE },                                  // WETH
			func() *big.Int { return s.Extra.SupportedTokenRates[indexIn] }, // supported tokens
		),
	).Div(
		rsETHAmount, s.RSETHRate,
	)
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: rsETHAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: s.Fee,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolExtra{
		TokenInIsNative:  strings.EqualFold(tokenIn, s.Info.Tokens[len(s.Info.Tokens)-2]),
		TokenOutIsNative: strings.EqualFold(tokenIn, s.Info.Tokens[len(s.Info.Tokens)-2]),
	}
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	index := s.GetTokenIndex(address)
	if index < 0 || index == len(s.Info.Tokens)-1 || len(s.Info.Tokens) < 2 ||
		(!s.NativeEnabled && index == len(s.Info.Tokens)-2) {
		return nil
	}
	return []string{s.Info.Tokens[len(s.Info.Tokens)-1]}
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	index := s.GetTokenIndex(address)
	if index < 0 || index != len(s.Info.Tokens)-1 {
		return nil
	}

	return s.Info.Tokens[:len(s.Info.Tokens)-lo.Ternary(s.NativeEnabled, 1, 2)]
}

package arberaden

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	tokens []*entity.PoolToken
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
		tokens: p.Tokens,
		Extra:  extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = params.TokenAmountIn
		tokenOut      = params.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || len(s.tokens) != 2 {
		return nil, ErrInvalidToken
	}
	amtIn := uint256.MustFromBig(tokenAmountIn.Amount)
	var amtOut, fee *uint256.Int
	var err error
	if indexIn == 0 {
		amtOut, fee, err = s.Debond(indexIn, indexOut, amtIn)
	} else {
		amtOut, fee, err = s.Bond(indexIn, indexOut, amtIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amtOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee.ToBig(),
		},
	}, nil
}

func (s *PoolSimulator) Debond(indexIn int, indexOut int, amtIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	_, tokenIdx, found := lo.FindIndexOf(s.Assets, func(asset Asset) bool { return asset.Token == s.tokens[indexOut].Address })
	if !found {
		return nil, nil, ErrTokenNotExist
	}
	upperBound := new(uint256.Int)
	upperBound.Mul(s.Supply, u98).Div(upperBound, u100)
	isLastOut := amtIn.Cmp(upperBound) >= 0
	amountAfterFee := lo.TernaryF(isLastOut, func() *uint256.Int {
		return amtIn
	}, func() *uint256.Int {
		amt := new(uint256.Int)
		amt.Sub(DEN, s.Fee.Debond).Mul(amt, amtIn).Div(amt, DEN)
		return amt
	})
	percAfterFeeX96, debondAmount := new(uint256.Int), new(uint256.Int)
	percAfterFeeX96.Mul(amountAfterFee, Q96).Div(percAfterFeeX96, s.Supply)
	debondAmount.Mul(s.AssetSupplies[tokenIdx], percAfterFeeX96).Div(debondAmount, Q96)
	return debondAmount, new(uint256.Int).Sub(amtIn, amountAfterFee), nil
}

func (s *PoolSimulator) Bond(indexIn int, indexOut int, amtIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	_, tokenIdx, found := lo.FindIndexOf(s.Assets, func(asset Asset) bool { return asset.Token == s.tokens[indexIn].Address })
	if !found {
		return nil, nil, ErrTokenNotExist
	}
	tokenCurSupply := s.AssetSupplies[tokenIdx]
	firstIn := tokenCurSupply.Sign() == 0
	tokenAmtSupplyRatioX96 := lo.TernaryF(firstIn, func() *uint256.Int {
		return Q96
	}, func() *uint256.Int {
		ratio := new(uint256.Int)
		return ratio.Mul(amtIn, Q96).Div(ratio, tokenCurSupply)
	})
	tokensMinted := new(uint256.Int)
	_ = lo.TernaryF(firstIn, func() *uint256.Int {
		return tokensMinted.Mul(amtIn, Q96).Mul(tokensMinted, big256.TenPow(s.tokens[indexOut].Decimals)).Div(tokensMinted, s.Assets[tokenIdx].Q1)
	}, func() *uint256.Int {
		return tokensMinted.Mul(s.Supply, tokenAmtSupplyRatioX96).Div(tokensMinted, Q96)
	})
	feeTokens := new(uint256.Int)
	feeTokens.Mul(tokensMinted, s.Fee.Bond).Div(feeTokens, DEN)
	return tokensMinted.Sub(tokensMinted, feeTokens), feeTokens, nil
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	feeBurned, newSupply, newAssetBalance := new(uint256.Int), new(uint256.Int), new(uint256.Int)
	// (fee * fees.burn) / DEN
	feeBurned.Mul(uint256.MustFromBig(params.Fee.Amount), s.Fee.Burn).Div(feeBurned, DEN)
	if indexIn == 0 { // debond - brToken -> token
		// burn amountIn after fee + burn part of fee
		s.Supply = newSupply.
			Sub(s.Supply, uint256.MustFromBig(params.TokenAmountIn.Amount)).
			Add(newSupply, uint256.MustFromBig(params.Fee.Amount)).
			Sub(newSupply, feeBurned)

		// decrease balance of token out
		s.AssetSupplies[indexOut] = newAssetBalance.
			Sub(s.AssetSupplies[indexOut], uint256.MustFromBig(params.TokenAmountOut.Amount))
	} else { // bond - token -> brToken
		// mint amountOut before fee + burn part of fee
		s.Supply = newSupply.
			Add(s.Supply, uint256.MustFromBig(params.TokenAmountOut.Amount)).
			Add(newSupply, uint256.MustFromBig(params.Fee.Amount)).
			Sub(newSupply, feeBurned)
		// increase balance of token in
		s.AssetSupplies[indexIn] = newAssetBalance.
			Sub(s.AssetSupplies[indexIn], uint256.MustFromBig(params.TokenAmountIn.Amount))
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Supply = new(uint256.Int).Set(s.Supply)
	return &cloned
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(address, s.Info.Tokens[0]) {
		result := make([]string, 0, len(s.Info.Tokens)-1)
		result = append(result, s.Info.Tokens[1:]...)
		return result
	}
	return []string{s.Info.Tokens[0]}
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	return p.CanSwapFrom(address)
}

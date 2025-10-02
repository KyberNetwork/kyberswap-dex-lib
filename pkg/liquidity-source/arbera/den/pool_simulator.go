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
	extra  Extra
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
		extra:  extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = params.TokenAmountIn
		tokenOut      = params.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
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
	_, tokenIdx, found := lo.FindIndexOf(s.extra.Assets, func(asset Asset) bool { return asset.Token == s.tokens[indexOut].Address })
	if !found {
		return nil, nil, ErrTokenNotExist
	}
	upperBound := new(uint256.Int)
	upperBound.Mul(s.extra.Supply, u98).Div(upperBound, u100)
	isLastOut := amtIn.Cmp(upperBound) >= 0
	amountAfterFee := lo.TernaryF(isLastOut, func() *uint256.Int {
		return amtIn
	}, func() *uint256.Int {
		amt := new(uint256.Int)
		amt.Sub(DEN, s.extra.Fee.Debond).Mul(amt, amtIn).Div(amt, DEN)
		return amt
	})
	percAfterFeeX96, debondAmount := new(uint256.Int), new(uint256.Int)
	percAfterFeeX96.Mul(amountAfterFee, Q96).Div(percAfterFeeX96, s.extra.Supply)
	debondAmount.Mul(s.extra.AssetSupplies[tokenIdx], percAfterFeeX96).Div(debondAmount, Q96)
	return debondAmount, new(uint256.Int).Sub(amtIn, amountAfterFee), nil
}

func (s *PoolSimulator) Bond(indexIn int, indexOut int, amtIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	_, tokenIdx, found := lo.FindIndexOf(s.extra.Assets, func(asset Asset) bool { return asset.Token == s.tokens[indexIn].Address })
	if !found {
		return nil, nil, ErrTokenNotExist
	}
	tokenCurSupply := s.extra.AssetSupplies[tokenIdx]
	firstIn := tokenCurSupply.Sign() == 0
	tokenAmtSupplyRatioX96 := lo.TernaryF(firstIn, func() *uint256.Int {
		return Q96
	}, func() *uint256.Int {
		ratio := new(uint256.Int)
		return ratio.Mul(amtIn, Q96).Div(ratio, tokenCurSupply)
	})
	tokensMinted := new(uint256.Int)
	_ = lo.TernaryF(firstIn, func() *uint256.Int {
		return tokensMinted.Mul(amtIn, Q96).Mul(tokensMinted, big256.TenPow(s.tokens[indexOut].Decimals)).Div(tokensMinted, s.extra.Assets[tokenIdx].Q1)
	}, func() *uint256.Int {
		return tokensMinted.Mul(s.extra.Supply, tokenAmtSupplyRatioX96).Div(tokensMinted, Q96)
	})
	feeTokens := new(uint256.Int)
	feeTokens.Mul(tokensMinted, s.extra.Fee.Bond).Div(feeTokens, DEN)
	return tokensMinted.Sub(tokensMinted, feeTokens), feeTokens, nil
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	return p
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(address, p.Info.Tokens[0]) {
		result := make([]string, 0, len(p.Info.Tokens)-1)
		result = append(result, p.Info.Tokens[1:]...)
		return result
	}
	return []string{p.Info.Tokens[0]}
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	return p.CanSwapFrom(address)
}

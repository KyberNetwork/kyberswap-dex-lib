package spendle

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	UnstakeNumerator *big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		UnstakeNumerator: big.NewInt(POne.Int64() - int64(extra.InstantUnstakeFeeRate)),
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexOut, amtOut := p.GetTokenIndex(param.TokenOut), param.TokenAmountIn.Amount
	if indexOut == 0 {
		amtOut = bignumber.MulDivDown(new(big.Int), amtOut, p.UnstakeNumerator, POne)
	}
	if amtOut.Cmp(p.GetReserves()[indexOut]) > 0 {
		return nil, ErrInvalidAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amtOut},
		Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gasUnstake,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	indexOut := p.GetTokenIndex(param.TokenAmountOut.Token)
	amount := param.TokenAmountOut.Amount
	if amount.Cmp(p.GetReserves()[indexOut]) > 0 {
		return nil, ErrInvalidAmount
	}
	if indexOut == 0 {
		amount = bignumber.MulDivUp(new(big.Int), amount, POne, p.UnstakeNumerator)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amount},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: bignumber.ZeroBI},
		Gas:           gasUnstake,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {
	tokenAmtIn, tokenAmtOut := param.TokenAmountIn, param.TokenAmountOut
	inIndex, outIndex := p.GetTokenIndex(tokenAmtIn.Token), p.GetTokenIndex(tokenAmtOut.Token)
	p.Info.Reserves = slices.Clone(p.Info.Reserves)
	p.Info.Reserves[inIndex] = new(big.Int).Add(p.Info.Reserves[inIndex], tokenAmtIn.Amount)
	p.Info.Reserves[outIndex] = new(big.Int).Sub(p.Info.Reserves[outIndex], tokenAmtOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

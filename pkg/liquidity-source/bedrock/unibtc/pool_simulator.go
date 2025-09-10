package unibtc

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

	Tokens []*entity.PoolToken
	extra  PoolExtra
	gas    Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Tokens: entityPool.Tokens,
		extra:  extra,
		gas:    defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	idIn, idOut := s.Info.GetTokenIndex(params.TokenAmountIn.Token), s.Info.GetTokenIndex(params.TokenOut)
	if idIn == len(s.Tokens)-1 || idOut != len(s.Tokens)-1 {
		return nil, ErrUnsupportedSwap
	}

	if s.extra.Paused || s.extra.TokensPaused[idIn] || !s.extra.TokensAllowed[idIn] {
		return nil, ErrUnsupportedToken
	}

	if s.extra.TokenUsedCaps[idIn] == nil || params.TokenAmountIn.Amount.Cmp(new(big.Int).Sub(s.extra.Caps[idIn], s.extra.TokenUsedCaps[idIn])) >= 0 {
		return nil, ErrInsufficientCap
	}

	var uniBTCAmt *big.Int
	switch s.Tokens[idIn].Decimals {
	case 8:
		uniBTCAmt = params.TokenAmountIn.Amount
	case 18:
		uniBTCAmt = new(big.Int).Div(params.TokenAmountIn.Amount, s.extra.ExchangeRateBase)
	default:
		uniBTCAmt = bignumber.ZeroBI
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: uniBTCAmt},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Mint,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	idIn := s.Info.GetTokenIndex(params.TokenAmountIn.Token)
	s.extra.TokenUsedCaps[idIn] = new(big.Int).Add(s.extra.TokenUsedCaps[idIn], params.TokenAmountIn.Amount)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.extra.TokenUsedCaps = slices.Clone(s.extra.TokenUsedCaps)
	return &cloned
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	return s.CanSwapFrom(token)
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	id := s.Info.GetTokenIndex(token)
	if id == len(s.Tokens)-1 {
		return s.Info.Tokens[:len(s.Tokens)-1]
	}

	return []string{s.Tokens[len(s.Tokens)-1].Address}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}

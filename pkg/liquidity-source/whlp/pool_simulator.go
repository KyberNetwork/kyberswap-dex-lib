package whlp

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra       Extra
	staticExtra StaticExtra
	oneShare    *big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		extra:       extra,
		staticExtra: staticExtra,
		oneShare:    bignumber.TenPowInt(entityPool.Tokens[0].Decimals),
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if strings.EqualFold(token, s.Info.Tokens[0]) {
		return []string{s.Info.Tokens[1]}
	}
	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if strings.EqualFold(token, s.Info.Tokens[1]) {
		return []string{s.Info.Tokens[0]}
	}
	if strings.EqualFold(token, s.Info.Tokens[0]) {
		return []string{s.Info.Tokens[1]}
	}
	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.IsAccountantPaused {
		return nil, ErrAccountantPaused
	}

	indexIn := s.GetTokenIndex(param.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	var amountOut *big.Int
	var err error

	switch {
	case indexIn == 1 && indexOut == 0:
		amountOut, err = quoteToShare(param.TokenAmountIn.Amount, s.extra.RateInQuote, s.oneShare)
	case indexIn == 0 && indexOut == 1:
		amountOut, err = shareToQuote(param.TokenAmountIn.Amount, s.extra.RateInQuote, s.oneShare)
	default:
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		Depositor:     s.staticExtra.Depositor,
		Accountant:    s.staticExtra.Accountant,
		CommunityCode: communityCode,
	}
}

package whlp

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken      = errors.New("invalid token for swap")
	ErrInvalidRate       = errors.New("invalid rate")
	ErrMinimumMintNotMet = errors.New("minimum mint not met")
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
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		extra:       extra,
		staticExtra: staticExtra,
		oneShare:    bignumber.TenPowInt(entityPool.Tokens[0].Decimals),
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}

	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(param.TokenAmountIn.Token)
	if indexIn != 1 {
		return nil, ErrInvalidToken
	}

	if s.extra.RateInQuote == nil || s.extra.RateInQuote.Sign() <= 0 {
		return nil, ErrInvalidRate
	}

	var shares big.Int
	bignumber.MulDivDown(&shares, param.TokenAmountIn.Amount, s.oneShare, s.extra.RateInQuote)

	if shares.Sign() <= 0 {
		return nil, ErrMinimumMintNotMet
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: &shares},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return MetaInfo{
		Depositor:       s.staticExtra.Depositor,
		Accountant:      s.staticExtra.Accountant,
		CommunityCode:   s.staticExtra.CommunityCode,
		ApprovalAddress: s.staticExtra.Depositor,
	}
}

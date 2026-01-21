package axima

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)

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
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
		}},
		extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {

	amountInF, _ := params.TokenAmountIn.Amount.Float64()
	rate := lo.Ternary(params.TokenAmountIn.Token == s.Info.Tokens[0],
		s.extra.ZeroToOneRate,
		s.extra.OneToZeroRate)

	amountOutF := amountInF * rate
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexOut == -1 {
		return nil, ErrInvalidToken
	}

	if amountOut.Cmp(s.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: bignumber.ZeroBI,
		},
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn != -1 && indexOut != -1 {
		s.Info.Reserves[indexIn].Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
		s.Info.Reserves[indexOut].Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	return &cloned
}

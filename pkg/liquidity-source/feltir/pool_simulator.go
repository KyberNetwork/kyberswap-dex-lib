package feltir

import (
	"math/big"
	"slices"
	"sort"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeFeltir)
)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return strings.ToLower(e.Address) })
	reserves := lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) })

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenOut)

	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if len(s.Samples[indexIn]) == 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn := params.TokenAmountIn.Amount
	samples := s.Samples[indexIn]

	if amountIn.Cmp(samples[len(samples)-1][0]) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	idx := sort.Search(len(samples), func(i int) bool {
		return samples[i][0].Cmp(amountIn) > 0
	})
	sampleIndex := max(idx-1, 0)

	amountOut := new(big.Int)
	bignumber.MulDivDown(amountOut, amountIn, samples[sampleIndex][1], samples[sampleIndex][0])

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	if limit := params.Limit; limit != nil {
		if amountOut.Cmp(limit.GetLimit(tokenOut)) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token,
			params.TokenAmountIn.Token,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = reserves[i]
	}
	return inventory
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.ApprovalInfo{ApprovalAddress: s.FeltirAddress}
}

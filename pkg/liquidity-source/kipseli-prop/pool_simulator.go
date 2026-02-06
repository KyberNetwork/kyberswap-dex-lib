package kipseliprop

import (
	"encoding/json"
	"math/big"
	"slices"
	"sort"
	"strings"

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
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeKipseliProp)
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
	tokenAmountIn := params.TokenAmountIn
	tokenOut := strings.ToLower(params.TokenOut)
	tokenIn := strings.ToLower(tokenAmountIn.Token)

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || len(s.Info.Tokens) != 2 {
		return nil, ErrInvalidToken
	}

	if len(s.Samples[indexIn]) == 0 {
		return nil, ErrInsufficientLiquidity
	}

	samples := s.Samples[indexIn]
	idx := sort.Search(len(samples), func(i int) bool {
		return samples[i][0].Cmp(tokenAmountIn.Amount) > 0
	})
	sampleIndex := max(idx-1, 0)

	var amountOut = new(big.Int)
	bignumber.MulDivDown(amountOut, tokenAmountIn.Amount, samples[sampleIndex][1], samples[sampleIndex][0])

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if amountOut.Cmp(inventoryLimit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: big.NewInt(0)},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
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

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.Samples = make([][][2]*big.Int, len(s.Samples))
	for i, dir := range s.Samples {
		cloned.Samples[i] = make([][2]*big.Int, len(dir))
		for j, pair := range dir {
			cloned.Samples[i][j] = [2]*big.Int{
				new(big.Int).Set(pair[0]),
				new(big.Int).Set(pair[1]),
			}
		}
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.ApprovalInfo{ApprovalAddress: s.RouterAddress}
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = reserves[i]
	}
	return inventory
}

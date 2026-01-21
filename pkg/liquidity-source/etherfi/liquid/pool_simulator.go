package liquid

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
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
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
		extra: extra,
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[0] {
		return s.Info.Tokens[1:]
	}

	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if slices.Contains(s.Info.Tokens[1:], token) {
		return []string{s.Info.Tokens[0]}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// 1:1 exchange ratio (mint & burn).
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: param.TokenAmountIn.Amount},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any { return s.extra }

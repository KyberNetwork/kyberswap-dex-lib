package miromigrator

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	PoolExtra
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
		PoolExtra: extra,
	}, nil
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	vlrTokenId := len(s.Info.Tokens) - 1
	if s.GetTokenIndex(address) == vlrTokenId {
		return []string{}
	}

	return []string{s.Info.Tokens[vlrTokenId]}
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	vlrTokenId := len(s.Info.Tokens) - 1
	if s.GetTokenIndex(address) == vlrTokenId {
		return s.Info.Tokens[:vlrTokenId]
	}

	return []string{}
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.Paused {
		return nil, ErrMigrationIsPaused
	}

	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: params.TokenAmountIn.Amount},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: big.NewInt(0)},
		Gas:            int64(defaultGas),
		SwapInfo: &SwapInfo{
			IsDeposit: lo.Ternary(indexIn == 0, true, false), // if tokenIn is PSP then it is deposit
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	return p
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}

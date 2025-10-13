package arberazap

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	basePools []pool.IPoolSimulator
}

var _ = pool.RegisterFactoryMeta(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
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
		basePools: lo.Map(staticExtra.BasePools, func(basePool string, _ int) pool.IPoolSimulator {
			return basePoolMap[basePool]
		}),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = params.TokenAmountIn
		tokenOut      = params.TokenOut
	)
	if len(s.basePools)+1 != len(s.Info.Tokens) {
		return nil, ErrBasePoolsMismatch
	}
	// basePools: [stLBGT, brLBGT, UniV2, brARBERO, stARBERO]
	// tokens:    [LBGT, stLBGT, brLBGT] <-> [brARBERO, ARBERO, stARBERO]
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 ||
		indexIn <= 2 && indexOut <= 2 || indexIn >= 3 && indexOut >= 3 {
		return nil, ErrInvalidToken
	}

	amountOut := new(big.Int).Set(tokenAmountIn.Amount)
	isBuy := indexIn < indexOut
	ln := len(s.basePools)
	for index := range ln {
		idx := lo.Ternary(isBuy, index, ln-index)
		if lo.Ternary(isBuy,
			indexIn <= idx && indexOut > idx,
			indexIn >= idx && indexOut < idx,
		) {
			// fmt.Println("idx", idx, s.Pool.Info.Tokens[idx], s.Pool.Info.Tokens[idx+lo.Ternary(isBuy, 1, -1)], s.basePools[idx-lo.Ternary(isBuy, 0, 1)].GetAddress())
			currentPool := s.basePools[idx-lo.Ternary(isBuy, 0, 1)]
			if currentPool == nil {
				return nil, ErrBasePoolNotFound
			}
			result, err := currentPool.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  s.Info.Tokens[idx],
					Amount: amountOut,
				},
				TokenOut: s.Info.Tokens[idx+lo.Ternary(isBuy, 1, -1)],
			})
			if err != nil {
				return nil, err
			}
			amountOut = result.TokenAmountOut.Amount
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: big.NewInt(0),
		},
		SwapInfo: SwapInfo{
			IsBuy: isBuy,
		},
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (t *PoolSimulator) SetBasePool(newBasePool pool.IPoolSimulator) {
	_, idx, found := lo.FindIndexOf(t.basePools, func(basePool pool.IPoolSimulator) bool {
		return basePool != nil && newBasePool != nil && strings.EqualFold(basePool.GetAddress(), newBasePool.GetAddress())
	})
	if found && idx >= 0 {
		t.basePools[idx] = newBasePool
	}
}

func (t *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	return t.basePools
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	idx := t.GetTokenIndex(address)
	res := []string{}
	if idx < 0 {
		return res
	}
	if idx <= 2 {
		return append(res, t.Info.Tokens[3:]...)
	}
	return append(res, t.Info.Tokens[:3]...)
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	return t.CanSwapFrom(address)
}

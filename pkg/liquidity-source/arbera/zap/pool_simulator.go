package arberazap

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	arberaden "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/arbera/den"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
			poolId := idx - lo.Ternary(isBuy, 0, 1)
			currentPool := s.basePools[poolId]
			isPrivate := currentPool.GetExchange() == valueobject.ExchangeArberaDenAmm
			if currentPool == nil {
				return nil, ErrBasePoolNotFound
			}

			// There is a burning fee logic when buying/selling den tokens through the private pool
			// The brLBGT-brarBERO private pool handles feeBurn for brarBERO only (although brLBGT has buy/sell fee config too, it is used for other private pools like brNECT-brLBGT)
			// The logic resides inside the token (brLBGT/brarBERO)'s transfer method
			// When selling brarBERO, it occurs during the transfer of brarBERO to the private pool, then adjusts before the swap
			// When buying brarBERO, it occurs during the transfer of brarBERO from the private pool, then adjusts after the swap
			var extra *arberaden.Fee
			var err error

			if isPrivate {
				extra, err = util.AnyToStruct[arberaden.Fee](s.basePools[poolId+lo.Ternary(isBuy, 1, -1)].GetMetaInfo("", ""))
				if err != nil || (isBuy && extra.Buy == nil) || (!isBuy && extra.Sell == nil) {
					return nil, ErrDenBuySellFeeNotFound
				}
			}
			burnedAmount := new(big.Int)
			if isPrivate && !isBuy {
				burnedAmount.Mul(amountOut, extra.Sell.ToBig()).Div(burnedAmount, arberaden.DEN.ToBig())
				amountOut.Sub(amountOut, burnedAmount)
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
			if isPrivate && isBuy {
				burnedAmount.Mul(amountOut, extra.Buy.ToBig()).Div(burnedAmount, arberaden.DEN.ToBig())
				amountOut.Sub(amountOut, burnedAmount)
			}
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

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator { return s }

func (s *PoolSimulator) SetBasePool(newBasePool pool.IPoolSimulator) {
	_, idx, found := lo.FindIndexOf(s.basePools, func(basePool pool.IPoolSimulator) bool {
		return basePool != nil && newBasePool != nil && strings.EqualFold(basePool.GetAddress(), newBasePool.GetAddress())
	})
	if found && idx >= 0 {
		s.basePools[idx] = newBasePool
	}
}

func (s *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	return s.basePools
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	idx := s.GetTokenIndex(address)
	res := []string{}
	if idx < 0 {
		return res
	}
	if idx <= 2 {
		return append(res, s.Info.Tokens[3:]...)
	}
	return append(res, s.Info.Tokens[:3]...)
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	return s.CanSwapFrom(address)
}

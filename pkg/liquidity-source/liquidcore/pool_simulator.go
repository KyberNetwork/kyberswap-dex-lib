package liquidcore

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	*PoolState
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},

		PoolState: &PoolState{
			Token0:      ep.Tokens[0].Address,
			Token1:      ep.Tokens[1].Address,
			Decimals0:   ep.Tokens[0].Decimals,
			Decimals1:   ep.Tokens[1].Decimals,
			Reserve0:    uint256.MustFromDecimal(ep.Reserves[0]),
			Reserve1:    uint256.MustFromDecimal(ep.Reserves[1]),
			SpotPrice:   extra.SpotPrice,
			OraclePrice: extra.OraclePrice,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)

	result, err := CalcSwap(s.PoolState, tokenAmountIn.Token, tokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: result.AmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: result.FeeAmount.ToBig(),
		},
		Gas:      defaultGas,
		SwapInfo: lo.T4(s.SpotPrice, s.OraclePrice, result.NewReserve0, result.NewReserve1),
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	clonedPoolState := *s.PoolState
	clonedPoolState.Reserve0 = new(uint256.Int).Set(s.Reserve0)
	clonedPoolState.Reserve1 = new(uint256.Int).Set(s.Reserve1)
	cloned.PoolState = &clonedPoolState

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	_, _, r0, r1 := params.SwapInfo.(lo.Tuple4[*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int]).Unpack()
	s.Reserve0 = r0.Clone()
	s.Reserve1 = r1.Clone()
}

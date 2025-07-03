package aavev3

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra           Extra
	aavePoolAddress string
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
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		extra:           extra,
		aavePoolAddress: staticExtra.AavePoolAddress,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("invalid token")
	}

	isSupply := indexIn == 1

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: param.TokenAmountIn.Amount},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: integer.Zero()},
		Gas:            lo.Ternary(isSupply, supplyGas, withdrawGas),
		SwapInfo: &SwapInfo{
			IsSupply:        isSupply,
			AavePoolAddress: s.aavePoolAddress,
		},
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	if s.GetTokenIndex(address) == 0 {
		return lo.Ternary(!s.extra.IsActive || s.extra.IsPaused,
			[]string{}, []string{s.Pool.Info.Tokens[1]})
	}

	return lo.Ternary(!s.extra.IsActive || s.extra.IsPaused || s.extra.IsFrozen,
		[]string{}, []string{s.Pool.Info.Tokens[0]})
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if s.GetTokenIndex(address) == 0 {
		return lo.Ternary(!s.extra.IsActive || s.extra.IsPaused || s.extra.IsFrozen,
			[]string{}, []string{s.Pool.Info.Tokens[1]})
	}

	return lo.Ternary(!s.extra.IsActive || s.extra.IsPaused,
		[]string{}, []string{s.Pool.Info.Tokens[0]})
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator { return s }

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

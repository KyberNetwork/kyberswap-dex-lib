package aavev3

import (
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
		return nil, ErrInvalidToken
	}

	isSupply := indexIn == 1

	// Validate swap output is smaller than available liquidity (reserve[1])
	// For withdraw (indexIn == 0, indexOut == 1), output is token[1] which is the asset token
	if !isSupply {
		if param.TokenAmountIn.Amount.Cmp(s.Info.Reserves[1]) > 0 {
			return nil, ErrSwapOutputExceedsLiquidity
		}
	} else if param.TokenAmountIn.Amount.Cmp(s.Info.Reserves[0]) > 0 {
		// Validate swap input does not exceed supply cap (reserve[0])
		// For supply (indexIn == 1, indexOut == 0), input is token[1] which is the asset token
		return nil, ErrSwapInputExceedsSupplyCap
	}

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

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	indexIn, indexOut := s.GetTokenIndex(param.TokenIn), s.GetTokenIndex(param.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	isSupply := indexIn == 1

	// Validate swap output is smaller than available liquidity (reserve[1])
	// For withdraw (indexIn == 0, indexOut == 1), output is token[1] which is the asset token
	if !isSupply {
		if param.TokenAmountOut.Amount.Cmp(s.Info.Reserves[1]) > 0 {
			return nil, ErrSwapOutputExceedsLiquidity
		}
	} else if param.TokenAmountOut.Amount.Cmp(s.Info.Reserves[0]) > 0 {
		// Validate swap output does not exceed supply cap (reserve[0])
		// For supply (indexIn == 1, indexOut == 0), output is token[0] which is the aToken
		return nil, ErrSwapInputExceedsSupplyCap
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: param.TokenAmountOut.Amount},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: integer.Zero()},
		Gas:           lo.Ternary(isSupply, supplyGas, withdrawGas),
		SwapInfo: &SwapInfo{
			IsSupply:        isSupply,
			AavePoolAddress: s.aavePoolAddress,
		},
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	if indexIn == 0 {
		// For withdraw (indexIn == 0, indexOut == 1), we're withdrawing from the pool
		// so, reserve[0] (supplyCap) increases by the amount withdrawn
		// and reserve[1] (liquidity) decreases by the amount withdrawn
		s.Info.Reserves = []*big.Int{
			new(big.Int).Add(s.Info.Reserves[0], params.TokenAmountOut.Amount),
			new(big.Int).Sub(s.Info.Reserves[1], params.TokenAmountOut.Amount),
		}
	} else {
		// For supply (indexIn == 1, indexOut == 0), we're supplying to the pool
		// so, reserve[0] (supply cap) decreases by the amount supplied
		// and reserve[1] (liquidity) increases by the amount supplied
		s.Info.Reserves = []*big.Int{
			new(big.Int).Sub(s.Info.Reserves[0], params.TokenAmountOut.Amount),
			new(big.Int).Add(s.Info.Reserves[1], params.TokenAmountOut.Amount),
		}
	}
}

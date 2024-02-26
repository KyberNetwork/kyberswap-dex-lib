package eeth

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var uint128Max = new(big.Int).Sub(
	new(big.Int).Lsh(big.NewInt(1), 128),
	big.NewInt(1),
)

var (
	ErrUnsupportedSwap = errors.New("unsupported swap")
	ErrInvalidAmount   = errors.New("invalid amount")
)

// PoolSimulator only support deposits ETH and get eETH
type PoolSimulator struct {
	poolpkg.Pool

	totalPooledEther *big.Int
	totalShares      *big.Int

	gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			BlockNumber: entityPool.BlockNumber,
		}},
		totalPooledEther: extra.TotalPooledEther,
		totalShares:      extra.TotalShares,
		gas:              defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	// NOTE: only support tokenIn is WETH and tokenOut is eETH
	if param.TokenAmountIn.Token != s.Pool.Info.Tokens[0] || param.TokenOut != s.Pool.Info.Tokens[1] {
		return nil, ErrUnsupportedSwap
	}

	amount := new(big.Int).Set(param.TokenAmountIn.Amount)
	share := s.sharesForDepositAmount(amount)

	if amount.Cmp(uint128Max) > 0 || amount.Cmp(bignumber.ZeroBI) == 0 || share.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrInvalidAmount
	}

	amountOut := s.amountForShare(share, amount)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[1], Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {
	s.totalPooledEther.Add(s.totalPooledEther, param.TokenAmountIn.Amount)
	s.totalShares.Add(s.totalShares, param.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) sharesForDepositAmount(depositAmount *big.Int) *big.Int {
	if s.totalPooledEther.Cmp(bignumber.ZeroBI) == 0 {
		return depositAmount
	}

	return new(big.Int).Div(
		new(big.Int).Mul(depositAmount, s.totalShares),
		s.totalPooledEther,
	)
}

func (s *PoolSimulator) amountForShare(share *big.Int, depositAmount *big.Int) *big.Int {
	totalShares := new(big.Int).Add(s.totalShares, share)

	return new(big.Int).Div(new(big.Int).Mul(share, new(big.Int).Add(s.totalPooledEther, depositAmount)), totalShares)
}

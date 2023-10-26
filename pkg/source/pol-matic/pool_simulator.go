package polmatic

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		gas Gas
	}

	Gas struct {
		Migrate   int64
		Unmigrate int64
	}

	SwapInfo struct {
		// IsMigrate is true when tokenIn is Matic
		IsMigrate bool `json:"isMigrate"`
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	return &PoolSimulator{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			},
		},
		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	var (
		isMigrate bool
		gas       int64
	)
	if tokenAmountIn.Token == s.Pool.Info.Tokens[0] {
		if tokenAmountIn.Amount.Cmp(s.Pool.Info.Reserves[1]) > 0 {
			return nil, ErrInsufficientLiquidity
		}

		isMigrate = true
		gas = s.gas.Migrate
	} else {
		if tokenAmountIn.Amount.Cmp(s.Pool.Info.Reserves[0]) > 0 {
			return nil, ErrInsufficientLiquidity
		}

		isMigrate = false
		gas = s.gas.Unmigrate
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: tokenOut, Amount: tokenAmountIn.Amount},
		Fee:            &poolpkg.TokenAmount{Token: tokenOut, Amount: integer.Zero()},
		Gas:            gas,
		SwapInfo:       SwapInfo{IsMigrate: isMigrate},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

package unieth

import (
	"errors"
	"math/big"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPaused          = errors.New("paused")
	ErrUnsupportedSwap = errors.New("unsupported swap")
)

type PoolSimulator struct {
	poolpkg.Pool

	paused         bool
	totalSupply    *big.Int
	currentReserve *big.Int

	gas Gas
}

type Gas struct {
	Mint int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := sonic.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		paused:         extra.Paused,
		totalSupply:    extra.TotalSupply,
		currentReserve: extra.CurrentReserve,
		gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPaused
	}

	// NOTE: only allow to mint uniETH from ETH, so tokenIn has to be WETH and tokenOut has to be uniETH
	if params.TokenAmountIn.Token != s.Info.Tokens[0] && params.TokenOut != s.Info.Tokens[1] {
		return nil, ErrUnsupportedSwap
	}

	amountOut := new(big.Int).Set(params.TokenAmountIn.Amount) // default exchange ratio 1:1

	if s.currentReserve.Cmp(bignumber.ZeroBI) > 0 { // avert division overflow
		amountOut.Div(
			new(big.Int).Mul(s.totalSupply, params.TokenAmountIn.Amount),
			s.currentReserve,
		)
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Mint,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	s.currentReserve = new(big.Int).Add(s.currentReserve, params.TokenAmountIn.Amount)
	s.totalSupply = new(big.Int).Add(s.totalSupply, params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == UNIETH {
		return []string{WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == WETH {
		return []string{UNIETH}
	}
	return []string{}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

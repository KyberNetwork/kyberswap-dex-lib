package sweth

import (
	"errors"
	"math/big"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrUnsupportedSwap = errors.New("unsupported swap")
	ErrPaused          = errors.New("paused")
)

// PoolSimulator only support deposits ETH and get eETH
type PoolSimulator struct {
	poolpkg.Pool

	paused bool

	swETHToETHRate *big.Int

	gas Gas
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
		swETHToETHRate: extra.SWETHToETHRate,
		gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	// NOTE: only support tokenIn is WETH and tokenOut is swETH
	if param.TokenAmountIn.Token != s.Pool.Info.Tokens[0] || param.TokenOut != s.Pool.Info.Tokens[1] {
		return nil, ErrUnsupportedSwap
	}

	if s.paused {
		return nil, ErrPaused
	}

	amountOut := new(big.Int).Div(
		new(big.Int).Mul(param.TokenAmountIn.Amount, bignumber.BONE),
		s.swETHToETHRate,
	)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[1], Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == common.SWETH {
		return []string{common.WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == common.WETH {
		return []string{common.SWETH}
	}
	return []string{}
}

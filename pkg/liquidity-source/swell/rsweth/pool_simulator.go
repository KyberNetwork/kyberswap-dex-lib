package rsweth

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator only support deposits ETH and get eETH
type PoolSimulator struct {
	pool.Pool

	paused bool

	ethToRswETHRate *big.Int

	gas Gas
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
		paused:          extra.Paused,
		ethToRswETHRate: extra.ETHToRswETHRate,
		gas:             defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// NOTE: only support tokenIn is WETH and tokenOut is rswETH
	if param.TokenAmountIn.Token != s.Pool.Info.Tokens[0] || param.TokenOut != s.Pool.Info.Tokens[1] {
		return nil, ErrUnsupportedSwap
	}

	if s.paused {
		return nil, ErrPaused
	}

	amountOut := new(big.Int).Div(
		new(big.Int).Mul(param.TokenAmountIn.Amount, s.ethToRswETHRate),
		bignumber.BONE,
	)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[1], Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == common.RSWETH {
		return []string{common.WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == common.WETH {
		return []string{common.RSWETH}
	}
	return []string{}
}

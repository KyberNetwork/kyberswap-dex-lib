package vaultT1

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/samber/lo"
)

var (
	ErrInvalidAmountIn = errors.New("invalid amountIn")
)

type PoolSimulator struct {
	poolpkg.Pool

	Ratio *big.Int
}

var (
	defaultGas = Gas{Liquidate: 250000}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
			SwapFee:     big.NewInt(0), // no swap fee on liquidations
		}},
		Ratio: extra.Ratio,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountIn
	}

	tokenAmountOut := new(big.Int).Mul(param.TokenAmountIn.Amount, s.Ratio)

	// ratio is scaled in 1e27, so divide by 1e27
	divisor1e27 := new(big.Int)
	divisor1e27.SetString(String1e27, 10) // 1e27

	tokenAmountOut = new(big.Int).Div(tokenAmountOut, divisor1e27)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            defaultGas.Liquidate,
	}, nil
}

func (t *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount

	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

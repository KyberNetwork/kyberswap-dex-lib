package dexT1

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
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")
)

type PoolSimulator struct {
	poolpkg.Pool

	CollateralReserves CollateralReserves
	DebtReserves       DebtReserves
}

var (
	defaultGas = Gas{Swap: 160000}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	fee := new(big.Int)
	fee.SetInt64(int64(entityPool.SwapFee * 10000))

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
			SwapFee:     fee,
		}},
		CollateralReserves: extra.CollateralReserves,
		DebtReserves:       extra.DebtReserves,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountIn
	}

	swap0To1 := param.TokenAmountIn.Token == s.Info.Tokens[0]

	// fee is applied on token in
	fee := new(big.Int).Mul(param.TokenAmountIn.Amount, s.Pool.Info.SwapFee)
	fee = new(big.Int).Div(fee, big.NewInt(Fee100PercentPrecision))

	amountInAfterFee := new(big.Int).Sub(param.TokenAmountIn.Amount, fee)

	_, tokenAmountOut := swapIn(swap0To1, amountInAfterFee, s.CollateralReserves, s.DebtReserves)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee},
		Gas:            defaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if param.TokenAmountOut.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountOut
	}

	swap0To1 := param.TokenAmountOut.Token == s.Info.Tokens[1]

	tokenAmountIn, _ := swapOut(swap0To1, param.TokenAmountOut.Amount, s.CollateralReserves, s.DebtReserves)

	// fee is applied on token in
	fee := new(big.Int).Mul(tokenAmountIn, s.Pool.Info.SwapFee)
	fee = new(big.Int).Div(fee, big.NewInt(Fee100PercentPrecision))

	amountInAfterFee := new(big.Int).Add(tokenAmountIn, fee)

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{Token: param.TokenIn, Amount: amountInAfterFee},
		Fee:           &poolpkg.TokenAmount{Token: param.TokenIn, Amount: fee},
		Gas:           defaultGas.Swap,
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

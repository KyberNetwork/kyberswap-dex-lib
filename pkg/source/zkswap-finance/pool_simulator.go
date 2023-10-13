package zkswapfinance

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		swapFee  = new(big.Int).SetUint64(uint64(entityPool.SwapFee))
		tokens   = make([]string, 2)
		weights  = make([]uint, 2)
		reserves = make([]*big.Int, 2)
	)

	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		weights[0] = defaultTokenWeight
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])

		tokens[1] = entityPool.Tokens[1].Address
		weights[1] = defaultTokenWeight
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: info},
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokenInIdx := t.Info.GetTokenIndex(tokenAmountIn.Token)
	if tokenInIdx < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("invalid token in: %s", tokenAmountIn.Token)
	}
	tokenOutIdx := t.Info.GetTokenIndex(tokenOut)
	if tokenOutIdx < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("invalid token out: %s", tokenOut)
	}

	amountOut, err := t.getAmountOut(tokenAmountIn.Token, tokenAmountIn.Amount)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("invalid amount out: %v", amountOut.String())
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: nil,
		},
		Gas: defaultGas,
	}, nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amountIn := new(big.Int).Div(
		new(big.Int).Mul(params.TokenAmountIn.Amount, new(big.Int).Sub(feePrecision, t.Info.SwapFee)),
		feePrecision,
	)
	amountOut := params.TokenAmountOut.Amount
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == params.TokenAmountIn.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], amountIn)
		}
		if t.Info.Tokens[i] == params.TokenAmountOut.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], amountOut)
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	swapFee := uint64(defaultSwapFee)
	if t.GetInfo().SwapFee != nil {
		swapFee = t.GetInfo().SwapFee.Uint64()
	}
	return Meta{
		SwapFee:      swapFee,
		FeePrecision: feePrecision.Uint64(),
	}
}

func (t *PoolSimulator) getAmountOut(tokenIn string, amountIn *big.Int) (*big.Int, error) {
	var (
		reserveIn  *big.Int
		reserveOut *big.Int
	)

	if strings.EqualFold(tokenIn, t.Info.Tokens[0]) {
		reserveIn, reserveOut = t.Info.Reserves[0], t.Info.Reserves[1]
	} else {
		reserveIn, reserveOut = t.Info.Reserves[1], t.Info.Reserves[0]
	}

	if amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientInputAmount
	}
	if reserveIn.Cmp(bignumber.ZeroBI) <= 0 || reserveOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	amountInAfterFee := new(big.Int).Mul(amountIn, new(big.Int).Sub(feePrecision, t.Info.SwapFee))
	numerator := new(big.Int).Mul(amountInAfterFee, reserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, feePrecision), amountInAfterFee)
	return new(big.Int).Div(numerator, denominator), nil
}

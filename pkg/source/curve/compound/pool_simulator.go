package compound

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	A           *big.Int
	Multipliers []*big.Int
	AdminFee    *big.Int
	Rates       []*big.Int
	gas         Gas
}

type Gas struct {
	Exchange           int64
	ExchangeUnderlying int64
}

var _ = pool.RegisterFactory0(curve.PoolTypeCompound, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolCompoundStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolCompoundExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = staticExtra.UnderlyingTokens[i]
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		multipliers[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	rates := make([]*big.Int, 0, len(extra.Rates))
	for _, rateStr := range extra.Rates {
		rate, _ := new(big.Int).SetString(rateStr, 10)
		rates = append(rates, rate)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.NewBig10(extra.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Multipliers: multipliers,
		A:           bignumber.NewBig10(extra.A),
		AdminFee:    bignumber.NewBig10(extra.AdminFee),
		Rates:       rates,
		gas:         DefaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)

	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := GetDyUnderlying(
			t.Info.Reserves,
			t.Rates,
			t.Multipliers,
			t.A,
			t.Info.SwapFee,
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if err == nil && amountOut.Cmp(bignumber.ZeroBI) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.gas.ExchangeUnderlying,
			}, nil

		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn := strings.ToLower(param.TokenIn)
	tokenAmountOut := param.TokenAmountOut
	tokenOut := strings.ToLower(tokenAmountOut.Token)
	var tokenIndexFrom = t.GetTokenIndex(tokenIn)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)

	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return &pool.CalcAmountInResult{}, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
	}

	amountIn, fee, err := GetDxUnderlying(
		t.Info.Reserves,
		t.Rates,
		t.Multipliers,
		t.A,
		t.Info.SwapFee,
		tokenIndexFrom,
		tokenIndexTo,
		tokenAmountOut.Amount,
	)
	if err != nil {
		return &pool.CalcAmountInResult{}, err
	}

	if amountIn.Cmp(bignumber.ZeroBI) > 0 {
		return &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: amountIn,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee,
			},
			Gas: t.gas.ExchangeUnderlying,
		}, nil
	}

	return &pool.CalcAmountInResult{}, ErrZero
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
				t.AdminFee,
			),
			FeeDenominator,
		),
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}
func (t *PoolSimulator) GetLpToken() string {
	return ""
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    true,
	}
}

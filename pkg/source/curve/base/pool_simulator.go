package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolBaseSimulator struct {
	pool.Pool
	Multipliers []*big.Int
	Rates       []*big.Int
	// extra fields
	InitialA     *big.Int
	FutureA      *big.Int
	InitialATime int64
	FutureATime  int64
	AdminFee     *big.Int
	LpToken      string
	LpSupply     *big.Int
	APrecision   *big.Int
	Gas          Gas
	NumTokensBI  *big.Int
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolBaseSimulator, error) {
	var staticExtra curve.PoolBaseStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolBaseExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	if entityPool.Reserves == nil || len(entityPool.Reserves) < numTokens {
		return nil, errors.New("empty reserve")
	}

	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	var multipliers = make([]*big.Int, numTokens)
	var rates = make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		multipliers[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
		rates[i] = bignumber.NewBig10(staticExtra.Rates[i])
	}

	var aPrecision = bignumber.One
	if len(staticExtra.APrecision) > 0 {
		aPrecision = bignumber.NewBig10(staticExtra.APrecision)
	}

	return &PoolBaseSimulator{
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
		Multipliers:  multipliers,
		Rates:        rates,
		InitialA:     bignumber.NewBig10(extra.InitialA),
		FutureA:      bignumber.NewBig10(extra.FutureA),
		InitialATime: extra.InitialATime,
		FutureATime:  extra.FutureATime,
		AdminFee:     bignumber.NewBig10(extra.AdminFee),
		LpToken:      staticExtra.LpToken,
		LpSupply:     bignumber.NewBig10(entityPool.Reserves[numTokens]),
		APrecision:   aPrecision,
		Gas:          DefaultGas,
		NumTokensBI:  big.NewInt(int64(numTokens)),
	}, nil
}

func (t *PoolBaseSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := t.GetDy(
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
			nil,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if amountOut.Cmp(bignumber.ZeroBI) > 0 {

			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.Gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolBaseSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (t *PoolBaseSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}

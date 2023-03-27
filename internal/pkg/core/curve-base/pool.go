package curveBase

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

type Pool struct {
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
	gas          Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	var multipliers = make([]*big.Int, numTokens)
	var rates = make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		multipliers[i] = utils.NewBig10(staticExtra.PrecisionMultipliers[i])
		rates[i] = utils.NewBig10(staticExtra.Rates[i])
	}

	var aPrecision = constant.One
	if len(staticExtra.APrecision) > 0 {
		aPrecision = utils.NewBig10(staticExtra.APrecision)
	}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    utils.NewBig10(extra.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Multipliers:  multipliers,
		Rates:        rates,
		InitialA:     utils.NewBig10(extra.InitialA),
		FutureA:      utils.NewBig10(extra.FutureA),
		InitialATime: extra.InitialATime,
		FutureATime:  extra.InitialATime,
		AdminFee:     utils.NewBig10(extra.AdminFee),
		LpToken:      staticExtra.LpToken,
		LpSupply:     utils.NewBig10(entityPool.Reserves[numTokens]),
		APrecision:   aPrecision,
		gas:          DefaultGas,
	}, nil
}

func (t *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := t.GetDy(
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if amountOut.Cmp(constant.Zero) > 0 {

			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (t *Pool) GetLpToken() string {
	return t.LpToken
}

func (t *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 && address != t.LpToken {
		return ret
	}
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	if address != t.LpToken {
		ret = append(ret, t.LpToken)
	}
	return ret
}

func (t *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn *big.Int
	var reserveOut *big.Int
	if tokenIn == t.LpToken {
		reserveIn = t.LpSupply
	} else {
		reserveIn = t.Info.Reserves[tokenInIndex]
	}
	if tokenOut == t.LpToken {
		reserveOut = t.LpSupply
	} else {
		reserveOut = t.Info.Reserves[tokenOutIndex]
	}
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}

func (t *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn *big.Int
	var reserveOut *big.Int
	if tokenIn == t.LpToken {
		reserveIn = t.LpSupply
	} else {
		reserveIn = t.Info.Reserves[tokenInIndex]
	}
	if tokenOut == t.LpToken {
		reserveOut = t.LpSupply
	} else {
		reserveOut = t.Info.Reserves[tokenOutIndex]
	}
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}

func (t *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}

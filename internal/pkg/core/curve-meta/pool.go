package curveMeta

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	errorsPkg "github.com/KyberNetwork/router-service/internal/pkg/core/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

// ICurveBasePool is the interface for curve base pool inside a meta pool
// It can be:
// 1. base/plain pool
// 2. plain oracle pool
// 3. lending pool
// 4. or even meta pool
// At the moment, our code can only support base/plain pool and plain oracle pool
type ICurveBasePool interface {
	GetInfo() pool.PoolInfo
	GetLpToken() string
	GetTokenIndex(address string) int
	GetVirtualPrice() *big.Int
	GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error)
	CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error)
	CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error)
	AddLiquidity(amounts []*big.Int) (*big.Int, error)
	RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error)
}

type Pool struct {
	pool.Pool
	BasePool       ICurveBasePool
	RateMultiplier *big.Int
	InitialA       *big.Int
	FutureA        *big.Int
	InitialATime   int64
	FutureATime    int64
	AdminFee       *big.Int
	LpToken        string
	LpSupply       *big.Int
	APrecision     *big.Int
	gas            Gas
}

func NewPool(entityPool entity.Pool, basePool ICurveBasePool) (*Pool, error) {
	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extraStr Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	rates := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		multipliers[i] = utils.NewBig10(staticExtra.PrecisionMultipliers[i])
		rates[i] = utils.NewBig10(staticExtra.Rates[i])
	}

	aPrecision := constant.One
	if len(staticExtra.APrecision) > 0 {
		aPrecision = utils.NewBig10(staticExtra.APrecision)
	}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    utils.NewBig10(extraStr.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		BasePool:       basePool,
		RateMultiplier: utils.NewBig10(staticExtra.RateMultiplier),
		InitialA:       utils.NewBig10(extraStr.InitialA),
		FutureA:        utils.NewBig10(extraStr.FutureA),
		InitialATime:   extraStr.InitialATime,
		FutureATime:    extraStr.FutureATime,
		AdminFee:       utils.NewBig10(extraStr.AdminFee),
		LpToken:        staticExtra.LpToken,
		LpSupply:       utils.NewBig10(entityPool.Reserves[numTokens]),
		APrecision:     aPrecision,
		gas:            DefaultGas,
	}, nil
}

func (t *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)

	if (tokenIndexFrom == len(t.Info.Tokens)-1 && tokenIndexTo < 0) || (tokenIndexTo == len(t.Info.Tokens)-1 && tokenIndexFrom < 0) {
		return &pool.CalcAmountOutResult{}, errorsPkg.ErrTokenToUnderLyingNotSupported
	}

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
	// check exchange_underlying
	var baseInputIndex = t.BasePool.GetTokenIndex(tokenAmountIn.Token)
	var baseOutputIndex = t.BasePool.GetTokenIndex(tokenOut)
	var maxCoin = len(t.Info.Tokens) - 1
	if tokenIndexFrom < 0 && baseInputIndex >= 0 {
		tokenIndexFrom = maxCoin + baseInputIndex
	}
	if tokenIndexTo < 0 && baseOutputIndex >= 0 {
		tokenIndexTo = maxCoin + baseOutputIndex
	}
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		// get_dy_underlying
		amountOut, fee, err := t.GetDyUnderlying(
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount)
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
				Gas: t.gas.ExchangeUnderlying,
			}, nil

		}
	}
	return &pool.CalcAmountOutResult{
		Gas: t.gas.ExchangeUnderlying,
	}, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var inputIndex = t.GetTokenIndex(input.Token)
	var outputIndex = t.GetTokenIndex(output.Token)
	if inputIndex >= 0 && outputIndex >= 0 {
		// exchange
		_, _ = t.Exchange(inputIndex, outputIndex, inputAmount)
		return
	}
	// check exchange_underlying
	var baseInputIndex = t.BasePool.GetTokenIndex(input.Token)
	var baseOutputIndex = t.BasePool.GetTokenIndex(output.Token)
	var maxCoin = len(t.Info.Tokens) - 1
	if inputIndex < 0 && baseInputIndex >= 0 {
		inputIndex = maxCoin + baseInputIndex
	}
	if outputIndex < 0 && baseOutputIndex >= 0 {
		outputIndex = maxCoin + baseOutputIndex
	}
	if inputIndex >= 0 && outputIndex >= 0 {
		// exchange_underlying
		_, _ = t.ExchangeUnderlying(inputIndex, outputIndex, inputAmount)
	}
}

func (t *Pool) GetLpToken() string {
	return t.LpToken
}

func (t *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		// check from underlying
		tokenIndex = t.BasePool.GetTokenIndex(address)
		if tokenIndex >= 0 {
			var baseLpToken = t.BasePool.GetLpToken()
			for i := 0; i < len(t.Info.Tokens); i += 1 {
				if t.Info.Tokens[i] != baseLpToken {
					ret = append(ret, t.Info.Tokens[i])
				}
			}
		}
		return ret
	}
	// exchange
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	// exchange_underlying
	for i := 0; i < len(t.BasePool.GetInfo().Tokens); i += 1 {
		ret = append(ret, t.BasePool.GetInfo().Tokens[i])
	}
	return ret
}

func (t *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var reserveIn = t.getTokenReserve(tokenIn)
	var reserveOut = t.getTokenReserve(tokenOut)
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}

func (t *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var reserveIn = t.getTokenReserve(tokenIn)
	var reserveOut = t.getTokenReserve(tokenOut)
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}

func (t *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	if fromId >= 0 && toId >= 0 {
		return Meta{
			TokenInIndex:  fromId,
			TokenOutIndex: toId,
			Underlying:    false,
		}
	}
	var baseFromId = t.getUnderlyingIndex(tokenIn)
	var baseToId = t.getUnderlyingIndex(tokenOut)
	return Meta{
		TokenInIndex:  baseFromId,
		TokenOutIndex: baseToId,
		Underlying:    true,
	}
}

func (t *Pool) getUnderlyingIndex(token string) int {
	var tokenIndex = t.GetTokenIndex(token)
	if tokenIndex >= 0 {
		return tokenIndex
	}
	var baseIndex = t.BasePool.GetTokenIndex(token)
	var maxCoin = len(t.Info.Tokens) - 1
	if tokenIndex < 0 && baseIndex >= 0 {
		tokenIndex = maxCoin + baseIndex
	}
	return tokenIndex
}

func (t *Pool) getTokenReserve(token string) *big.Int {
	var tokenIndex = t.GetTokenIndex(token)
	if tokenIndex >= 0 {
		return t.Info.Reserves[tokenIndex]
	}
	var baseIndex = t.BasePool.GetTokenIndex(token)
	if baseIndex >= 0 {
		return t.BasePool.GetInfo().Reserves[baseIndex]
	}
	return nil
}

package meta

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// ICurveBasePool is the interface for curve base pool inside a meta pool
// It can be:
// 1. base/plain pool
// 2. plain oracle pool
// 3. lending pool
// 4. or even meta pool
// At the moment, our code can only support base/plain pool and plain oracle pool
type ICurveBasePool interface {
	pool.IPoolSimulator
	GetInfo() pool.PoolInfo

	// GetVirtualPrice returns both vPrice and D
	GetVirtualPrice() (vPrice *big.Int, D *big.Int, err error)
	// GetDy recalculates `dCached` if it is nil
	GetDy(i int, j int, dx *big.Int, dCached *big.Int) (*big.Int, *big.Int, error)
	CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error)
	CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error)
	AddLiquidity(amounts []*big.Int) (*big.Int, error)
	RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error)
}

type PoolSimulator struct {
	pool.Pool
	basePool       ICurveBasePool
	RateMultiplier *big.Int
	InitialA       *big.Int
	FutureA        *big.Int
	InitialATime   int64
	FutureATime    int64
	AdminFee       *big.Int
	LpToken        string
	LpSupply       *big.Int
	APrecision     *big.Int
	gas            curve.Gas
}

var _ = pool.RegisterFactoryMeta(curve.PoolTypeMeta, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var staticExtra curve.PoolMetaStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	basePool, ok := basePoolMap[staticExtra.BasePool].(ICurveBasePool)
	if !ok {
		return nil, ErrInvalidBasePool
	}

	var extraStr curve.PoolMetaExtra
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

	rateMultiplier := utils.NewBig10(staticExtra.RateMultiplier)
	// Handle a specific case for the RAI Curve-Meta pool,
	// since this pool uses a different contract version, leading the "rates"
	// is calculated using contract data.
	if entityPool.Address == curve.RAIMetaPool {
		rateMultiplier.Set(extraStr.SnappedRedemptionPrice)
		rateMultiplier.Div(rateMultiplier, utils.TenPowInt(9))
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				SwapFee:  utils.NewBig10(extraStr.SwapFee),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		basePool:       basePool,
		RateMultiplier: rateMultiplier,
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

func (t *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	return []pool.IPoolSimulator{t.basePool}
}

func (t *PoolSimulator) SetBasePool(basePool pool.IPoolSimulator) {
	if curveBasePool, ok := basePool.(ICurveBasePool); ok {
		t.basePool = curveBasePool
	}
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)

	if (tokenIndexFrom == len(t.Info.Tokens)-1 && tokenIndexTo < 0) || (tokenIndexTo == len(t.Info.Tokens)-1 && tokenIndexFrom < 0) {
		return &pool.CalcAmountOutResult{}, ErrTokenToUnderLyingNotSupported
	}

	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := t.GetDy(
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
		)
		if err != nil {
			return nil, err
		}
		if amountOut.Cmp(constant.ZeroBI) > 0 {
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
	var baseInputIndex = t.basePool.GetTokenIndex(tokenAmountIn.Token)
	var baseOutputIndex = t.basePool.GetTokenIndex(tokenOut)
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
			return nil, err
		}
		if amountOut.Cmp(constant.ZeroBI) > 0 {
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

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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
	var baseInputIndex = t.basePool.GetTokenIndex(input.Token)
	var baseOutputIndex = t.basePool.GetTokenIndex(output.Token)
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

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		// check from underlying
		tokenIndex = t.basePool.GetTokenIndex(address)
		if tokenIndex >= 0 {
			// base token can be swapped to anything other than the last meta token
			for i := 0; i < len(t.Info.Tokens)-1; i += 1 {
				ret = append(ret, t.Info.Tokens[i])
			}

			// We don't allow swapping between underlying tokens here.
			// Swap between underlying tokens must go directly through the base pool.
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
	// last meta token can't be swapped with underlying tokens
	if tokenIndex != len(t.Info.Tokens)-1 {
		ret = append(ret, t.basePool.GetInfo().Tokens...)
	}
	return ret
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	fromId, toId := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	if fromId >= 0 && toId >= 0 {
		return curve.Meta{
			TokenInIndex:  fromId,
			TokenOutIndex: toId,
			Underlying:    false,
		}
	}
	baseFromId, baseToId := t.getUnderlyingIndex(tokenIn), t.getUnderlyingIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  baseFromId,
		TokenOutIndex: baseToId,
		Underlying:    true,
	}
}

func (t *PoolSimulator) GetTokens() []string {
	var result []string
	result = append(result, t.GetInfo().Tokens...)
	result = append(result, t.basePool.GetInfo().Tokens...)
	return result
}

func (t *PoolSimulator) getUnderlyingIndex(token string) int {
	tokenIndex := t.GetTokenIndex(token)
	if tokenIndex >= 0 {
		return tokenIndex
	}
	baseIndex := t.basePool.GetTokenIndex(token)
	maxCoin := len(t.Info.Tokens) - 1
	if baseIndex >= 0 {
		tokenIndex = maxCoin + baseIndex
	}
	return tokenIndex
}
